package mcp

import (
	"context"
	"time"
)

// MCPClient defines the interface for MCP (Model Context Protocol) clients
type MCPClient interface {
	// SetTimeout sets the HTTP client timeout for MCP requests
	SetTimeout(timeout time.Duration)

	// CreateTool creates a new MCP tool with schema generation from a Go struct
	CreateTool(name, description, mcpToolName string, inputSchema any) (*Tool, error)

	// CallTool calls an MCP tool with the given arguments and returns the result
	CallTool(ctx context.Context, tool *Tool, arguments map[string]any) (*ToolResult, error)

	// CallToolByName calls an MCP tool by its MCP tool name directly
	CallToolByName(ctx context.Context, mcpToolName string, arguments map[string]any) (*ToolResult, error)

	// Ping sends a ping request to the MCP server to check connectivity
	Ping(ctx context.Context) error

	// ListTools lists available tools from the MCP server
	ListTools(ctx context.Context) ([]map[string]any, error)
}

// ToolManager defines the interface for managing MCP tools
type ToolManager interface {
	// AddTool adds a tool to the managed collection
	AddTool(tool *Tool) error

	// RemoveTool removes a tool from the managed collection
	RemoveTool(name string) error

	// GetTool retrieves a tool by name
	GetTool(name string) (*Tool, bool)

	// GetAllTools returns all managed tools
	GetAllTools() []*Tool

	// CallTool calls a managed tool by name
	CallTool(ctx context.Context, name string, arguments map[string]any) (*ToolResult, error)
}

// Ensure Client implements MCPClient interface
var _ MCPClient = (*Client)(nil)
