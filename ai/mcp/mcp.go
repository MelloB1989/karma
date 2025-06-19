package mcp

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"reflect"
	"sync"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

type RateLimit struct {
	Limit  int
	Window time.Duration
}

type MiddlewareConfig struct {
	EnableLogging   bool
	EnableRateLimit bool
	EnableAuth      bool
	RateLimit       *RateLimit
	Debug           bool
	JWTConfig       *JWTConfig
	mu              sync.RWMutex
}

type Tool struct {
	Tool    mcp.Tool
	Handler server.ToolHandlerFunc
}

type MCPServer struct {
	Name             string
	Version          string
	Port             int
	Server           *server.MCPServer
	Debug            bool
	Endpoint         string
	MiddlewareConfig *MiddlewareConfig
	httpServer       *server.StreamableHTTPServer
	Tools            []Tool
	mu               sync.RWMutex
}

type MCPOptions func(*MCPServer)

func NewMCPServer(name, ver string, opts ...MCPOptions) *MCPServer {
	newServer := &MCPServer{
		Name:    name,
		Port:    6060, // Default port
		Version: ver,
		MiddlewareConfig: &MiddlewareConfig{
			EnableLogging:   false,
			EnableRateLimit: false,
			EnableAuth:      true, // Default to true for security
			Debug:           false,
			JWTConfig:       NewDefaultJWTConfig(), // Will use default if nil
		},
	}

	for _, opt := range opts {
		opt(newServer)
	}

	newServer.createServer()

	return newServer
}

func (s *MCPServer) createServer() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.MiddlewareConfig.mu.Lock()
	s.MiddlewareConfig.Debug = s.Debug
	s.MiddlewareConfig.mu.Unlock()

	middleware := s.createMiddlewareChain()

	s.Server = server.NewMCPServer(
		s.Name,
		s.Version,
		server.WithToolCapabilities(true),
		server.WithResourceCapabilities(true, true),
		server.WithRecovery(),
		server.WithToolHandlerMiddleware(middleware),
	)

	for _, tool := range s.Tools {
		if tool.Handler != nil {
			s.Server.AddTool(tool.Tool, tool.Handler)
		} else {
			log.Printf("Warning: Tool %v is missing Tool or Handler", tool.Tool.Name)
		}
	}
}

func (s *MCPServer) recreateServer() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.createServer()
	for _, tool := range s.Tools {
		if tool.Handler != nil {
			s.Server.AddTool(tool.Tool, tool.Handler)
		} else {
			log.Printf("Warning: Tool %v is missing Tool or Handler", tool.Tool.Name)
		}
	}
}

func WithLogging(enable bool) MCPOptions {
	return func(s *MCPServer) {
		s.MiddlewareConfig.EnableLogging = enable
	}
}

func WithRateLimit(rl RateLimit) MCPOptions {
	return func(s *MCPServer) {
		s.MiddlewareConfig.EnableRateLimit = true
		s.MiddlewareConfig.RateLimit = &rl
	}
}

func WithAuthentication(enable bool) MCPOptions {
	return func(s *MCPServer) {
		s.MiddlewareConfig.EnableAuth = enable
	}
}

func WithJWTConfig(jwtConfig *JWTConfig) MCPOptions {
	return func(s *MCPServer) {
		s.MiddlewareConfig.JWTConfig = jwtConfig
		s.MiddlewareConfig.EnableAuth = true
	}
}

func WithCustomJWT(claimsType any) MCPOptions {
	return func(s *MCPServer) {
		var reflectType = reflect.TypeOf(claimsType)
		if reflectType.Kind() == reflect.Ptr {
			reflectType = reflectType.Elem()
		}
		s.MiddlewareConfig.JWTConfig = NewCustomJWTConfig(reflectType)
		s.MiddlewareConfig.EnableAuth = true
	}
}

func WithDebug(debug bool) MCPOptions {
	return func(s *MCPServer) {
		s.Debug = debug
		if s.MiddlewareConfig != nil {
			s.MiddlewareConfig.mu.Lock()
			s.MiddlewareConfig.Debug = debug
			s.MiddlewareConfig.EnableLogging = debug
			s.MiddlewareConfig.mu.Unlock()
		}
	}
}

func WithPort(port int) MCPOptions {
	return func(s *MCPServer) {
		s.Port = port
	}
}

func WithEndpoint(endpoint string) MCPOptions {
	return func(s *MCPServer) {
		s.Endpoint = endpoint
	}
}

