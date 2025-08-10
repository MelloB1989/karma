package mcp

import (
	"context"
	"fmt"
	"sync"
)

// Manager implements the ToolManager interface for managing MCP tools
type Manager struct {
	client MCPClient
	tools  map[string]*Tool
	mutex  sync.RWMutex
}

// NewManager creates a new tool manager with the given MCP client
func NewManager(client MCPClient) *Manager {
	return &Manager{
		client: client,
		tools:  make(map[string]*Tool),
	}
}

// AddTool adds a tool to the managed collection
func (m *Manager) AddTool(tool *Tool) error {
	if tool == nil {
		return fmt.Errorf("tool cannot be nil")
	}
	if tool.Name == "" {
		return fmt.Errorf("tool name cannot be empty")
	}

	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.tools[tool.Name] = tool
	return nil
}

// RemoveTool removes a tool from the managed collection
func (m *Manager) RemoveTool(name string) error {
	if name == "" {
		return fmt.Errorf("tool name cannot be empty")
	}

	m.mutex.Lock()
	defer m.mutex.Unlock()

	if _, exists := m.tools[name]; !exists {
		return fmt.Errorf("tool %s not found", name)
	}

	delete(m.tools, name)
	return nil
}

// GetTool retrieves a tool by name
func (m *Manager) GetTool(name string) (*Tool, bool) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	tool, exists := m.tools[name]
	return tool, exists
}

// GetAllTools returns all managed tools
func (m *Manager) GetAllTools() []*Tool {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	tools := make([]*Tool, 0, len(m.tools))
	for _, tool := range m.tools {
		tools = append(tools, tool)
	}
	return tools
}

// CallTool calls a managed tool by name
func (m *Manager) CallTool(ctx context.Context, name string, arguments map[string]any) (*ToolResult, error) {
	tool, exists := m.GetTool(name)
	if !exists {
		return nil, fmt.Errorf("tool %s not found", name)
	}

	return m.client.CallTool(ctx, tool, arguments)
}

// AddToolFromSchema creates and adds a tool from schema parameters
func (m *Manager) AddToolFromSchema(name, description, mcpToolName string, inputSchema any) error {
	tool, err := m.client.CreateTool(name, description, mcpToolName, inputSchema)
	if err != nil {
		return fmt.Errorf("failed to create tool: %w", err)
	}

	return m.AddTool(tool)
}

// GetToolNames returns a slice of all tool names
func (m *Manager) GetToolNames() []string {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	names := make([]string, 0, len(m.tools))
	for name := range m.tools {
		names = append(names, name)
	}
	return names
}

// HasTool checks if a tool with the given name exists
func (m *Manager) HasTool(name string) bool {
	_, exists := m.GetTool(name)
	return exists
}

// Clear removes all tools from the manager
func (m *Manager) Clear() {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.tools = make(map[string]*Tool)
}

// Count returns the number of tools managed
func (m *Manager) Count() int {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	return len(m.tools)
}

// Ensure Manager implements ToolManager interface
var _ ToolManager = (*Manager)(nil)
