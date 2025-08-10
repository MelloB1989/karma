package claude

import (
	"context"
	"encoding/json"
	"fmt"

	mcp "github.com/MelloB1989/karma/ai/mcp_client"
	"github.com/MelloB1989/karma/config"
	"github.com/MelloB1989/karma/models"
	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/anthropics/anthropic-sdk-go/packages/param"
)

type ClaudeClient struct {
	Client          *anthropic.Client
	MaxTokens       int
	Model           anthropic.Model
	Temp            float64
	TopP            float64
	TopK            int64
	SystemPrompt    string
	MCPManager      *mcp.Manager
	MultiMCPManager *mcp.MultiManager
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
		MCPManager:   nil,
	}
}

// SetMCPServer configures the MCP server and creates a tool manager
func (cc *ClaudeClient) SetMCPServer(serverURL, authToken string) {
	mcpClient := mcp.NewClient(serverURL, authToken)
	cc.MCPManager = mcp.NewManager(mcpClient)
}

// SetMultiMCPManager configures multiple MCP servers
func (cc *ClaudeClient) SetMultiMCPManager(multiManager *mcp.MultiManager) {
	cc.MultiMCPManager = multiManager
}

// AddMCPTool adds an MCP tool that Claude can use
func (cc *ClaudeClient) AddMCPTool(name, description, mcpToolName string, inputSchema any) error {
	if cc.MCPManager == nil {
		return fmt.Errorf("MCP server not configured. Call SetMCPServer first")
	}
	return cc.MCPManager.AddToolFromSchema(name, description, mcpToolName, inputSchema)
}

// GetMCPManager returns the MCP manager for advanced tool management
func (cc *ClaudeClient) GetMCPManager() *mcp.Manager {
	return cc.MCPManager
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

// ClaudeChatCompletionWithTools handles chat completion with optional MCP tool support
func (cc *ClaudeClient) ClaudeChatCompletion(messages models.AIChatHistory, enableTools bool) (string, error) {
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
	if enableTools && cc.hasMCPTools() {
		mgsParam.Tools = cc.convertMCPToolsToAnthropic()
	}

	ctx := context.TODO()
	for {
		message, err := cc.Client.Messages.New(ctx, mgsParam)
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

					result, err := cc.callMCPTool(ctx, block.Name, arguments)
					if err != nil {
						fmt.Printf("MCP tool error: %v\n", err)
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
	if enableTools && cc.hasMCPTools() {
		streamParams.Tools = cc.convertMCPToolsToAnthropic()
	}

	ctx := context.TODO()
	stream := cc.Client.Messages.NewStreaming(ctx, streamParams)
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

				result, err := cc.callMCPTool(ctx, block.Name, arguments)
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

			followUpStream := cc.Client.Messages.NewStreaming(ctx, followUpParams)
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