func WithTools(tools ...Tool) MCPOptions {
	return func(s *MCPServer) {
		s.Tools = tools
	}
}

func (s *MCPServer) EnableLogging(enable bool) {
	s.MiddlewareConfig.mu.Lock()
	changed := s.MiddlewareConfig.EnableLogging != enable
	s.MiddlewareConfig.EnableLogging = enable
	s.MiddlewareConfig.mu.Unlock()

	if changed {
		if s.Debug {
			log.Printf("Logging middleware %s", map[bool]string{true: "enabled", false: "disabled"}[enable])
		}
		s.recreateServer()
	}
}

func (s *MCPServer) EnableRateLimit(rl *RateLimit) {
	s.MiddlewareConfig.mu.Lock()
	var changed bool
	if rl == nil {
		changed = s.MiddlewareConfig.EnableRateLimit
		s.MiddlewareConfig.EnableRateLimit = false
		s.MiddlewareConfig.RateLimit = nil
	} else {
		changed = !s.MiddlewareConfig.EnableRateLimit ||
			s.MiddlewareConfig.RateLimit == nil ||
			*s.MiddlewareConfig.RateLimit != *rl
		s.MiddlewareConfig.EnableRateLimit = true
		s.MiddlewareConfig.RateLimit = rl
	}
	s.MiddlewareConfig.mu.Unlock()

	if changed {
		if s.Debug {
			if rl == nil {
				log.Println("Rate limiting middleware disabled")
			} else {
				log.Printf("Rate limiting middleware enabled: %d requests per %v", rl.Limit, rl.Window)
			}
		}
		s.recreateServer()
	}
}

func (s *MCPServer) EnableAuthentication(enable bool) {
	s.MiddlewareConfig.mu.Lock()
	changed := s.MiddlewareConfig.EnableAuth != enable
	s.MiddlewareConfig.EnableAuth = enable
	s.MiddlewareConfig.mu.Unlock()

	if changed {
		if s.Debug {
			log.Printf("Authentication middleware %s", map[bool]string{true: "enabled", false: "disabled"}[enable])
		}
		s.recreateServer()
	}
}

func (s *MCPServer) SetDebug(debug bool) {
	s.mu.Lock()
	oldDebug := s.Debug
	s.Debug = debug
	s.mu.Unlock()

	s.MiddlewareConfig.mu.Lock()
	s.MiddlewareConfig.Debug = debug
	// Auto-enable/disable logging based on debug setting
	oldLogging := s.MiddlewareConfig.EnableLogging
	s.MiddlewareConfig.EnableLogging = debug
	s.MiddlewareConfig.mu.Unlock()

	if oldDebug != debug || oldLogging != debug {
		if debug {
			log.Println("Debug mode enabled - logging middleware auto-enabled")
		} else {
			log.Println("Debug mode disabled - logging middleware auto-disabled")
		}
		s.recreateServer()
	}
}

func (s *MCPServer) createMiddlewareChain() server.ToolHandlerMiddleware {
	return func(next server.ToolHandlerFunc) server.ToolHandlerFunc {
		handler := next

		s.MiddlewareConfig.mu.RLock()
		config := MiddlewareConfig{
			EnableLogging:   s.MiddlewareConfig.EnableLogging,
			EnableRateLimit: s.MiddlewareConfig.EnableRateLimit,
			EnableAuth:      s.MiddlewareConfig.EnableAuth,
			RateLimit:       s.MiddlewareConfig.RateLimit,
			Debug:           s.MiddlewareConfig.Debug,
			JWTConfig:       s.MiddlewareConfig.JWTConfig,
		}
		s.MiddlewareConfig.mu.RUnlock()

		// Apply middleware in reverse order (last middleware wraps first)
		if config.EnableAuth {
			handler = authenticationMiddlewareWithConfig(handler, config.JWTConfig)
			if config.Debug {
				if config.JWTConfig != nil {
					log.Println("Authentication middleware applied with custom JWT config")
				} else {
					log.Println("Authentication middleware applied with default config")
				}
			}
		}

		if config.EnableRateLimit && config.RateLimit != nil {
			handler = rateLimitingMiddleware(handler, *config.RateLimit, config.Debug)
			if config.Debug {
				log.Printf("Rate limiting middleware applied: %d requests per %v",
					config.RateLimit.Limit, config.RateLimit.Window)
			}
		}

		if config.EnableLogging {
			handler = loggingMiddleware(handler, config.Debug)
			if config.Debug {
				log.Println("Logging middleware applied")
			}
		}

		return handler
	}
}

