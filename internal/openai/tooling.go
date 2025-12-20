package openai

import (
	"context"
	"fmt"

	mcp "github.com/MelloB1989/karma/ai/mcp_client"
	"github.com/MelloB1989/karma/models"
	"github.com/openai/openai-go/v3"
)

const defaultMaxToolPasses = 5

type GoFunctionTool struct {
	Name        string
	Description string
	Parameters  openai.FunctionParameters
	Strict      bool
	Handler     func(context.Context, map[string]any) (string, error)
}

func (o *OpenAI) SetMCPServer(serverURL string, authToken string) {
	mcpClient := mcp.NewClient(serverURL, authToken)
	o.MCPManager = mcp.NewManager(mcpClient)
}

func (o *OpenAI) SetMultiMCPManager(multiManager *mcp.MultiManager) {
	o.MultiMCPManager = multiManager
}

func (o *OpenAI) SetMaxToolPasses(max int) {
	o.maxToolPasses = max
}

func (o *OpenAI) AddMCPTool(name, description, mcpToolName string, inputSchema any) error {
	if o.MCPManager == nil {
		return fmt.Errorf("MCP server not configured. Call SetMCPServer first")
	}
	return o.MCPManager.AddToolFromSchema(name, description, mcpToolName, inputSchema)
}

func (o *OpenAI) AddGoFunctionTool(tool GoFunctionTool) error {
	if tool.Name == "" {
		return fmt.Errorf("tool name required")
	}
	if tool.Handler == nil {
		return fmt.Errorf("tool handler required")
	}
	tool.Parameters = coerceFunctionParameters(tool.Parameters)
	o.FunctionTools[tool.Name] = tool
	return nil
}

func (o *OpenAI) AddGoFunctionDefinition(def models.OpenAIFunctionDefinition, handler func(context.Context, map[string]any) (string, error)) error {
	tool := GoFunctionTool{
		Name:        def.Name,
		Description: def.Description,
		Parameters:  coerceFunctionParameters(def.Parameters),
		Strict:      def.Strict,
		Handler:     handler,
	}
	return o.AddGoFunctionTool(tool)
}

func (o *OpenAI) ClearGoFunctionTools() {
	o.FunctionTools = make(map[string]GoFunctionTool)
}

func (o *OpenAI) GetMCPManager() *mcp.Manager {
	return o.MCPManager
}
