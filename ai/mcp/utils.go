package mcp

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func httpContextExtractorWithDebug(ctx context.Context, r *http.Request, debug bool) context.Context {
	if authHeader := r.Header.Get("Authorization"); authHeader != "" {
		ctx = context.WithValue(ctx, "Authorization", authHeader)
	}

	if authToken := r.Header.Get("X-Auth-Token"); authToken != "" {
		ctx = context.WithValue(ctx, "X-Auth-Token", authToken)
	}

	// Extract client IP with better handling
	clientIP := r.RemoteAddr
	if realIP := r.Header.Get("X-Real-IP"); realIP != "" {
		clientIP = realIP
	} else if forwardedFor := r.Header.Get("X-Forwarded-For"); forwardedFor != "" {
		// X-Forwarded-For can contain multiple IPs, take the first one
		if commaIndex := strings.Index(forwardedFor, ","); commaIndex != -1 {
			clientIP = strings.TrimSpace(forwardedFor[:commaIndex])
		} else {
			clientIP = forwardedFor
		}
	}

	// Remove port from IP address if present
	if colonIndex := strings.LastIndex(clientIP, ":"); colonIndex != -1 {
		// Check if this is IPv6 or IPv4:port
		if strings.Count(clientIP, ":") == 1 && !strings.Contains(clientIP, "[") {
			// IPv4:port case
			clientIP = clientIP[:colonIndex]
		}
	}

	ctx = context.WithValue(ctx, "ClientIP", clientIP)

	if debug {
		log.Printf("[DEBUG] Extracted client IP: %s from RemoteAddr: %s", clientIP, r.RemoteAddr)
	}

	if requestID := r.Header.Get("X-Request-ID"); requestID != "" {
		ctx = context.WithValue(ctx, "RequestID", requestID)
	}

	return ctx
}

func authenticationMiddlewareWithConfig(next server.ToolHandlerFunc, jwtConfig *JWTConfig) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		token := extractJWTFromContext(ctx)
		if token == "" {
			return mcp.NewToolResultError("Authentication required: please authenticate first and include the JWT token in the Authorization header"), nil
		}

		var claims JWTClaims
		var user any
		var err error

		if jwtConfig != nil && jwtConfig.Validator != nil {
			claims, err = jwtConfig.Validator.ValidateToken(token)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Invalid or expired token: %v", err)), nil
			}
		} else {
			defaultClaims, err := validateJWTWithClaims(token)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Invalid or expired token: %v", err)), nil
			}
			claims = &DefaultClaims{
				UserID:         defaultClaims.UserID,
				Role:           defaultClaims.Role,
				Email:          defaultClaims.Email,
				StandardClaims: defaultClaims.StandardClaims,
			}
			user = &User{
				ID:    defaultClaims.UserID,
				Email: defaultClaims.Email,
				Role:  defaultClaims.Role,
				Name:  "",
			}
		}

		ctx = context.WithValue(ctx, "jwt_claims", claims)
		if user != nil {
			ctx = context.WithValue(ctx, "user", user)
		}

		return next(ctx, req)
	}
}

// Global rate limiting state
var (
	globalRequestCounts = make(map[string][]time.Time)
	globalRateLimitMu   sync.RWMutex
	cleanupOnce         sync.Once
)

func startRateLimitCleanup() {
	cleanupOnce.Do(func() {
		go func() {
			ticker := time.NewTicker(5 * time.Minute)
			defer ticker.Stop()

			for range ticker.C {
				globalRateLimitMu.Lock()
				now := time.Now()
				for ip, requests := range globalRequestCounts {
					var validRequests []time.Time
					for _, reqTime := range requests {
						if now.Sub(reqTime) < time.Hour {
							validRequests = append(validRequests, reqTime)
						}
					}
					if len(validRequests) == 0 {
						delete(globalRequestCounts, ip)
					} else {
						globalRequestCounts[ip] = validRequests
					}
				}
				globalRateLimitMu.Unlock()
			}
		}()
	})
}

