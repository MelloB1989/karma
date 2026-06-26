package openai

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	mcp "github.com/MelloB1989/karma/ai/mcp_client"
	"github.com/MelloB1989/karma/models"
	"github.com/MelloB1989/karma/utils"
	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/shared"
)

type OpenAI struct {
	Client            openai.Client
	Model             string
	Temperature       float64
	MaxTokens         int64
	SystemMessage     string
	ExtraFields       map[string]any
	MCPManager        *mcp.Manager
	MultiMCPManager   *mcp.MultiManager
	FunctionTools     map[string]GoFunctionTool
	ReasoningEffort   *shared.ReasoningEffort
	maxToolPasses     int
	RequestGate       func() error
	RequestTimeout    time.Duration
	clientOptions     *CompatibleOptions
	clientInitialized bool
	// toolNameMap maps sanitized tool names (sent upstream) back to their
	// originals, so dotted names like "calendar.add" round-trip correctly.
	toolNameMap map[string]string
}

func NewOpenAI(model, sysmgs string, temperature float64, maxTokens int64) *OpenAI {
	return &OpenAI{
		Model:         model,
		Temperature:   temperature,
		MaxTokens:     maxTokens,
		SystemMessage: sysmgs,
		FunctionTools: make(map[string]GoFunctionTool),
	}
}

func NewOpenAICompatible(model, sysmgs string, temperature float64, maxTokens int64, baseURL, apikey string) *OpenAI {
	clientOptions := &CompatibleOptions{
		BaseURL: baseURL,
		API_Key: apikey,
	}
	return &OpenAI{
		clientOptions: clientOptions,
		Model:         model,
		Temperature:   temperature,
		MaxTokens:     maxTokens,
		SystemMessage: sysmgs,
		FunctionTools: make(map[string]GoFunctionTool),
		maxToolPasses: 5,
	}
}

// detectRawFunctionCall checks if the message content contains raw function call syntax
func detectRawFunctionCall(content string) bool {
	if content == "" {
		return false
	}
	// Look for both opening and closing function tags to reduce false positives
	hasOpening := strings.Contains(content, "<function")
	hasClosing := strings.Contains(content, "</function")
	return hasOpening || hasClosing
}

func (o *OpenAI) requestContext() (context.Context, context.CancelFunc) {
	timeout := o.RequestTimeout
	if timeout <= 0 {
		timeout = 75 * time.Second
	}
	return context.WithTimeout(context.Background(), timeout)
}

func (o *OpenAI) ApplyRequestTimeout() {
	if o.clientInitialized {
		return
	}
	o.clientInitialized = true
	var opts []CompatibleOptions
	if o.clientOptions != nil {
		opts = append(opts, *o.clientOptions)
	}
	o.Client = createClientWithTimeout(o.RequestTimeout, opts...)
}

// isToolCallParsingError checks if an error is related to tool call argument parsing
// failures from the API (e.g., Groq/OpenAI returning tool_use_failed).
func isToolCallParsingError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "tool_use_failed") ||
		strings.Contains(msg, "Failed to parse tool call arguments") ||
		strings.Contains(msg, "invalid_request_error") && strings.Contains(msg, "tool") && strings.Contains(strings.ToLower(msg), "parse")
}

// extractFailedGeneration attempts to extract the failed_generation field from a
// tool_use_failed error so the model can be asked to fix it.
func extractFailedGeneration(err error) string {
	if err == nil {
		return ""
	}
	msg := err.Error()
	idx := strings.Index(msg, "failed_generation")
	if idx == -1 {
		return ""
	}
	// Try to extract JSON from the error
	braceStart := strings.Index(msg, "{")
	if braceStart == -1 {
		return msg[idx:]
	}
	var errData map[string]any
	if jsonErr := json.Unmarshal([]byte(msg[braceStart:]), &errData); jsonErr == nil {
		if fg, ok := errData["failed_generation"].(string); ok {
			return fg
		}
	}
	return msg[idx:]
}

