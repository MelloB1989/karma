package mcp

import (
	"context"
	"fmt"
	"sync"
)

type MultiManager struct {
	managers map[string]*Manager
	tools    map[string]string
	mutex    sync.RWMutex
}

func NewMultiManager() *MultiManager {
	return &MultiManager{
		managers: make(map[string]*Manager),
		tools:    make(map[string]string),
	}
}

func (mm *MultiManager) AddServer(serverID, serverURL, authToken string) {
	mm.mutex.Lock()
	defer mm.mutex.Unlock()

	client := NewClient(serverURL, authToken)
	manager := NewManager(client)
	mm.managers[serverID] = manager
}

func (mm *MultiManager) AddToolToServer(serverID, name, description, mcpToolName string, inputSchema any) error {
	mm.mutex.Lock()
	defer mm.mutex.Unlock()

	manager, exists := mm.managers[serverID]
	if !exists {
		return fmt.Errorf("server %s not found", serverID)
	}

	err := manager.AddToolFromSchema(name, description, mcpToolName, inputSchema)
	if err != nil {
		return err
	}

	mm.tools[name] = serverID
	return nil
}

func (mm *MultiManager) GetAllTools() []*Tool {
	mm.mutex.RLock()
	defer mm.mutex.RUnlock()

	var allTools []*Tool
	for _, manager := range mm.managers {
		tools := manager.GetAllTools()
		allTools = append(allTools, tools...)
	}
	return allTools
}

func (mm *MultiManager) CallTool(ctx context.Context, toolName string, arguments map[string]any) (*ToolResult, error) {
	mm.mutex.RLock()
	serverID, exists := mm.tools[toolName]
	if !exists {
		mm.mutex.RUnlock()
		return nil, fmt.Errorf("tool %s not found", toolName)
	}

	manager, exists := mm.managers[serverID]
	if !exists {
		mm.mutex.RUnlock()
		return nil, fmt.Errorf("server %s not found for tool %s", serverID, toolName)
	}
	mm.mutex.RUnlock()

	return manager.CallTool(ctx, toolName, arguments)
}

func (mm *MultiManager) Count() int {
	mm.mutex.RLock()
	defer mm.mutex.RUnlock()

	return len(mm.tools)
}

func (mm *MultiManager) HasTool(name string) bool {
	mm.mutex.RLock()
	defer mm.mutex.RUnlock()

	_, exists := mm.tools[name]
	return exists
}

func (mm *MultiManager) GetTool(name string) (*Tool, bool) {
	mm.mutex.RLock()
	defer mm.mutex.RUnlock()

	serverID, exists := mm.tools[name]
	if !exists {
		return nil, false
	}

	manager, exists := mm.managers[serverID]
	if !exists {
		return nil, false
	}

	return manager.GetTool(name)
}

func (mm *MultiManager) RemoveServer(serverID string) error {
	mm.mutex.Lock()
	defer mm.mutex.Unlock()

	_, exists := mm.managers[serverID]
	if !exists {
		return fmt.Errorf("server %s not found", serverID)
	}

	for toolName, sid := range mm.tools {
		if sid == serverID {
			delete(mm.tools, toolName)
		}
	}

	delete(mm.managers, serverID)
	return nil
}

func (mm *MultiManager) Clear() {
	mm.mutex.Lock()
	defer mm.mutex.Unlock()

	mm.managers = make(map[string]*Manager)
	mm.tools = make(map[string]string)
}
