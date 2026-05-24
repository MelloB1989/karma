package claude

import (
	"context"
	"fmt"

	mcp "github.com/MelloB1989/karma/ai/mcp_client"
	"github.com/MelloB1989/karma/models"
	"github.com/anthropics/anthropic-sdk-go"
)

func (cc *ClaudeClient) AddGoFunctionTool(tool GoFunctionTool) error {
	if tool.Name == "" {
		return fmt.Errorf("tool name required")
	}
	if tool.Handler == nil {
		return fmt.Errorf("tool handler required")
	}
	cc.FunctionTools[tool.Name] = tool
	return nil
}

func (cc *ClaudeClient) hasGoFunctionTools() bool {
	return len(cc.FunctionTools) > 0
}

func (cc *ClaudeClient) hasAnyTools() bool {
	return cc.hasMCPTools() || cc.hasGoFunctionTools()
}

func (cc *ClaudeClient) convertGoFunctionToolsToAnthropic() []anthropic.ToolUnionParam {
	tools := make([]anthropic.ToolUnionParam, 0, len(cc.FunctionTools))
	for _, tool := range cc.FunctionTools {
		inputSchema := anthropic.ToolInputSchemaParam{}
		if props, ok := tool.Parameters["properties"].(map[string]any); ok {
			inputSchema.Properties = props
		}
		if required, ok := tool.Parameters["required"]; ok {
			inputSchema.ExtraFields = map[string]any{"required": required}
		}
		tools = append(tools, anthropic.ToolUnionParam{
			OfTool: &anthropic.ToolParam{
				Name:        tool.Name,
				Description: anthropic.String(tool.Description),
				InputSchema: inputSchema,
			},
		})
	}
	return tools
}

func (cc *ClaudeClient) getAllToolsAsAnthropic() []anthropic.ToolUnionParam {
	var tools []anthropic.ToolUnionParam
	if cc.hasMCPTools() {
		tools = append(tools, cc.convertMCPToolsToAnthropic()...)
	}
	if cc.hasGoFunctionTools() {
		tools = append(tools, cc.convertGoFunctionToolsToAnthropic()...)
	}
	return tools
}

func (cc *ClaudeClient) callTool(ctx context.Context, toolName string, arguments map[string]any) (string, error) {
	if fnTool, ok := cc.FunctionTools[toolName]; ok {
		return fnTool.Handler(ctx, arguments)
	}
	return cc.callMCPTool(ctx, toolName, arguments)
}

// hasMCPTools checks if any MCP tools are available
func (cc *ClaudeClient) hasMCPTools() bool {
	if cc.MultiMCPManager != nil {
		return cc.MultiMCPManager.Count() > 0
	}
	return cc.MCPManager != nil && cc.MCPManager.Count() > 0
}

// convertMCPToolsToAnthropic converts MCP tools to Anthropic tool format
func (cc *ClaudeClient) convertMCPToolsToAnthropic() []anthropic.ToolUnionParam {
	if !cc.hasMCPTools() {
		return nil
	}

	var mcpTools []*mcp.Tool
	if cc.MultiMCPManager != nil {
		mcpTools = cc.MultiMCPManager.GetAllTools()
	} else {
		mcpTools = cc.MCPManager.GetAllTools()
	}
	tools := make([]anthropic.ToolUnionParam, len(mcpTools))

	for i, mcpTool := range mcpTools {
		// Convert MCP tool schema to Anthropic format
		inputSchema := anthropic.ToolInputSchemaParam{}

		// Extract properties from the MCP tool schema
		if properties, ok := mcpTool.InputSchema["properties"].(map[string]any); ok {
			inputSchema.Properties = properties
		}

		toolParam := anthropic.ToolParam{
			Name:        mcpTool.Name,
			Description: anthropic.String(mcpTool.Description),
			InputSchema: inputSchema,
		}
		tools[i] = anthropic.ToolUnionParam{OfTool: &toolParam}
	}

	return tools
}

// callMCPTool calls an MCP tool and returns the result
func (cc *ClaudeClient) callMCPTool(ctx context.Context, toolName string, arguments map[string]any) (string, error) {
	var result *mcp.ToolResult
	var err error

	if cc.MultiMCPManager != nil {
		result, err = cc.MultiMCPManager.CallTool(ctx, toolName, arguments)
	} else if cc.MCPManager != nil {
		result, err = cc.MCPManager.CallTool(ctx, toolName, arguments)
	} else {
		return "", fmt.Errorf("MCP server not configured")
	}

	if err != nil {
		return "", err
	}

	if result.IsError {
		return "", fmt.Errorf("MCP tool error (%d): %s", result.ErrorCode, result.Content)
	}

	return result.Content, nil
}

func processMessages(messages models.AIChatHistory) []anthropic.MessageParam {
	var processedMessages []anthropic.MessageParam
	for _, msg := range messages.Messages {
		var role anthropic.MessageParamRole
		if msg.Role == models.User {
			role = anthropic.MessageParamRoleUser
		} else {
			role = anthropic.MessageParamRoleAssistant
		}
		processedMessages = append(processedMessages, anthropic.MessageParam{
			Role: role,
			Content: []anthropic.ContentBlockParamUnion{{
				OfText: &anthropic.TextBlockParam{Text: msg.Message},
			}},
		})
	}
	return processedMessages
}
