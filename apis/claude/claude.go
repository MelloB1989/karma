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
	"github.com/anthropics/anthropic-sdk-go/bedrock"
	"github.com/anthropics/anthropic-sdk-go/option"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
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
	// Cache controls prompt caching. The zero value caches when it is worth it;
	// see CachePolicy.
	Cache CachePolicy
}

func (cc *ClaudeClient) isThinkingModel() bool {
	return strings.Contains(string(cc.Model), "thinking")
}

// UseBedrockTransport reports whether the Anthropic client should be signed with
// SigV4 and routed to Bedrock instead of api.anthropic.com.
//
// The native Bedrock provider (ai.Bedrock) talks the Converse API and carries no
// tool definitions, so it cannot run a tool loop. Routing the *Anthropic* client
// at Bedrock keeps the entire tool-calling path — GoFunctionTools, MCP, managed
// multi-pass completions — byte-identical between local and deployed, with only
// the transport and the model string differing.
func UseBedrockTransport() bool {
	switch strings.ToLower(strings.TrimSpace(config.GetEnvRaw("KARMA_ANTHROPIC_BEDROCK"))) {
	case "1", "true", "yes":
		return true
	}
	return false
}

func NewClaudeClient(maxTokens int, model anthropic.Model, temp float64, topP float64, topK float64, systemPrompt string) *ClaudeClient {
	var opts []option.RequestOption
	if UseBedrockTransport() {
		// Credentials, region and endpoint all come from the ambient AWS config
		// (env, shared config, or the Lambda execution role). Deliberately not
		// combined with the ANTHROPIC_* options below: those set x-api-key and
		// override the base URL, which would defeat SigV4 signing.
		var loadOpts []func(*awsconfig.LoadOptions) error
		if region := config.GetEnvRaw("KARMA_ANTHROPIC_BEDROCK_REGION"); region != "" {
			loadOpts = append(loadOpts, awsconfig.WithRegion(region))
		}
		opts = append(opts, bedrock.WithLoadDefaultConfig(context.Background(), loadOpts...))
		// Same fixed ceiling as the gateway path below, for the same reason:
		// the SDK otherwise derives the request timeout from max_tokens, and a
		// tight completion cap becomes a ~2-second deadline that fails any
		// slower pass.
		opts = append(opts, option.WithRequestTimeout(5*time.Minute))
		client := anthropic.NewClient(opts...)
		return newClaudeClient(&client, maxTokens, model, temp, topP, topK, systemPrompt)
	}
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
			// The SDK derives a non-streaming request timeout FROM max_tokens —
			// 3600s scaled by max_tokens/128000 — so a 90-token cap becomes a
			// 2.5-second deadline and a 64-token cap a 1.8-second one. Small
			// completions are exactly where a caller wants tight output, and the
			// heuristic turned that thrift into random "context deadline
			// exceeded" failures on any pass slower than the cap. A fixed
			// ceiling replaces it; callers still bound real time via their ctx.
			option.WithRequestTimeout(5*time.Minute),
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
	return newClaudeClient(&client, maxTokens, model, temp, topP, topK, systemPrompt)
}

func newClaudeClient(client *anthropic.Client, maxTokens int, model anthropic.Model, temp float64, topP float64, topK float64, systemPrompt string) *ClaudeClient {
	return &ClaudeClient{
		Client:        client,
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
	cc.applyTo(&mgsParam)
	if cc.SystemPrompt != "" {
		mgsParam.System = cc.systemBlocks()
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
		AIResponse:       responseText,
		InputTokens:      int(message.Usage.InputTokens),
		OutputTokens:     int(message.Usage.OutputTokens),
		CacheReadTokens:  int(message.Usage.CacheReadInputTokens),
		CacheWriteTokens: int(message.Usage.CacheCreationInputTokens),
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
	cc.applyTo(&mgsParam)
	if cc.SystemPrompt != "" {
		mgsParam.System = cc.systemBlocks()
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
				AIResponse:       responseText,
				InputTokens:      int(message.Usage.InputTokens),
				OutputTokens:     int(message.Usage.OutputTokens),
				CacheReadTokens:  int(message.Usage.CacheReadInputTokens),
				CacheWriteTokens: int(message.Usage.CacheCreationInputTokens),
				Tokens:           int(message.Usage.InputTokens) + int(message.Usage.OutputTokens),
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
	streamParams.Temperature, streamParams.TopP, streamParams.TopK = cc.sampling()
	if cc.SystemPrompt != "" {
		streamParams.System = cc.systemBlocks()
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
