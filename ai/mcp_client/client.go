package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"time"

	"github.com/invopop/jsonschema"
)

// Client represents an MCP (Model Context Protocol) client
type Client struct {
	ServerURL  string
	AuthToken  string
	HTTPClient *http.Client
}

// Tool represents an MCP tool that can be called
type Tool struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"inputSchema"`
	MCPToolName string         `json:"mcpToolName"` // The actual tool name in MCP server
}

// Request represents an MCP JSON-RPC request
type Request struct {
	JSONRPC string `json:"jsonrpc"`
	ID      int    `json:"id"`
	Method  string `json:"method"`
	Params  any    `json:"params,omitempty"`
}

// Response represents an MCP JSON-RPC response
type Response struct {
	JSONRPC string `json:"jsonrpc"`
	ID      int    `json:"id"`
	Result  any    `json:"result,omitempty"`
	Error   *Error `json:"error,omitempty"`
}

// Error represents an MCP error
type Error struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

// CallToolParams represents MCP tool call parameters
type CallToolParams struct {
	Name      string         `json:"name"`
	Arguments map[string]any `json:"arguments,omitempty"`
}

// ToolResult represents the result of an MCP tool call
type ToolResult struct {
	Content   string `json:"content"`
	IsError   bool   `json:"isError"`
	ErrorCode int    `json:"errorCode,omitempty"`
}

// NewClient creates a new MCP client with the given server URL and optional auth token
func NewClient(serverURL, authToken string) *Client {
	return &Client{
		ServerURL:  serverURL,
		AuthToken:  authToken,
		HTTPClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// SetTimeout sets the HTTP client timeout for MCP requests
func (c *Client) SetTimeout(timeout time.Duration) {
	c.HTTPClient.Timeout = timeout
}

// CreateTool creates a new MCP tool with schema generation from a Go struct
func (c *Client) CreateTool(name, description, mcpToolName string, inputSchema any) (*Tool, error) {
	schema, err := c.generateSchema(inputSchema)
	if err != nil {
		return nil, fmt.Errorf("failed to generate schema for tool %s: %w", name, err)
	}

	return &Tool{
		Name:        name,
		Description: description,
		InputSchema: schema,
		MCPToolName: mcpToolName,
	}, nil
}

// generateSchema generates JSON schema from a Go struct
func (c *Client) generateSchema(inputStruct any) (map[string]any, error) {
	reflector := jsonschema.Reflector{
		AllowAdditionalProperties: false,
		DoNotReference:            true,
	}

	// Handle nil input: return an empty object schema to avoid panic
	if inputStruct == nil || reflect.TypeOf(inputStruct) == nil {
		return map[string]any{
			"type":                 "object",
			"properties":           map[string]any{},
			"additionalProperties": false,
		}, nil
	}

	var schema *jsonschema.Schema
	// Reflect from type to avoid nil pointer dereference on typed nils
	schema = reflector.ReflectFromType(reflect.TypeOf(inputStruct))

	// Convert to map[string]any for easier handling
	schemaBytes, err := json.Marshal(schema)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal schema: %w", err)
	}

	var schemaMap map[string]any
	err = json.Unmarshal(schemaBytes, &schemaMap)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal schema: %w", err)
	}

	return schemaMap, nil
}

// CallTool calls an MCP tool with the given arguments and returns the result
func (c *Client) CallTool(ctx context.Context, tool *Tool, arguments map[string]any) (*ToolResult, error) {
	if c.ServerURL == "" {
		return nil, fmt.Errorf("MCP server URL not configured")
	}

	// Create MCP request
	request := Request{
		JSONRPC: "2.0",
		ID:      int(time.Now().Unix()),
		Method:  "tools/call",
		Params: CallToolParams{
			Name:      tool.MCPToolName,
			Arguments: arguments,
		},
	}

	// Send request to MCP server
	result, err := c.sendRequest(ctx, request)
	if err != nil {
		return &ToolResult{
			Content:   fmt.Sprintf("Error calling tool: %v", err),
			IsError:   true,
			ErrorCode: 500,
		}, nil
	}

	return result, nil
}

// CallToolByName calls an MCP tool by its MCP tool name directly
func (c *Client) CallToolByName(ctx context.Context, mcpToolName string, arguments map[string]any) (*ToolResult, error) {
	if c.ServerURL == "" {
		return nil, fmt.Errorf("MCP server URL not configured")
	}

	// Create MCP request
	request := Request{
		JSONRPC: "2.0",
		ID:      int(time.Now().Unix()),
		Method:  "tools/call",
		Params: CallToolParams{
			Name:      mcpToolName,
			Arguments: arguments,
		},
	}

	// Send request to MCP server
	return c.sendRequest(ctx, request)
}

// sendRequest sends an HTTP request to the MCP server and handles the response
func (c *Client) sendRequest(ctx context.Context, request Request) (*ToolResult, error) {
	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal MCP request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.ServerURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if c.AuthToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.AuthToken)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send MCP request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("MCP server returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read MCP response: %w", err)
	}

	var mcpResp Response
	err = json.Unmarshal(body, &mcpResp)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal MCP response: %w", err)
	}

	if mcpResp.Error != nil {
		return &ToolResult{
			Content:   mcpResp.Error.Message,
			IsError:   true,
			ErrorCode: mcpResp.Error.Code,
		}, nil
	}

	// Extract content from MCP response
	content := c.extractContentFromResult(mcpResp.Result)
	return &ToolResult{
		Content: content,
		IsError: false,
	}, nil
}

// extractContentFromResult extracts text content from MCP result
func (c *Client) extractContentFromResult(result any) string {
	if result == nil {
		return ""
	}

	// Try to extract from standard MCP response format
	resultMap, ok := result.(map[string]any)
	if !ok {
		return fmt.Sprintf("%v", result)
	}

	// Check for content array
	content, ok := resultMap["content"].([]any)
	if !ok || len(content) == 0 {
		return fmt.Sprintf("%v", result)
	}

	// Extract text from first content item
	textContent, ok := content[0].(map[string]any)
	if !ok {
		return fmt.Sprintf("%v", content[0])
	}

	text, ok := textContent["text"].(string)
	if !ok {
		return fmt.Sprintf("%v", textContent)
	}

	return text
}

// Ping sends a ping request to the MCP server to check connectivity
func (c *Client) Ping(ctx context.Context) error {
	request := Request{
		JSONRPC: "2.0",
		ID:      int(time.Now().Unix()),
		Method:  "ping",
	}

	_, err := c.sendRequest(ctx, request)
	return err
}

// ListTools lists available tools from the MCP server
func (c *Client) ListTools(ctx context.Context) ([]map[string]any, error) {
	request := Request{
		JSONRPC: "2.0",
		ID:      int(time.Now().Unix()),
		Method:  "tools/list",
	}

	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal MCP request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.ServerURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if c.AuthToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.AuthToken)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send MCP request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read MCP response: %w", err)
	}

	var mcpResp Response
	err = json.Unmarshal(body, &mcpResp)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal MCP response: %w", err)
	}

	if mcpResp.Error != nil {
		return nil, fmt.Errorf("MCP error %d: %s", mcpResp.Error.Code, mcpResp.Error.Message)
	}

	// Extract tools from result
	resultMap, ok := mcpResp.Result.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("unexpected result format")
	}

	tools, ok := resultMap["tools"].([]any)
	if !ok {
		return nil, fmt.Errorf("no tools found in response")
	}

	var toolsList []map[string]any
	for _, tool := range tools {
		if toolMap, ok := tool.(map[string]any); ok {
			toolsList = append(toolsList, toolMap)
		}
	}

	return toolsList, nil
}
