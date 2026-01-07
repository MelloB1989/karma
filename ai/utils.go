package ai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	mcp "github.com/MelloB1989/karma/ai/mcp_client"
	"github.com/MelloB1989/karma/apis/claude"
	"github.com/MelloB1989/karma/apis/gemini"
	"github.com/MelloB1989/karma/config"
	"github.com/MelloB1989/karma/internal/openai"
	"github.com/MelloB1989/karma/models"
	"google.golang.org/genai"
)

const llama_single_prompt_format = `
	<|begin_of_text|><|start_header_id|>system<|end_header_id|>

Cutting Knowledge Date: December 2023
Today Date: %s

%s<|eot_id|><|start_header_id|>user<|end_header_id|>

%s<|eot_id|><|start_header_id|>assistant<|end_header_id|>
`

const llama_system_prompt_format = `
<|begin_of_text|><|start_header_id|>system<|end_header_id|>

Cutting Knowledge Date: December 2023
Today Date: %s

%s<|eot_id|>
`

const user_message_format = `
<|start_header_id|>user<|end_header_id|>

%s<|eot_id|>
`

const assistant_message_format = `
<|start_header_id|>assistant<|end_header_id|>

%s<|eot_id|>
`

const role_message_format = `
<|start_header_id|>%s<|end_header_id|>

%s<|eot_id|>
`

const assitant_end = `
<|start_header_id|>assistant<|end_header_id|>
`

func (kai *KarmaAI) addUserPreprompt(chat models.AIChatHistory) models.AIChatHistory {
	if len(chat.Messages) == 0 {
		return chat
	}
	chat.Messages[len(chat.Messages)-1].Message = kai.UserPrePrompt + "\n" + chat.Messages[len(chat.Messages)-1].Message
	return chat
}

func (kai *KarmaAI) removeUserPrePrompt(chat models.AIChatHistory) models.AIChatHistory {
	if len(chat.Messages) == 0 {
		return chat
	}

	last := &chat.Messages[len(chat.Messages)-1]
	prefix := kai.UserPrePrompt + "\n"

	last.Message, _ = strings.CutPrefix(last.Message, prefix)
	return chat
}

func (kai *KarmaAI) processMessagesForLlamaBedrockSystemPrompt(chat models.AIChatHistory) string {
	var finalPrompt strings.Builder
	finalPrompt.WriteString(fmt.Sprintf(llama_system_prompt_format, time.Now().String(), kai.SystemMessage))
	for _, message := range chat.Messages {
		if message.Role == models.User {
			finalPrompt.WriteString(fmt.Sprintf(user_message_format, message.Message))
		} else if message.Role == models.Assistant {
			finalPrompt.WriteString(fmt.Sprintf(assistant_message_format, message.Message))
		} else {
			finalPrompt.WriteString(fmt.Sprintf(role_message_format, message.Role, message.Message))
		}
	}
	finalPrompt.WriteString(assitant_end)

	return finalPrompt.String()
}

func (kai *KarmaAI) configureClaudeClientForMCP(cc *claude.ClaudeClient) {
	if len(kai.MCPServers) > 0 {
		kai.configureMultiMCPForClaude(cc)
	} else if len(kai.MCPTools) > 0 {
		cc.SetMCPServer(kai.MCPUrl, kai.AuthToken)
		for _, tool := range kai.MCPTools {
			err := cc.AddMCPTool(tool.ToolName, tool.Description, tool.ToolName, tool.InputSchema) // Claude requires tool names to match ^[a-zA-Z0-9_-]{1,128}$ (letters, numbers, underscore, hyphen only)
			if err != nil {
				log.Printf("Failed to add MCP tool: %v", err)
			}
		}
	}
}

func (kai *KarmaAI) configureOpenaiClientForMCP(o *openai.OpenAI) {
	if len(kai.MCPServers) > 0 {
		kai.configureMultiMCPForOpenAI(o)
	} else if len(kai.MCPTools) > 0 {
		o.SetMCPServer(kai.MCPUrl, kai.AuthToken)
		for _, tool := range kai.MCPTools {
			err := o.AddMCPTool(tool.ToolName, tool.Description, tool.ToolName, tool.InputSchema) // Claude requires tool names to match ^[a-zA-Z0-9_-]{1,128}$ (letters, numbers, underscore, hyphen only)
			if err != nil {
				log.Printf("Failed to add MCP tool: %v", err)
			}
		}
	}
	if kai.MaxToolPasses > 0 {
		o.SetMaxToolPasses(kai.MaxToolPasses)
	}
	for _, fnTool := range kai.GoFunctionTools {
		tool := openai.GoFunctionTool{
			Name:        fnTool.Name,
			Description: fnTool.Description,
			Parameters:  fnTool.Parameters,
			Strict:      fnTool.Strict,
			Handler:     fnTool.Handler,
		}
		if err := o.AddGoFunctionTool(tool); err != nil {
			log.Printf("Failed to add Go function tool: %v", err)
		}
	}
}