func (s *MCPServer) Start() error {
	s.mu.RLock()
	endpoint := s.Endpoint
	if endpoint == "" {
		endpoint = "mcp"
	}
	port := s.Port
	debug := s.Debug
	s.mu.RUnlock()

	if debug {
		log.Printf("Starting Advanced StreamableHTTP server on :%d", port)
		log.Println("Endpoints available:")
		log.Printf("  POST /%s - MCP requests", endpoint)
		log.Printf("  GET  /%s - MCP stream connection", endpoint)

		// Log middleware status
		s.MiddlewareConfig.mu.RLock()
		log.Printf("Middleware status:")
		log.Printf("  - Logging: %v", s.MiddlewareConfig.EnableLogging)
		log.Printf("  - Rate Limiting: %v", s.MiddlewareConfig.EnableRateLimit)
		if s.MiddlewareConfig.EnableRateLimit && s.MiddlewareConfig.RateLimit != nil {
			log.Printf("    Rate Limit: %d requests per %v",
				s.MiddlewareConfig.RateLimit.Limit, s.MiddlewareConfig.RateLimit.Window)
		}
		log.Printf("  - Authentication: %v", s.MiddlewareConfig.EnableAuth)
		s.MiddlewareConfig.mu.RUnlock()
	}

	s.mu.Lock()

	contextExtractor := func(ctx context.Context, r *http.Request) context.Context {
		return httpContextExtractorWithDebug(ctx, r, debug)
	}

	s.httpServer = server.NewStreamableHTTPServer(s.Server,
		server.WithEndpointPath("/"+endpoint),
		server.WithHeartbeatInterval(30*time.Second),
		server.WithStateLess(true),
		server.WithHTTPContextFunc(contextExtractor),
	)
	s.mu.Unlock()

	if err := s.httpServer.Start(fmt.Sprintf(":%d", port)); err != nil {
		if debug {
			log.Printf("Error starting MCP server: %v", err)
		}
		return fmt.Errorf("failed to start MCP server: %w", err)
	}
	return nil
}

func (s *MCPServer) GetMiddlewareStatus() map[string]any {
	s.MiddlewareConfig.mu.RLock()
	defer s.MiddlewareConfig.mu.RUnlock()

	status := map[string]any{
		"logging":        s.MiddlewareConfig.EnableLogging,
		"authentication": s.MiddlewareConfig.EnableAuth,
		"rate_limiting":  s.MiddlewareConfig.EnableRateLimit,
		"debug":          s.MiddlewareConfig.Debug,
		"jwt_config":     s.MiddlewareConfig.JWTConfig != nil,
	}

	if s.MiddlewareConfig.EnableRateLimit && s.MiddlewareConfig.RateLimit != nil {
		status["rate_limit_config"] = map[string]any{
			"limit":  s.MiddlewareConfig.RateLimit.Limit,
			"window": s.MiddlewareConfig.RateLimit.Window.String(),
		}
	}

	if s.MiddlewareConfig.JWTConfig != nil {
		status["jwt_claims_type"] = s.MiddlewareConfig.JWTConfig.ClaimsType.String()
	}

	return status
}

// SetJWTConfig allows dynamic JWT configuration changes
func (s *MCPServer) SetJWTConfig(jwtConfig *JWTConfig) {
	s.MiddlewareConfig.mu.Lock()
	changed := s.MiddlewareConfig.JWTConfig != jwtConfig
	s.MiddlewareConfig.JWTConfig = jwtConfig
	if jwtConfig != nil {
		s.MiddlewareConfig.EnableAuth = true
	}
	s.MiddlewareConfig.mu.Unlock()

	if changed {
		if s.Debug {
			if jwtConfig != nil {
				log.Printf("JWT configuration updated with claims type: %v", jwtConfig.ClaimsType)
			} else {
				log.Println("JWT configuration cleared - using default")
			}
		}
		s.recreateServer()
	}
}

func (s *MCPServer) GetRateLimitStatus() map[string]any {
	s.MiddlewareConfig.mu.RLock()
	if !s.MiddlewareConfig.EnableRateLimit || s.MiddlewareConfig.RateLimit == nil {
		s.MiddlewareConfig.mu.RUnlock()
		return map[string]any{
			"enabled": false,
		}
	}

	limit := s.MiddlewareConfig.RateLimit.Limit
	window := s.MiddlewareConfig.RateLimit.Window
	s.MiddlewareConfig.mu.RUnlock()

	// Get current request counts from utils.go
	return getRateLimitStatus(limit, window)
}
