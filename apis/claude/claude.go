package claude

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	mcp "github.com/MelloB1989/karma/ai/mcp_client"
	"github.com/MelloB1989/karma/config"
	"github.com/MelloB1989/karma/models"
	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/anthropics/anthropic-sdk-go/packages/param"
)

func extractToolCallsFromClaude(content []anthropic.ContentBlockUnion) []models.ToolCall {
	toolCalls := make([]models.ToolCall, 0, len(content))
	for _, block := range content {
		if toolUse, ok := block.AsAny().(anthropic.ToolUseBlock); ok {
			toolCalls = append(toolCalls, models.ToolCall{
				ID:   toolUse.ID,
				Type: string(toolUse.Type),
				Function: models.ToolCallFunction{
					Name:      toolUse.Name,
					Arguments: string(toolUse.Input),
				},
			})
		}
	}
	return toolCalls
}

type GoFunctionTool struct {
	Name        string
	Description string
	Parameters  map[string]any
	Handler     func(context.Context, map[string]any) (string, error)
}

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
	RequestGate     func() error
	RequestTimeout  time.Duration
	FunctionTools   map[string]GoFunctionTool
	MaxToolPasses   int
}

func (cc *ClaudeClient) isThinkingModel() bool {
	return strings.Contains(string(cc.Model), "thinking")
}