func rateLimitingMiddleware(next server.ToolHandlerFunc, rl RateLimit, debug bool) server.ToolHandlerFunc {
	maxRequests := rl.Limit
	timeWindow := rl.Window

	startRateLimitCleanup()

	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		clientIP := getClientIP(ctx)
		now := time.Now()

		globalRateLimitMu.Lock()
		defer globalRateLimitMu.Unlock()

		// Clean up old requests outside the time window
		if requests, exists := globalRequestCounts[clientIP]; exists {
			var validRequests []time.Time
			for _, reqTime := range requests {
				if now.Sub(reqTime) < timeWindow {
					validRequests = append(validRequests, reqTime)
				}
			}
			globalRequestCounts[clientIP] = validRequests
		}

		currentCount := len(globalRequestCounts[clientIP])

		// Debug logging
		if debug {
			log.Printf("[RATE LIMIT DEBUG] Client IP: %s, Current requests: %d/%d, Window: %v",
				clientIP, currentCount, maxRequests, timeWindow)
		}

		// Check if rate limit is exceeded
		if currentCount >= maxRequests {
			if debug {
				log.Printf("[RATE LIMIT] Rate limit exceeded for IP %s: %d/%d requests",
					clientIP, currentCount, maxRequests)
			}
			return mcp.NewToolResultError(fmt.Sprintf("Rate limit exceeded (%d requests per %v). Please try again later.", maxRequests, timeWindow)), nil
		}

		// Add current request to the count
		globalRequestCounts[clientIP] = append(globalRequestCounts[clientIP], now)

		if debug {
			log.Printf("[RATE LIMIT DEBUG] Request allowed for IP %s, new count: %d/%d",
				clientIP, currentCount+1, maxRequests)
		}

		return next(ctx, req)
	}
}

func loggingMiddleware(next server.ToolHandlerFunc, debug bool) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		start := time.Now()

		if debug {
			log.Printf("Tool call started: %s", req.Params.Name)
		}

		result, err := next(ctx, req)

		duration := time.Since(start)
		if err != nil {
			if debug {
				log.Printf("Tool call failed: %s (duration: %v, error: %v)", req.Params.Name, duration, err)
			} else {
				log.Printf("Tool call failed: %s", req.Params.Name)
			}
		} else {
			if debug {
				log.Printf("Tool call completed: %s (duration: %v)", req.Params.Name, duration)
			}
		}

		return result, err
	}
}

func extractJWTFromContext(ctx context.Context) string {
	if authHeader, ok := ctx.Value("Authorization").(string); ok {
		if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
			return authHeader[7:]
		}
		return authHeader
	}

	if token, ok := ctx.Value("jwt_token").(string); ok {
		return token
	}

	if token, ok := ctx.Value("X-Auth-Token").(string); ok {
		return token
	}

	return ""
}

func validateJWTWithClaims(tokenString string) (*DefaultClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &DefaultClaims{}, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(JWTSecret), nil
	})

	if err != nil {
		return nil, fmt.Errorf("token parsing failed: %w", err)
	}

	claims, ok := token.Claims.(*DefaultClaims)
	if !ok {
		return nil, fmt.Errorf("invalid token claims")
	}

	if !token.Valid {
		return nil, fmt.Errorf("token is invalid")
	}

	if claims.ExpiresAt < time.Now().Unix() {
		return nil, fmt.Errorf("token has expired")
	}

	return claims, nil
}

func getUserFromContext(ctx context.Context) *User {
	if user, ok := ctx.Value("user").(*User); ok {
		return user
	}
	return nil
}

func getClientIP(ctx context.Context) string {
	if ip, ok := ctx.Value("ClientIP").(string); ok && ip != "" {
		return ip
	}
	return "127.0.0.1" // Default to localhost instead of "unknown"
}

// getRateLimitStatus returns current rate limit status for all clients
func getRateLimitStatus(limit int, window time.Duration) map[string]any {
	globalRateLimitMu.RLock()
	defer globalRateLimitMu.RUnlock()

	now := time.Now()
	status := map[string]any{
		"enabled": true,
		"limit":   limit,
		"window":  window.String(),
		"clients": make(map[string]any),
	}

	clients := make(map[string]any)
	for ip, requests := range globalRequestCounts {
		// Count valid requests within the time window
		validCount := 0
		for _, reqTime := range requests {
			if now.Sub(reqTime) < window {
				validCount++
			}
		}

		clients[ip] = map[string]any{
			"current_requests": validCount,
			"limit":            limit,
			"remaining":        limit - validCount,
			"reset_in":         window.String(),
		}
	}

	status["clients"] = clients
	return status
}