func (kai *KarmaAI) configureMultiMCPForOpenAI(o *openai.OpenAI) {
	multiManager := mcp.NewMultiManager()

	for i, server := range kai.MCPServers {
		serverID := fmt.Sprintf("server_%d", i)
		multiManager.AddServer(serverID, server.URL, server.AuthToken)

		for _, tool := range server.Tools {
			err := multiManager.AddToolToServer(serverID, tool.ToolName, tool.Description, tool.ToolName, tool.InputSchema)
			if err != nil {
				log.Printf("Failed to add MCP tool %s to server %s: %v", tool.FriendlyName, serverID, err)
			}
		}
	}

	o.SetMultiMCPManager(multiManager)
}

func (kai *KarmaAI) configureMultiMCPForClaude(cc *claude.ClaudeClient) {
	multiManager := mcp.NewMultiManager()

	for i, server := range kai.MCPServers {
		serverID := fmt.Sprintf("server_%d", i)
		multiManager.AddServer(serverID, server.URL, server.AuthToken)

		for _, tool := range server.Tools {
			err := multiManager.AddToolToServer(serverID, tool.ToolName, tool.Description, tool.ToolName, tool.InputSchema)
			if err != nil {
				log.Printf("Failed to add MCP tool %s to server %s: %v", tool.FriendlyName, serverID, err)
			}
		}
	}

	cc.SetMultiMCPManager(multiManager)
}

// createGeminiClient creates a Gemini client using SpecialConfig or environment variables
// Each SpecialConfig field is optional and overrides its corresponding environment variable
func (kai *KarmaAI) createGeminiClient() (*gemini.Gemini, error) {
	// Check for API key first (uses Gemini API backend)
	if apiKey, ok := kai.SpecialConfig[GoogleAPIKey].(string); ok && apiKey != "" {
		return gemini.NewGeminiWithAPIKey(
			kai.Model.GetModelString(),
			kai.SystemMessage,
			float64(kai.Temperature),
			float64(kai.TopP),
			float64(kai.TopK),
			int64(kai.MaxTokens),
			apiKey,
		)
	}

	// For Vertex AI, each field can be individually overridden
	// Start with environment variables as defaults
	projectID := config.GetEnvRaw("GOOGLE_PROJECT_ID")
	location := config.GetEnvRaw("GOOGLE_LOCATION")

	// Override with SpecialConfig if set
	if configProjectID, ok := kai.SpecialConfig[GoogleProjectID].(string); ok && configProjectID != "" {
		projectID = configProjectID
	}
	if configLocation, ok := kai.SpecialConfig[GoogleLocation].(string); ok && configLocation != "" {
		location = configLocation
	}

	return gemini.NewGeminiWithVertexAI(
		kai.Model.GetModelString(),
		kai.SystemMessage,
		float64(kai.Temperature),
		float64(kai.TopP),
		float64(kai.TopK),
		int64(kai.MaxTokens),
		projectID,
		location,
	)
}

func (kai *KarmaAI) configureGeminiClient(g *gemini.Gemini) {
	kai.configureGeminiClientForMCP(g)
	if kai.ResponseType != "" {
		g.SetResponseType(kai.ResponseType)
	}
	if kai.MaxToolPasses > 0 {
		g.SetMaxToolPasses(kai.MaxToolPasses)
	}
}

func (kai *KarmaAI) configureGeminiClientForMCP(g *gemini.Gemini) {
	// Configure MCP tools
	if kai.MCPUrl != "" && kai.AuthToken != "" {
		g.SetMCPServer(kai.MCPUrl, kai.AuthToken)
	}

	// Add Go function tools
	for _, tool := range kai.GoFunctionTools {
		// Capture tool in local variable to avoid closure issue
		toolCopy := tool
		geminiTool := gemini.GoFunctionTool{
			Name:        toolCopy.Name,
			Description: toolCopy.Description,
			Parameters:  convertOpenAIParamsToGeminiSchema(toolCopy.Parameters),
			Handler: func(ctx context.Context, fp gemini.FuncParams) (string, error) {
				// Convert gemini.FuncParams to openai.FuncParams
				openaiParams := make(map[string]any)
				for k, v := range fp {
					openaiParams[k] = v
				}
				return toolCopy.Handler(ctx, openaiParams)
			},
		}
		g.AddGoFunctionTool(geminiTool)
	}
}

