package mcp

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/golang-jwt/jwt"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func httpContextExtractor(ctx context.Context, r *http.Request) context.Context {
	if authHeader := r.Header.Get("Authorization"); authHeader != "" {
		ctx = context.WithValue(ctx, "Authorization", authHeader)
	}

	if authToken := r.Header.Get("X-Auth-Token"); authToken != "" {
		ctx = context.WithValue(ctx, "X-Auth-Token", authToken)
	}

	clientIP := r.RemoteAddr
	if realIP := r.Header.Get("X-Real-IP"); realIP != "" {
		clientIP = realIP
	} else if forwardedFor := r.Header.Get("X-Forwarded-For"); forwardedFor != "" {
		clientIP = forwardedFor
	}
	ctx = context.WithValue(ctx, "ClientIP", clientIP)

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

func rateLimitingMiddleware(next server.ToolHandlerFunc, rl RateLimit) server.ToolHandlerFunc {
	requestCounts := make(map[string][]time.Time)
	var mu sync.RWMutex
	maxRequests := rl.Limit
	timeWindow := rl.Window

	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		clientIP := getClientIP(ctx)
		now := time.Now()

		mu.Lock()
		defer mu.Unlock()

		if requests, exists := requestCounts[clientIP]; exists {
			var validRequests []time.Time
			for _, reqTime := range requests {
				if now.Sub(reqTime) < timeWindow {
					validRequests = append(validRequests, reqTime)
				}
			}
			requestCounts[clientIP] = validRequests
		}

		if len(requestCounts[clientIP]) >= maxRequests {
			return mcp.NewToolResultError(fmt.Sprintf("Rate limit exceeded (%d requests per %v). Please try again later.", maxRequests, timeWindow)), nil
		}

		requestCounts[clientIP] = append(requestCounts[clientIP], now)

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
	return "unknown"
}
