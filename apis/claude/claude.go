package claude

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"time"

	"github.com/MelloB1989/karma/config"
	"github.com/MelloB1989/karma/models"
	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/anthropics/anthropic-sdk-go/packages/param"
	"github.com/invopop/jsonschema"
)

type ClaudeClient struct {
	Client       *anthropic.Client
	MaxTokens    int
	Model        anthropic.Model
	Temp         float64
	TopP         float64
	TopK         int64
	SystemPrompt string
	MCPTools     []MCPTool
	MCPServerURL string
	AuthToken    string
}

// MCPTool represents an MCP tool that can be called
type MCPTool struct {
	Name        string                         `json:"name"`
	Description string                         `json:"description"`
	InputSchema anthropic.ToolInputSchemaParam `json:"inputSchema"`
	MCPToolName string                         `json:"mcpToolName"` // The actual tool name in MCP server
}

// MCPRequest represents an MCP JSON-RPC request
type MCPRequest struct {
	JSONRPC string `json:"jsonrpc"`
	ID      int    `json:"id"`
	Method  string `json:"method"`
	Params  any    `json:"params,omitempty"`
}

// MCPResponse represents an MCP JSON-RPC response
type MCPResponse struct {
	JSONRPC string    `json:"jsonrpc"`
	ID      int       `json:"id"`
	Result  any       `json:"result,omitempty"`
	Error   *MCPError `json:"error,omitempty"`
}

// MCPError represents an MCP error
type MCPError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

// CallToolParams represents MCP tool call parameters
type CallToolParams struct {
	Name      string         `json:"name"`
	Arguments map[string]any `json:"arguments,omitempty"`
}

func NewClaudeClient(maxTokens int, model anthropic.Model, temp float64, topP float64, topK float64, systemPrompt string) *ClaudeClient {
	client := anthropic.NewClient(
		option.WithAPIKey(config.GetEnvRaw("ANTHROPIC_API_KEY")),
	)
	return &ClaudeClient{
		Client:       &client,
		MaxTokens:    maxTokens,
		Model:        model,
		Temp:         temp,
		TopP:         topP,
		TopK:         int64(topK),
		SystemPrompt: systemPrompt,
		MCPTools:     []MCPTool{},
		MCPServerURL: "",
		AuthToken:    "",
	}
}

// SetMCPServer configures the MCP server URL and authentication
func (cc *ClaudeClient) SetMCPServer(serverURL, authToken string) {
	cc.MCPServerURL = serverURL
	cc.AuthToken = authToken
}

// AddMCPTool adds an MCP tool that Claude can use
func (cc *ClaudeClient) AddMCPTool(name, description, mcpToolName string, inputSchema any) error {
	schema, err := cc.GenerateSchema(inputSchema)
	if err != nil {
		return fmt.Errorf("failed to generate schema for tool %s: %w", name, err)
	}

	tool := MCPTool{
		Name:        name,
		Description: description,
		InputSchema: schema,
		MCPToolName: mcpToolName,
	}

	cc.MCPTools = append(cc.MCPTools, tool)
	return nil
}

// generateSchema generates JSON schema from a Go struct
func (cc *ClaudeClient) GenerateSchema(inputStruct any) (anthropic.ToolInputSchemaParam, error) {
	reflector := jsonschema.Reflector{
		AllowAdditionalProperties: false,
		DoNotReference:            true,
	}

	var schema *jsonschema.Schema
	if reflect.TypeOf(inputStruct).Kind() == reflect.Ptr {
		schema = reflector.Reflect(inputStruct)
	} else {
		schema = reflector.ReflectFromType(reflect.TypeOf(inputStruct))
	}

	return anthropic.ToolInputSchemaParam{
		Properties: schema.Properties,
	}, nil
}