func (o *OpenAI) CreateChat(messages *models.AIChatHistory, enableTools bool, useMCPExecution bool) (*openai.ChatCompletion, error) {
	ctx, cancel := o.requestContext()
	defer cancel()
	params := o.buildParams(*messages, enableTools)
	var lastParsingErr error

	for range o.toolPassLimit() {
		if o.RequestGate != nil {
			if err := o.RequestGate(); err != nil {
				return nil, err
			}
		}
		chatCompletion, err := o.Client.Chat.Completions.New(ctx, params)
		if err != nil {
			if isToolCallParsingError(err) {
				failedGen := extractFailedGeneration(err)
				log.Printf("[karma] Tool call parsing error, retrying (has_failed_gen=%v)", failedGen != "")
				lastParsingErr = err
				correctionMsg := "Your previous tool call had malformed JSON arguments and could not be parsed. " +
					"Please call the tool again with properly escaped JSON. " +
					"Make sure all string values are properly escaped (no unescaped newlines, quotes, or special characters in JSON strings). " +
					"Use \\n for newlines within string values."
				if failedGen != "" {
					correctionMsg += "\nThe failed generation was: " + failedGen
				}
				params.Messages = append(params.Messages, openai.UserMessage(correctionMsg))
				continue
			}
			return nil, err
		}

		// Check for raw function calls in the response
		if len(chatCompletion.Choices) > 0 {
			content := chatCompletion.Choices[0].Message.Content
			if detectRawFunctionCall(content) {
				// LLM used wrong format - add correction message and retry
				params.Messages = append(params.Messages, chatCompletion.Choices[0].Message.ToParam())
				params.Messages = append(params.Messages, openai.UserMessage(
					"You used the wrong tool calling format. Do not output raw text like '<function(name)>()</function>'. "+
						"Instead, use the proper tool calling mechanism provided by the API. Please try again with the correct format.",
				))
				continue
			}
		}

		if !o.shouldExecuteTools(chatCompletion, enableTools, useMCPExecution) {
			if len(chatCompletion.Choices) > 0 {
				msg := chatCompletion.Choices[0].Message

				messages.Messages = append(messages.Messages, models.AIMessage{
					Role:      models.Assistant,
					Message:   msg.Content,
					Timestamp: time.Now(),
					UniqueId:  utils.GenerateID(16),
				})
			}
			return chatCompletion, nil
		}

		if len(chatCompletion.Choices) == 0 {
			return nil, fmt.Errorf("no choices in chat completion")
		}

		assistant := chatCompletion.Choices[0].Message
		params.Messages = append(params.Messages, assistant.ToParam())

		idMapping := make(map[string]string)
		for _, toolCall := range assistant.ToolCalls {
			idMapping[toolCall.ID] = generateShortToolCallID(toolCall.ID)
		}

		assistantMsg := models.AIMessage{
			Role:      models.Assistant,
			Message:   assistant.Content,
			Timestamp: time.Now(),
			UniqueId:  utils.GenerateID(16),
		}
		if len(assistant.ToolCalls) > 0 {
			assistantMsg.ToolCalls = make([]models.OpenAIToolCall, len(assistant.ToolCalls))
			for i, tc := range assistant.ToolCalls {
				assistantMsg.ToolCalls[i] = models.OpenAIToolCall{
					ID:   tc.ID,
					Type: string(tc.Type),
					Function: struct {
						Name      string `json:"name"`
						Arguments string `json:"arguments"`
					}{
						Name:      tc.Function.Name,
						Arguments: tc.Function.Arguments,
					},
				}
			}
		}
		messages.Messages = append(messages.Messages, assistantMsg)

		for _, toolCall := range assistant.ToolCalls {
			shortID := idMapping[toolCall.ID]
			var arguments map[string]any
			if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &arguments); err != nil {
				errMsg := fmt.Sprintf("Error parsing arguments: %v", err)
				params.Messages = append(params.Messages, openai.ToolMessage(errMsg, shortID))
				messages.Messages = append(messages.Messages, models.AIMessage{
					Role:       models.Tool,
					Message:    errMsg,
					ToolCallId: shortID,
					Timestamp:  time.Now(),
					UniqueId:   utils.GenerateID(16),
				})
				continue
			}

			result, err := o.callAnyTool(ctx, toolCall.Function.Name, arguments)
			if err != nil {
				errMsg := fmt.Sprintf("Error calling tool: %v", err)
				params.Messages = append(params.Messages, openai.ToolMessage(errMsg, shortID))
				messages.Messages = append(messages.Messages, models.AIMessage{
					Role:       models.Tool,
					Message:    errMsg,
					ToolCallId: shortID,
					Timestamp:  time.Now(),
					UniqueId:   utils.GenerateID(16),
				})
				continue
			}

			params.Messages = append(params.Messages, openai.ToolMessage(result, shortID))
			messages.Messages = append(messages.Messages, models.AIMessage{
				Role:       models.Tool,
				Message:    result,
				ToolCallId: shortID,
				Timestamp:  time.Now(),
				UniqueId:   utils.GenerateID(16),
			})
		}
	}
	if lastParsingErr != nil {
		return nil, fmt.Errorf("exceeded tool execution passes: tool call parsing failed: %w", lastParsingErr)
	}
	return nil, fmt.Errorf("exceeded tool execution passes")
}