func convertOpenAIParamsToGeminiSchema(params any) *genai.Schema {
	if params == nil {
		return &genai.Schema{
			Type:       "object",
			Properties: map[string]*genai.Schema{},
		}
	}

	// Handle map[string]any format (OpenAI FunctionParameters)
	// First, try to convert via JSON marshaling to handle typed maps
	var paramsMap map[string]any

	switch p := params.(type) {
	case map[string]any:
		paramsMap = p
	default:
		// Try JSON round-trip for other map types (like openai.FunctionParameters)
		jsonBytes, err := json.Marshal(params)
		if err != nil {
			return &genai.Schema{
				Type:       "object",
				Properties: map[string]*genai.Schema{},
			}
		}
		if err := json.Unmarshal(jsonBytes, &paramsMap); err != nil {
			return &genai.Schema{
				Type:       "object",
				Properties: map[string]*genai.Schema{},
			}
		}
	}

	schema := &genai.Schema{
		Type:       "object",
		Properties: make(map[string]*genai.Schema),
	}

	if properties, ok := paramsMap["properties"].(map[string]any); ok {
		for name, prop := range properties {
			if propMap, ok := prop.(map[string]any); ok {
				schema.Properties[name] = convertPropertyMapToSchema(propMap)
			}
		}
	}

	if required, ok := paramsMap["required"].([]any); ok {
		reqStrings := make([]string, len(required))
		for i, r := range required {
			if s, ok := r.(string); ok {
				reqStrings[i] = s
			}
		}
		schema.Required = reqStrings
	}

	// Also handle required as []string (some callers may use this format)
	if required, ok := paramsMap["required"].([]string); ok {
		schema.Required = required
	}

	return schema
}

func convertPropertyMapToSchema(propMap map[string]any) *genai.Schema {
	schema := &genai.Schema{}

	if typeStr, ok := propMap["type"].(string); ok {
		// Use lowercase type strings to match OpenAPI/JSON Schema spec
		// The genai package accepts both formats, but lowercase is more compatible
		switch typeStr {
		case "string":
			schema.Type = "string"
		case "number":
			schema.Type = "number"
		case "integer":
			schema.Type = "integer"
		case "boolean":
			schema.Type = "boolean"
		case "array":
			schema.Type = "array"
			if items, ok := propMap["items"].(map[string]any); ok {
				schema.Items = convertPropertyMapToSchema(items)
			}
		case "object":
			schema.Type = "object"
			if props, ok := propMap["properties"].(map[string]any); ok {
				schema.Properties = make(map[string]*genai.Schema)
				for name, prop := range props {
					if propMap, ok := prop.(map[string]any); ok {
						schema.Properties[name] = convertPropertyMapToSchema(propMap)
					}
				}
			}
		default:
			schema.Type = "string" // Default to string for unknown types
		}
	}

	if desc, ok := propMap["description"].(string); ok {
		schema.Description = desc
	}

	if enum, ok := propMap["enum"].([]any); ok {
		enumStrings := make([]string, len(enum))
		for i, e := range enum {
			if s, ok := e.(string); ok {
				enumStrings[i] = s
			}
		}
		schema.Enum = enumStrings
	}

	return schema
}

func buildGeminiChatResponse(response *genai.GenerateContentResponse, startTime time.Time) (*models.AIChatResponse, error) {
	if response == nil || len(response.Candidates) == 0 {
		return nil, errors.New("no response from Gemini")
	}

	res := &models.AIChatResponse{
		AIResponse: response.Text(),
		TimeTaken:  int(time.Since(startTime).Milliseconds()),
	}

	if response.UsageMetadata != nil {
		res.Tokens = int(response.UsageMetadata.TotalTokenCount)
		res.InputTokens = int(response.UsageMetadata.PromptTokenCount)
		res.OutputTokens = int(response.UsageMetadata.CandidatesTokenCount)
	}

	// Add tool calls if present
	functionCalls := response.FunctionCalls()
	if len(functionCalls) > 0 {
		res.ToolCalls = buildToolCallsFromGemini(functionCalls)
	}

	return res, nil
}

func buildToolCallsFromGemini(functionCalls []*genai.FunctionCall) []models.ToolCall {
	result := make([]models.ToolCall, len(functionCalls))
	for i, fc := range functionCalls {
		argsJSON, _ := json.Marshal(fc.Args)
		result[i] = models.ToolCall{
			ID:   fc.ID,
			Type: "function",
			Function: models.ToolCallFunction{
				Name:      fc.Name,
				Arguments: string(argsJSON),
			},
		}
	}
	return result
}

func createGeminiChunkHandler(callback func(chunk models.StreamedResponse) error) func(*genai.GenerateContentResponse) {
	return func(chunk *genai.GenerateContentResponse) {
		if chunk == nil {
			return
		}

		streamResp := models.StreamedResponse{
			AIResponse: chunk.Text(),
		}

		if chunk.UsageMetadata != nil {
			streamResp.TokenUsed = int(chunk.UsageMetadata.TotalTokenCount)
		}

		// Add tool calls if present
		functionCalls := chunk.FunctionCalls()
		if len(functionCalls) > 0 {
			streamResp.ToolCalls = buildToolCallsFromGemini(functionCalls)
		}

		callback(streamResp)
	}
}