// callMCPTool calls an MCP tool and returns the result
func (cc *ClaudeClient) callMCPTool(toolName string, arguments map[string]any) (string, error) {
	if cc.MCPServerURL == "" {
		return "", fmt.Errorf("MCP server URL not configured")
	}

	// Find the MCP tool name
	var mcpToolName string
	for _, tool := range cc.MCPTools {
		if tool.Name == toolName {
			mcpToolName = tool.MCPToolName
			break
		}
	}
	if mcpToolName == "" {
		return "", fmt.Errorf("MCP tool not found: %s", toolName)
	}

	// Create MCP request
	request := MCPRequest{
		JSONRPC: "2.0",
		ID:      int(time.Now().Unix()),
		Method:  "tools/call",
		Params: CallToolParams{
			Name:      mcpToolName,
			Arguments: arguments,
		},
	}

	// Send request to MCP server
	jsonData, err := json.Marshal(request)
	if err != nil {
		return "", fmt.Errorf("failed to marshal MCP request: %w", err)
	}

	req, err := http.NewRequest("POST", cc.MCPServerURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create HTTP request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if cc.AuthToken != "" {
		req.Header.Set("Authorization", "Bearer "+cc.AuthToken)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send MCP request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read MCP response: %w", err)
	}

	var mcpResp MCPResponse
	err = json.Unmarshal(body, &mcpResp)
	if err != nil {
		return "", fmt.Errorf("failed to unmarshal MCP response: %w", err)
	}

	if mcpResp.Error != nil {
		return "", fmt.Errorf("MCP error %d: %s", mcpResp.Error.Code, mcpResp.Error.Message)
	}

	// Extract text from MCP response
	resultMap, ok := mcpResp.Result.(map[string]any)
	if !ok {
		return fmt.Sprintf("%v", mcpResp.Result), nil
	}

	content, ok := resultMap["content"].([]any)
	if !ok || len(content) == 0 {
		return fmt.Sprintf("%v", mcpResp.Result), nil
	}

	textContent, ok := content[0].(map[string]any)
	if !ok {
		return fmt.Sprintf("%v", content[0]), nil
	}

	text, ok := textContent["text"].(string)
	if !ok {
		return fmt.Sprintf("%v", textContent), nil
	}

	return text, nil
}

func (cc *ClaudeClient) ClaudeSinglePrompt(prompt string) (string, error) {
	mgsParam := anthropic.MessageNewParams{
		MaxTokens: int64(cc.MaxTokens),
		Messages: []anthropic.MessageParam{{
			Content: []anthropic.ContentBlockParamUnion{{
				OfText: &anthropic.TextBlockParam{Text: prompt},
			}},
			Role: anthropic.MessageParamRoleUser,
		}},
		Model: cc.Model,
	}
	if cc.Temp > 0 {
		mgsParam.Temperature = param.NewOpt(cc.Temp)
	}
	if cc.TopP > 0 {
		mgsParam.TopP = param.NewOpt(cc.TopP)
	}
	if cc.TopK > 0 {
		mgsParam.TopK = param.NewOpt(cc.TopK)
	}
	if cc.SystemPrompt != "" {
		mgsParam.System = []anthropic.TextBlockParam{{Text: cc.SystemPrompt}}
	}
	message, err := cc.Client.Messages.New(context.TODO(), mgsParam)
	if err != nil {
		return "", err
	}
	return message.Content[0].Text, nil
}

func (cc *ClaudeClient) ClaudeChatCompletion(messages models.AIChatHistory) (string, error) {
	return cc.ClaudeChatCompletionWithTools(messages, false)
}

// ClaudeChatCompletionWithTools handles chat completion with optional MCP tool support
func (cc *ClaudeClient) ClaudeChatCompletionWithTools(messages models.AIChatHistory, enableTools bool) (string, error) {
	processedMessages := processMessages(messages)
	mgsParam := anthropic.MessageNewParams{
		MaxTokens: int64(cc.MaxTokens),
		Messages:  processedMessages,
		Model:     cc.Model,
	}
	if cc.Temp > 0 {
		mgsParam.Temperature = param.NewOpt(cc.Temp)
	}
	if cc.TopP > 0 {
		mgsParam.TopP = param.NewOpt(cc.TopP)
	}
	if cc.TopK > 0 {
		mgsParam.TopK = param.NewOpt(cc.TopK)
	}
	if cc.SystemPrompt != "" {
		mgsParam.System = []anthropic.TextBlockParam{{Text: cc.SystemPrompt}}
	}

	// Add MCP tools if enabled and available
	if enableTools && len(cc.MCPTools) > 0 {
		tools := make([]anthropic.ToolUnionParam, len(cc.MCPTools))
		for i, mcpTool := range cc.MCPTools {
			toolParam := anthropic.ToolParam{
				Name:        mcpTool.Name,
				Description: anthropic.String(mcpTool.Description),
				InputSchema: mcpTool.InputSchema,
			}
			tools[i] = anthropic.ToolUnionParam{OfTool: &toolParam}
		}
		mgsParam.Tools = tools
	}

	for {
		message, err := cc.Client.Messages.New(context.TODO(), mgsParam)
		if err != nil {
			return "", err
		}

		// Check if Claude wants to use tools
		var toolResults []anthropic.ContentBlockParamUnion
		var hasToolUse bool
		var responseText string

		for _, block := range message.Content {
			switch block := block.AsAny().(type) {
			case anthropic.TextBlock:
				responseText += block.Text
			case anthropic.ToolUseBlock:
				hasToolUse = true
				if enableTools {
					// Call the MCP tool
					var arguments map[string]any
					err := json.Unmarshal(block.Input, &arguments)
					if err != nil {
						toolResults = append(toolResults, anthropic.NewToolResultBlock(block.ID,
							fmt.Sprintf("Error parsing arguments: %v", err), true))
						continue
					}

					result, err := cc.callMCPTool(block.Name, arguments)
					if err != nil {
						fmt.Println(err)
						toolResults = append(toolResults, anthropic.NewToolResultBlock(block.ID,
							fmt.Sprintf("Error calling tool: %v", err), true))
					} else {
						toolResults = append(toolResults, anthropic.NewToolResultBlock(block.ID, result, false))
					}
				}
			}
		}

		if !hasToolUse || !enableTools {
			return responseText, nil
		}

		// Continue the conversation with tool results
		processedMessages = append(processedMessages, message.ToParam())
		if len(toolResults) > 0 {
			processedMessages = append(processedMessages, anthropic.NewUserMessage(toolResults...))
		}
		mgsParam.Messages = processedMessages
	}
}

func (cc *ClaudeClient) ClaudeStreamCompletion(messages models.AIChatHistory, callback func(chunck models.StreamedResponse) error) (string, error) {
	return cc.ClaudeStreamCompletionWithTools(messages, callback, false)
}

// ClaudeStreamCompletionWithTools handles streaming completion with optional MCP tool support
func (cc *ClaudeClient) ClaudeStreamCompletionWithTools(messages models.AIChatHistory, callback func(chunck models.StreamedResponse) error, enableTools bool) (string, error) {
	processedMessages := processMessages(messages)
	streamParams := anthropic.MessageNewParams{
		MaxTokens: int64(cc.MaxTokens),
		Messages:  processedMessages,
		Model:     cc.Model,
	}
	if cc.Temp > 0 {
		streamParams.Temperature = param.NewOpt(cc.Temp)
	}
	if cc.TopP > 0 {
		streamParams.TopP = param.NewOpt(cc.TopP)
	}
	if cc.TopK > 0 {
		streamParams.TopK = param.NewOpt(cc.TopK)
	}
	if cc.SystemPrompt != "" {
		streamParams.System = []anthropic.TextBlockParam{{Text: cc.SystemPrompt}}
	}

	// Add MCP tools if enabled and available
	if enableTools && len(cc.MCPTools) > 0 {
		tools := make([]anthropic.ToolUnionParam, len(cc.MCPTools))
		for i, mcpTool := range cc.MCPTools {
			toolParam := anthropic.ToolParam{
				Name:        mcpTool.Name,
				Description: anthropic.String(mcpTool.Description),
				InputSchema: mcpTool.InputSchema,
			}
			tools[i] = anthropic.ToolUnionParam{OfTool: &toolParam}
		}
		streamParams.Tools = tools
	}

	stream := cc.Client.Messages.NewStreaming(context.TODO(), streamParams)
	message := anthropic.Message{}
	for stream.Next() {
		event := stream.Current()
		err := message.Accumulate(event)
		if err != nil {
			return "", err
		}

		switch eventVariant := event.AsAny().(type) {
		case anthropic.ContentBlockDeltaEvent:
			switch deltaVariant := eventVariant.Delta.AsAny().(type) {
			case anthropic.TextDelta:
				chunk := models.StreamedResponse{
					AIResponse: deltaVariant.Text,
				}
				if err := callback(chunk); err != nil {
					return "", err
				}
			}
		}
	}

	if stream.Err() != nil {
		return "", stream.Err()
	}

	// Handle tool calls if any
	if enableTools && len(message.Content) > 0 {
		var toolResults []anthropic.ContentBlockParamUnion
		var hasToolUse bool

		for _, block := range message.Content {
			if block, ok := block.AsAny().(anthropic.ToolUseBlock); ok {
				hasToolUse = true
				var arguments map[string]any
				err := json.Unmarshal(block.Input, &arguments)
				if err != nil {
					toolResults = append(toolResults, anthropic.NewToolResultBlock(block.ID,
						fmt.Sprintf("Error parsing arguments: %v", err), true))
					continue
				}

				result, err := cc.callMCPTool(block.Name, arguments)
				if err != nil {
					toolResults = append(toolResults, anthropic.NewToolResultBlock(block.ID,
						fmt.Sprintf("Error calling tool: %v", err), true))
				} else {
					toolResults = append(toolResults, anthropic.NewToolResultBlock(block.ID, result, false))
					// Stream the tool result
					chunk := models.StreamedResponse{
						AIResponse: fmt.Sprintf("\n[Tool Result: %s]", result),
					}
					if err := callback(chunk); err != nil {
						return "", err
					}
				}
			}
		}

		if hasToolUse && len(toolResults) > 0 {
			// Continue with tool results (simplified for streaming)
			processedMessages = append(processedMessages, message.ToParam())
			processedMessages = append(processedMessages, anthropic.NewUserMessage(toolResults...))

			// Make a follow-up call to get the final response
			followUpParams := streamParams
			followUpParams.Messages = processedMessages
			followUpParams.Tools = nil // Disable tools for follow-up to avoid loops

			followUpStream := cc.Client.Messages.NewStreaming(context.TODO(), followUpParams)
			for followUpStream.Next() {
				event := followUpStream.Current()
				switch eventVariant := event.AsAny().(type) {
				case anthropic.ContentBlockDeltaEvent:
					switch deltaVariant := eventVariant.Delta.AsAny().(type) {
					case anthropic.TextDelta:
						chunk := models.StreamedResponse{
							AIResponse: deltaVariant.Text,
						}
						if err := callback(chunk); err != nil {
							return "", err
						}
					}
				}
			}
			if followUpStream.Err() != nil {
				return "", followUpStream.Err()
			}
		}
	}

	if len(message.Content) > 0 {
		if textBlock, ok := message.Content[0].AsAny().(anthropic.TextBlock); ok {
			return textBlock.Text, nil
		}
	}
	return "", nil
}
