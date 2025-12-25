package openai

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"strings"

	mcp "github.com/MelloB1989/karma/ai/mcp_client"
	"github.com/MelloB1989/karma/config"
	"github.com/MelloB1989/karma/models"
	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
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

func formatMessages(messages models.AIChatHistory, sysmgs string) []openai.ChatCompletionMessageParamUnion {
	mgs := []openai.ChatCompletionMessageParamUnion{}
	mgs = append(mgs, openai.SystemMessage(sysmgs))

	for _, message := range messages.Messages {
		switch message.Role {
		case "user":
			if len(message.Images) > 0 {
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
				// Simple text-only user message
				mgs = append(mgs, openai.UserMessage(message.Message))
			}
		case "assistant":
			mgs = append(mgs, openai.AssistantMessage(message.Message))
		case "system":
			mgs = append(mgs, openai.SystemMessage(message.Message))
		case "tool":
			mgs = append(mgs, openai.ToolMessage(message.Message, message.ToolCallId))
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
			"type":       "object",
			"properties": map[string]any{},
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

func (o *OpenAI) buildParams(messages models.AIChatHistory, enableTools bool) openai.ChatCompletionNewParams {
	mgs := formatMessages(messages, o.SystemMessage)
	params := openai.ChatCompletionNewParams{
		Model:    o.Model,
		Messages: mgs,
		Seed:     openai.Int(69),
	}
	params.SetExtraFields(o.ExtraFields)
	if o.Temperature > 0 {
		params.Temperature = openai.Float(o.Temperature)
	}
	if o.MaxTokens > 0 {
		if strings.Contains(o.Model, "gpt-5") {
			params.MaxCompletionTokens = openai.Int(o.MaxTokens)
		} else {
			params.MaxTokens = openai.Int(o.MaxTokens)
		}
	}
	if enableTools {
		var tools []openai.ChatCompletionToolUnionParam
		if o.hasMCPTools() {
			tools = append(tools, o.convertMCPToolsToOpenAI()...)
		}
		if o.hasGoFunctionTools() {
			tools = append(tools, o.convertGoFunctionToolsToOpenAI()...)
		}
		params.Tools = tools
	}
	if o.ReasoningEffort != nil {
		params.ReasoningEffort = *o.ReasoningEffort
	}
	return params
}

func (o *OpenAI) shouldExecuteTools(chatCompletion *openai.ChatCompletion, enableTools bool, useMCPExecution bool) bool {
	return enableTools && useMCPExecution && chatCompletion != nil && len(chatCompletion.Choices) > 0 && len(chatCompletion.Choices[0].Message.ToolCalls) > 0
}

func (o *OpenAI) toolPassLimit() int {
	if o.maxToolPasses > 0 {
		return o.maxToolPasses
	}
	return defaultMaxToolPasses
}

func (o *OpenAI) callAnyTool(ctx context.Context, name string, arguments map[string]any) (string, error) {
	if fn, ok := o.FunctionTools[name]; ok && fn.Handler != nil {
		return fn.Handler(ctx, FuncParams(arguments))
	}
	return o.callMCPTool(ctx, name, arguments)
}

func coerceFunctionParameters(params any) openai.FunctionParameters {
	switch p := params.(type) {
	case openai.FunctionParameters:
		return normalizeFunctionParameters(p)
	case map[string]any:
		return normalizeFunctionParameters(openai.FunctionParameters(p))
	default:
		data, err := json.Marshal(p)
		if err != nil {
			return normalizeFunctionParameters(nil)
		}
		var m map[string]any
		if err := json.Unmarshal(data, &m); err != nil {
			return normalizeFunctionParameters(nil)
		}
		return normalizeFunctionParameters(openai.FunctionParameters(m))
	}
}

func normalizeFunctionParameters(params openai.FunctionParameters) openai.FunctionParameters {
	if params == nil {
		return openai.FunctionParameters{
			"type":       "object",
			"properties": map[string]any{},
		}
	}
	if _, ok := params["type"]; !ok {
		params["type"] = "object"
	}
	if _, ok := params["properties"]; !ok {
		params["properties"] = map[string]any{}
	}
	return params
}

func (tool GoFunctionTool) toFunctionDefinitionParam() openai.FunctionDefinitionParam {
	return openai.FunctionDefinitionParam{
		Name:        tool.Name,
		Description: openai.String(tool.Description),
		Parameters:  normalizeFunctionParameters(tool.Parameters),
		Strict:      openai.Bool(tool.Strict),
	}
}

func (o *OpenAI) streamAndAccumulate(ctx context.Context, params openai.ChatCompletionNewParams, chunkHandler func(openai.ChatCompletionChunk)) (*openai.ChatCompletionAccumulator, error) {
	stream := o.Client.Chat.Completions.NewStreaming(ctx, params)
	acc := openai.ChatCompletionAccumulator{}
	for stream.Next() {
		chunk := stream.Current()
		acc.AddChunk(chunk)
		if chunkHandler != nil {
			chunkHandler(chunk)
		}
	}
	if err := stream.Err(); err != nil {
		return nil, err
	}
	return &acc, nil
}

func (o *OpenAI) hasGoFunctionTools() bool {
	return len(o.FunctionTools) > 0
}

func (o *OpenAI) convertGoFunctionToolsToOpenAI() []openai.ChatCompletionToolUnionParam {
	tools := make([]openai.ChatCompletionToolUnionParam, 0, len(o.FunctionTools))
	for _, tool := range o.FunctionTools {
		def := tool.toFunctionDefinitionParam()
		tools = append(tools, openai.ChatCompletionFunctionTool(def))
	}
	return tools
}