func (o *OpenAI) CreateChatStream(messages *models.AIChatHistory, chunkHandler func(chunk openai.ChatCompletionChunk), enableTools bool, useMCPExecution bool) (*openai.ChatCompletion, error) {
	ctx, cancel := o.requestContext()
	defer cancel()
	params := o.buildParams(*messages, enableTools)
	var lastParsingErr error

	for range o.toolPassLimit() {
		if o.RequestGate != nil {
			if err := o.RequestGate(); err != nil {
				return nil, err
			}
		}
		acc, err := o.streamAndAccumulate(ctx, params, chunkHandler)
		if err != nil {
			if isToolCallParsingError(err) && acc != nil {
				failedGen := extractFailedGeneration(err)
				log.Printf("[karma] Tool call parsing error during streaming, retrying (has_failed_gen=%v)", failedGen != "")
				lastParsingErr = err
				correctionMsg := "Your previous tool call had malformed JSON arguments and could not be parsed. " +
					"Please call the tool again with properly escaped JSON. " +
					"Make sure all string values are properly escaped (no unescaped newlines, quotes, or special characters in JSON strings). " +
					"Use \\n for newlines within string values."
				if failedGen != "" {
					correctionMsg += "\nThe failed generation was: " + failedGen
				}
				if len(acc.Choices) > 0 && acc.Choices[0].Message.Content != "" {
					params.Messages = append(params.Messages, openai.AssistantMessage(acc.Choices[0].Message.Content))
				}
				params.Messages = append(params.Messages, openai.UserMessage(correctionMsg))
				continue
			}
			return nil, err
		}

		// Check for raw function calls in the accumulated response
		if len(acc.Choices) > 0 {
			content := acc.Choices[0].Message.Content
			if detectRawFunctionCall(content) {
				// LLM used wrong format - add correction message and retry
				params.Messages = append(params.Messages, acc.Choices[0].Message.ToParam())
				params.Messages = append(params.Messages, openai.UserMessage(
					"You used the wrong tool calling format. Do not output raw text like '<function(name)>()</function>'. "+
						"Instead, use the proper tool calling mechanism provided by the API. Please try again with the correct format.",
				))
				continue
			}
		}

		if !o.shouldExecuteTools(&acc.ChatCompletion, enableTools, useMCPExecution) {
			if len(acc.Choices) > 0 {
				msg := acc.Choices[0].Message

				messages.Messages = append(messages.Messages, models.AIMessage{
					Role:      models.Assistant,
					Message:   msg.Content,
					Timestamp: time.Now(),
					UniqueId:  utils.GenerateID(16),
				})
			}
			return &acc.ChatCompletion, nil
		}

		if len(acc.Choices) == 0 {
			return nil, fmt.Errorf("no choices in chat completion")
		}

		assistant := acc.Choices[0].Message
		params.Messages = append(params.Messages, assistant.ToParam())

		idMapping := make(map[string]string)
		for _, toolCall := range assistant.ToolCalls {
			idMapping[toolCall.ID] = generateShortToolCallID(toolCall.ID)
		}

		assistantMsg := models.AIMessage{
			Role:      models.Assistant,
			Message:   assistant.Content,
			Timestamp: time.Now(),
			UniqueId:  utils.GenerateID(16),
		}
		if len(assistant.ToolCalls) > 0 {
			assistantMsg.ToolCalls = make([]models.OpenAIToolCall, len(assistant.ToolCalls))
			for i, tc := range assistant.ToolCalls {
				assistantMsg.ToolCalls[i] = models.OpenAIToolCall{
					ID:   tc.ID,
					Type: string(tc.Type),
					Function: struct {
						Name      string `json:"name"`
						Arguments string `json:"arguments"`
					}{
						Name:      tc.Function.Name,
						Arguments: tc.Function.Arguments,
					},
				}
			}
		}
		messages.Messages = append(messages.Messages, assistantMsg)

		for _, toolCall := range assistant.ToolCalls {
			shortID := idMapping[toolCall.ID]
			var arguments map[string]any
			if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &arguments); err != nil {
				errMsg := fmt.Sprintf("Error parsing arguments: %v", err)
				params.Messages = append(params.Messages, openai.ToolMessage(errMsg, shortID))
				messages.Messages = append(messages.Messages, models.AIMessage{
					Role:       models.Tool,
					Message:    errMsg,
					ToolCallId: shortID,
					Timestamp:  time.Now(),
					UniqueId:   utils.GenerateID(16),
				})
				continue
			}

			result, err := o.callAnyTool(ctx, toolCall.Function.Name, arguments)
			if err != nil {
				errMsg := fmt.Sprintf("Error calling tool: %v", err)
				params.Messages = append(params.Messages, openai.ToolMessage(errMsg, shortID))
				messages.Messages = append(messages.Messages, models.AIMessage{
					Role:       models.Tool,
					Message:    errMsg,
					ToolCallId: shortID,
					Timestamp:  time.Now(),
					UniqueId:   utils.GenerateID(16),
				})
				continue
			}

			params.Messages = append(params.Messages, openai.ToolMessage(result, shortID))
			messages.Messages = append(messages.Messages, models.AIMessage{
				Role:       models.Tool,
				Message:    result,
				ToolCallId: shortID,
				Timestamp:  time.Now(),
				UniqueId:   utils.GenerateID(16),
			})
		}
	}
	if lastParsingErr != nil {
		return nil, fmt.Errorf("exceeded tool execution passes: tool call parsing failed: %w", lastParsingErr)
	}
	return nil, fmt.Errorf("exceeded tool execution passes")
}
