package openai

import (
	"context"
	"crypto/sha256"
	"fmt"
	"log"

	mcp "github.com/MelloB1989/karma/ai/mcp_client"
	"github.com/MelloB1989/karma/config"
	"github.com/MelloB1989/karma/models"
	"github.com/openai/openai-go/v2"
	"github.com/openai/openai-go/v2/option"
)

type CompatibleOptions struct {
	BaseURL string
	API_Key string
}

func createClient(opts ...CompatibleOptions) openai.Client {
	if len(opts) > 0 {
		return openai.NewClient(option.WithAPIKey(opts[0].API_Key), option.WithBaseURL(opts[0].BaseURL))
	}
	return openai.NewClient(option.WithAPIKey(config.DefaultConfig().OPENAI_KEY))
}

func formatMessages(messages models.AIChatHistory, sysmgs string, supportsVision bool) []openai.ChatCompletionMessageParamUnion {
	mgs := []openai.ChatCompletionMessageParamUnion{}
	mgs = append(mgs, openai.SystemMessage(sysmgs))

	for _, message := range messages.Messages {
		switch message.Role {
		case "user":
			if len(message.Images) > 0 {
				if supportsVision {
					// Create content parts for text and images
					content := []openai.ChatCompletionContentPartUnionParam{
						openai.TextContentPart(message.Message),
					}

					// Add image content parts
					for _, image := range message.Images {
						imageContent := openai.ImageContentPart(openai.ChatCompletionContentPartImageImageURLParam{
							URL: image,
						})
						content = append(content, imageContent)
					}

					// Create user message with mixed content
					mgs = append(mgs, openai.UserMessage(content))
				} else {
					// Vision not supported - log warning and ignore images
					log.Printf("Warning: Model does not support vision capabilities. Ignoring %d image(s) in message.", len(message.Images))
					// Simple text-only user message
					mgs = append(mgs, openai.UserMessage(message.Message))
				}
			} else {
				// Simple text-only user message
				mgs = append(mgs, openai.UserMessage(message.Message))
			}
		case "assistant":
			mgs = append(mgs, openai.AssistantMessage(message.Message))
		case "system":
			mgs = append(mgs, openai.SystemMessage(message.Message))
		}
	}
	return mgs
}

func (o *OpenAI) hasMCPTools() bool {
	if o.MultiMCPManager != nil {
		return o.MultiMCPManager.Count() > 0
	}
	return o.MCPManager != nil && o.MCPManager.Count() > 0
}

// convertMCPToolsToOpenAI converts MCP tools to OpenAI tool format
func (o *OpenAI) convertMCPToolsToOpenAI() []openai.ChatCompletionToolUnionParam {
	if !o.hasMCPTools() {
		return nil
	}

	var mcpTools []*mcp.Tool
	if o.MultiMCPManager != nil {
		mcpTools = o.MultiMCPManager.GetAllTools()
	} else {
		mcpTools = o.MCPManager.GetAllTools()
	}
	tools := make([]openai.ChatCompletionToolUnionParam, len(mcpTools))

	for i, mcpTool := range mcpTools {
		// Convert MCP tool schema to OpenAI format
		parameters := openai.FunctionParameters{
			"type": "object",
		}

		// Extract properties from the MCP tool schema
		if properties, ok := mcpTool.InputSchema["properties"].(map[string]any); ok {
			parameters["properties"] = properties
		}

		// Extract required fields
		if required, ok := mcpTool.InputSchema["required"].([]any); ok {
			parameters["required"] = required
		}

		tools[i] = openai.ChatCompletionFunctionTool(openai.FunctionDefinitionParam{
			Name:        mcpTool.Name,
			Description: openai.String(mcpTool.Description),
			Parameters:  parameters,
		})
	}

	return tools
}

// callMCPTool calls an MCP tool and returns the result
func (o *OpenAI) callMCPTool(ctx context.Context, toolName string, arguments map[string]any) (string, error) {
	var result *mcp.ToolResult
	var err error

	if o.MultiMCPManager != nil {
		result, err = o.MultiMCPManager.CallTool(ctx, toolName, arguments)
	} else if o.MCPManager != nil {
		result, err = o.MCPManager.CallTool(ctx, toolName, arguments)
	} else {
		return "", fmt.Errorf("MCP server not configured")
	}

	if err != nil {
		return "", err
	}

	if result.IsError {
		return "", fmt.Errorf("MCP tool error %d: %s", result.ErrorCode, result.Content)
	}

	return result.Content, nil
}

// generateShortToolCallID creates a short, unique tool call ID from a longer MCP ID
// OpenAI requires tool call IDs to be max 40 characters
func generateShortToolCallID(originalID string) string {
	if len(originalID) <= 40 {
		return originalID
	}

	// Use first 8 chars + hash of remaining to ensure uniqueness within 40 char limit
	hash := sha256.Sum256([]byte(originalID))
	hashStr := fmt.Sprintf("%x", hash)[:24] // Take first 24 chars of hash
	prefix := originalID[:8]                // Take first 8 chars of original
	return prefix + "_" + hashStr[:23]      // Total: 8 + 1 + 23 = 32 chars (well under 40)
}
