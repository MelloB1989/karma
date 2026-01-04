package openai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	mcp "github.com/MelloB1989/karma/ai/mcp_client"
	"github.com/MelloB1989/karma/models"
	"github.com/MelloB1989/karma/utils"
	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/shared"
)

type OpenAI struct {
	Client          openai.Client
	Model           string
	Temperature     float64
	MaxTokens       int64
	SystemMessage   string
	ExtraFields     map[string]any
	MCPManager      *mcp.Manager
	MultiMCPManager *mcp.MultiManager
	FunctionTools   map[string]GoFunctionTool
	ReasoningEffort *shared.ReasoningEffort
	maxToolPasses   int
}

func NewOpenAI(model, sysmgs string, temperature float64, maxTokens int64) *OpenAI {
	return &OpenAI{
		Client:        createClient(),
		Model:         model,
		Temperature:   temperature,
		MaxTokens:     maxTokens,
		SystemMessage: sysmgs,
		FunctionTools: make(map[string]GoFunctionTool),
	}
}

func NewOpenAICompatible(model, sysmgs string, temperature float64, maxTokens int64, baseURL, apikey string) *OpenAI {
	return &OpenAI{
		Client: createClient(CompatibleOptions{
			BaseURL: baseURL,
			API_Key: apikey,
		}),
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

func (o *OpenAI) CreateChat(messages *models.AIChatHistory, enableTools bool, useMCPExecution bool) (*openai.ChatCompletion, error) {
	ctx := context.TODO()
	params := o.buildParams(*messages, enableTools)

	for range o.toolPassLimit() {
		chatCompletion, err := o.Client.Chat.Completions.New(ctx, params)
		if err != nil {
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
	return nil, fmt.Errorf("exceeded tool execution passes")
}

func (o *OpenAI) CreateChatStream(messages *models.AIChatHistory, chunkHandler func(chunk openai.ChatCompletionChunk), enableTools bool, useMCPExecution bool) (*openai.ChatCompletion, error) {
	ctx := context.TODO()
	params := o.buildParams(*messages, enableTools)

	for range o.toolPassLimit() {
		acc, err := o.streamAndAccumulate(ctx, params, chunkHandler)
		if err != nil {
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
	return nil, fmt.Errorf("exceeded tool execution passes")
}