func NewClaudeClient(maxTokens int, model anthropic.Model, temp float64, topP float64, topK float64, systemPrompt string) *ClaudeClient {
	var opts []option.RequestOption
	if key := config.GetEnvRaw("ANTHROPIC_API_KEY"); key != "" {
		opts = append(opts, option.WithAPIKey(key))
	}
	if baseURL := config.GetEnvRaw("ANTHROPIC_BASE_URL"); baseURL != "" {
		opts = append(opts, option.WithBaseURL(baseURL))
	}
	if token := config.GetEnvRaw("ANTHROPIC_AUTH_TOKEN"); token != "" && token != config.GetEnvRaw("ANTHROPIC_API_KEY") {
		opts = append(opts, option.WithAuthToken(token))
	}
	if config.GetEnvRaw("ANTHROPIC_BASE_URL") != "" {
		opts = append(opts,
			option.WithHTTPClient(&http.Client{
				Transport: &http.Transport{
					DisableCompression: true,
					Proxy:              http.ProxyFromEnvironment,
				},
			}),
			option.WithHeader("User-Agent", "karma-ai-sdk/anthropic"),
			option.WithHeaderDel("X-Stainless-Lang"),
			option.WithHeaderDel("X-Stainless-Package-Version"),
			option.WithHeaderDel("X-Stainless-OS"),
			option.WithHeaderDel("X-Stainless-Arch"),
			option.WithHeaderDel("X-Stainless-Runtime"),
			option.WithHeaderDel("X-Stainless-Runtime-Version"),
			option.WithHeaderDel("X-Stainless-Retry-Count"),
			option.WithHeaderDel("X-Stainless-Timeout"),
		)
	}
	client := anthropic.NewClient(opts...)
	return &ClaudeClient{
		Client:        &client,
		MaxTokens:     maxTokens,
		Model:         model,
		Temp:          temp,
		TopP:          topP,
		TopK:          int64(topK),
		SystemPrompt:  systemPrompt,
		MCPManager:    nil,
		FunctionTools: make(map[string]GoFunctionTool),
		MaxToolPasses: 10,
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

func (cc *ClaudeClient) requestContext() (context.Context, context.CancelFunc) {
	if cc.RequestTimeout <= 0 {
		return context.Background(), func() {}
	}
	return context.WithTimeout(context.Background(), cc.RequestTimeout)
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

func (cc *ClaudeClient) ClaudeSinglePrompt(prompt string) (*models.AIChatResponse, error) {
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
	if cc.TopP > 0 && !cc.isThinkingModel() {
		mgsParam.TopP = param.NewOpt(cc.TopP)
	}
	if cc.TopK > 0 && !cc.isThinkingModel() {
		mgsParam.TopK = param.NewOpt(cc.TopK)
	}
	if cc.SystemPrompt != "" {
		mgsParam.System = []anthropic.TextBlockParam{{Text: cc.SystemPrompt}}
	}
	if cc.RequestGate != nil {
		if err := cc.RequestGate(); err != nil {
			return nil, err
		}
	}
	ctx, cancel := cc.requestContext()
	defer cancel()
	message, err := cc.Client.Messages.New(ctx, mgsParam)
	if err != nil {
		return nil, err
	}
	var thinkingText, responseText string
	for _, block := range message.Content {
		switch b := block.AsAny().(type) {
		case anthropic.ThinkingBlock:
			thinkingText = b.Thinking
		case anthropic.TextBlock:
			responseText = b.Text
		}
	}
	if thinkingText != "" {
		responseText = "<think>" + thinkingText + "</think>" + responseText
	}
	return &models.AIChatResponse{
		AIResponse:   responseText,
		InputTokens:  int(message.Usage.InputTokens),
		OutputTokens: int(message.Usage.OutputTokens),
	}, nil
}

// ClaudeChatCompletionWithTools handles chat completion with optional MCP tool support
func (cc *ClaudeClient) ClaudeChatCompletion(messages models.AIChatHistory, enableTools bool, useMCPExecution bool) (*models.AIChatResponse, error) {
	processedMessages := processMessages(messages)
	mgsParam := anthropic.MessageNewParams{
		MaxTokens: int64(cc.MaxTokens),
		Messages:  processedMessages,
		Model:     cc.Model,
	}
	if cc.Temp > 0 {
		mgsParam.Temperature = param.NewOpt(cc.Temp)
	}
	if cc.TopP > 0 && !cc.isThinkingModel() {
		mgsParam.TopP = param.NewOpt(cc.TopP)
	}
	if cc.TopK > 0 && !cc.isThinkingModel() {
		mgsParam.TopK = param.NewOpt(cc.TopK)
	}
	if cc.SystemPrompt != "" {
		mgsParam.System = []anthropic.TextBlockParam{{Text: cc.SystemPrompt}}
	}

	// Add tools if enabled and available
	if enableTools && cc.hasAnyTools() {
		mgsParam.Tools = cc.getAllToolsAsAnthropic()
	}

	ctx, cancel := cc.requestContext()
	defer cancel()

	maxPasses := cc.MaxToolPasses
	if maxPasses <= 0 {
		maxPasses = 10
	}

	for round := 0; round <= maxPasses; round++ {
		if cc.RequestGate != nil {
			if err := cc.RequestGate(); err != nil {
				return nil, err
			}
		}
		message, err := cc.Client.Messages.New(ctx, mgsParam)
		if err != nil {
			return nil, err
		}

		// Check if Claude wants to use tools
		var toolResults []anthropic.ContentBlockParamUnion
		var hasToolUse bool
		var responseText string
		var thinkingText string

		for _, block := range message.Content {
			switch block := block.AsAny().(type) {
			case anthropic.ThinkingBlock:
				thinkingText += block.Thinking
			case anthropic.TextBlock:
				responseText += block.Text
			case anthropic.ToolUseBlock:
				hasToolUse = true
				// If not using MCP execution, return immediately with tool calls for external handling
				if !useMCPExecution {
					return &models.AIChatResponse{
						AIResponse: responseText,
						ToolCalls:  extractToolCallsFromClaude(message.Content),
					}, nil
				}
				if enableTools {
					// Call the MCP tool
					var arguments map[string]any
					err := json.Unmarshal(block.Input, &arguments)
					if err != nil {
						toolResults = append(toolResults, anthropic.NewToolResultBlock(block.ID,
							fmt.Sprintf("Error parsing arguments: %v", err), true))
						continue
					}

					result, err := cc.callTool(ctx, block.Name, arguments)
					if err != nil {
						fmt.Printf("Tool error: %v\n", err)
						toolResults = append(toolResults, anthropic.NewToolResultBlock(block.ID,
							fmt.Sprintf("Error calling tool: %v", err), true))
					} else {
						toolResults = append(toolResults, anthropic.NewToolResultBlock(block.ID, result, false))
					}
				}
			}
		}

		if !hasToolUse || !enableTools {
			if thinkingText != "" {
				responseText = "<think>" + thinkingText + "</think>" + responseText
			}
			return &models.AIChatResponse{
				AIResponse:   responseText,
				InputTokens:  int(message.Usage.InputTokens),
				OutputTokens: int(message.Usage.OutputTokens),
				Tokens:       int(message.Usage.InputTokens) + int(message.Usage.OutputTokens),
			}, nil
		}

		// Continue the conversation with tool results
		processedMessages = append(processedMessages, message.ToParam())
		if len(toolResults) > 0 {
			processedMessages = append(processedMessages, anthropic.NewUserMessage(toolResults...))
		}
		mgsParam.Messages = processedMessages
	}

	return nil, fmt.Errorf("exceeded maximum tool passes (%d)", maxPasses)
}

func (cc *ClaudeClient) ClaudeStreamCompletion(messages models.AIChatHistory, callback func(chunck models.StreamedResponse) error) (*models.AIChatResponse, error) {
	return cc.ClaudeStreamCompletionWithTools(messages, callback, false, true)
}

// ClaudeStreamCompletionWithTools handles streaming completion with optional MCP tool support
func (cc *ClaudeClient) ClaudeStreamCompletionWithTools(messages models.AIChatHistory, callback func(chunck models.StreamedResponse) error, enableTools bool, useMCPExecution bool) (*models.AIChatResponse, error) {
	processedMessages := processMessages(messages)
	streamParams := anthropic.MessageNewParams{
		MaxTokens: int64(cc.MaxTokens),
		Messages:  processedMessages,
		Model:     cc.Model,
	}
	if cc.Temp > 0 {
		streamParams.Temperature = param.NewOpt(cc.Temp)
	}
	if cc.TopP > 0 && !cc.isThinkingModel() {
		streamParams.TopP = param.NewOpt(cc.TopP)
	}
	if cc.TopK > 0 && !cc.isThinkingModel() {
		streamParams.TopK = param.NewOpt(cc.TopK)
	}
	if cc.SystemPrompt != "" {
		streamParams.System = []anthropic.TextBlockParam{{Text: cc.SystemPrompt}}
	}

	// Add tools if enabled and available
	if enableTools && cc.hasAnyTools() {
		streamParams.Tools = cc.getAllToolsAsAnthropic()
	}

	ctx, cancel := cc.requestContext()
	defer cancel()

	maxPasses := cc.MaxToolPasses
	if maxPasses <= 0 {
		maxPasses = 10
	}

	for round := 0; round <= maxPasses; round++ {
		if cc.RequestGate != nil {
			if err := cc.RequestGate(); err != nil {
				return nil, err
			}
		}
		stream := cc.Client.Messages.NewStreaming(ctx, streamParams)
		message := anthropic.Message{}
		thinkingStarted := false
		thinkingEnded := false
		for stream.Next() {
			event := stream.Current()
			err := message.Accumulate(event)
			if err != nil {
				return nil, err
			}

			switch eventVariant := event.AsAny().(type) {
			case anthropic.ContentBlockDeltaEvent:
				switch deltaVariant := eventVariant.Delta.AsAny().(type) {
				case anthropic.ThinkingDelta:
					prefix := ""
					if !thinkingStarted {
						thinkingStarted = true
						prefix = "<think>"
					}
					chunk := models.StreamedResponse{
						AIResponse: prefix + deltaVariant.Thinking,
					}
					if err := callback(chunk); err != nil {
						return nil, err
					}
				case anthropic.TextDelta:
					prefix := ""
					if thinkingStarted && !thinkingEnded {
						thinkingEnded = true
						prefix = "</think>"
					}
					chunk := models.StreamedResponse{
						AIResponse: prefix + deltaVariant.Text,
					}
					if err := callback(chunk); err != nil {
						return nil, err
					}
				}
			}
		}

		if stream.Err() != nil {
			return nil, stream.Err()
		}

		// Check for tool calls
		if !enableTools || message.StopReason != "tool_use" {
			if len(message.Content) > 0 {
				var thinkingText, responseText string
				for _, block := range message.Content {
					switch b := block.AsAny().(type) {
					case anthropic.ThinkingBlock:
						thinkingText += b.Thinking
					case anthropic.TextBlock:
						responseText += b.Text
					}
				}
				if thinkingText != "" {
					responseText = "<think>" + thinkingText + "</think>" + responseText
				}
				return &models.AIChatResponse{
					AIResponse:   responseText,
					InputTokens:  int(message.Usage.InputTokens),
					OutputTokens: int(message.Usage.OutputTokens),
				}, nil
			}
			return nil, nil
		}

		// Execute tool calls and build results
		var toolResults []anthropic.ContentBlockParamUnion
		for _, block := range message.Content {
			if block, ok := block.AsAny().(anthropic.ToolUseBlock); ok {
				if !useMCPExecution {
					return &models.AIChatResponse{
						ToolCalls: extractToolCallsFromClaude(message.Content),
					}, nil
				}
				var arguments map[string]any
				err := json.Unmarshal(block.Input, &arguments)
				if err != nil {
					toolResults = append(toolResults, anthropic.NewToolResultBlock(block.ID,
						fmt.Sprintf("Error parsing arguments: %v", err), true))
					continue
				}

				result, err := cc.callTool(ctx, block.Name, arguments)
				if err != nil {
					toolResults = append(toolResults, anthropic.NewToolResultBlock(block.ID,
						fmt.Sprintf("Error calling tool: %v", err), true))
				} else {
					toolResults = append(toolResults, anthropic.NewToolResultBlock(block.ID, result, false))
				}
			}
		}

		// Append assistant turn + tool results and continue loop
		processedMessages = append(processedMessages, message.ToParam())
		processedMessages = append(processedMessages, anthropic.NewUserMessage(toolResults...))
		streamParams.Messages = processedMessages
	}

	return nil, fmt.Errorf("exceeded maximum tool passes (%d)", maxPasses)
}
