// Package mcp provides a client implementation for the Model Context Protocol (MCP).
//
// MCP is a standardized protocol for AI models to interact with external tools and resources.
// This package provides a clean, reusable interface for calling MCP tools from any AI model
// implementation in the Karma framework.
//
// Basic Usage:
//
//	// Create a new MCP client
//	client := mcp.NewClient("http://localhost:3000", "your-auth-token")
//
//	// Create a tool manager
//	manager := mcp.NewManager(client)
//
//	// Define a tool schema
//	type FileReadParams struct {
//		Path string `json:"path" jsonschema:"required"`
//	}
//
//	// Add a tool
//	err := manager.AddToolFromSchema(
//		"read_file",
//		"Read contents of a file",
//		"file_read",
//		FileReadParams{},
//	)
//
//	// Call the tool
//	result, err := manager.CallTool(context.Background(), "read_file", map[string]any{
//		"path": "/path/to/file.txt",
//	})
//
// The package supports:
//   - Automatic JSON schema generation from Go structs
//   - Concurrent tool management with proper synchronization
//   - Context-aware HTTP requests with configurable timeouts
//   - Comprehensive error handling and response parsing
//   - Interface-based design for easy testing and mocking
//
// Integration with AI Models:
//
// This package is designed to be used by AI model implementations (Claude, Gemini, etc.)
// to provide tool calling capabilities. The clean interface allows each model to integrate
// MCP tools without duplicating the protocol implementation.
package mcp
