package openai

import (
	"context"
	"encoding/json"
	"fmt"

	mcp "github.com/MelloB1989/karma/ai/mcp_client"
	"github.com/MelloB1989/karma/models"
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

func (o *OpenAI) CreateChat(messages models.AIChatHistory, enableTools bool, useMCPExecution bool) (*openai.ChatCompletion, error) {
	ctx := context.TODO()
	params := o.buildParams(messages, enableTools)
	for range o.toolPassLimit() {
		chatCompletion, err := o.Client.Chat.Completions.New(ctx, params)
		if err != nil {
			return nil, err
		}
		if !o.shouldExecuteTools(chatCompletion, enableTools, useMCPExecution) {
			return chatCompletion, nil
		}
		params.Messages = append(params.Messages, chatCompletion.Choices[0].Message.ToParam())
		idMapping := make(map[string]string)
		for _, toolCall := range chatCompletion.Choices[0].Message.ToolCalls {
			idMapping[toolCall.ID] = generateShortToolCallID(toolCall.ID)
		}
		for _, toolCall := range chatCompletion.Choices[0].Message.ToolCalls {
			shortID := idMapping[toolCall.ID]
			var arguments map[string]any
			if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &arguments); err != nil {
				params.Messages = append(params.Messages, openai.ToolMessage(fmt.Sprintf("Error parsing arguments: %v", err), shortID))
				continue
			}
			result, err := o.callAnyTool(ctx, toolCall.Function.Name, arguments)
			if err != nil {
				params.Messages = append(params.Messages, openai.ToolMessage(fmt.Sprintf("Error calling tool: %v", err), shortID))
				continue
			}
			params.Messages = append(params.Messages, openai.ToolMessage(result, shortID))
		}
	}
	return nil, fmt.Errorf("exceeded tool execution passes")
}

func (o *OpenAI) CreateChatStream(messages models.AIChatHistory, chunkHandler func(chunk openai.ChatCompletionChunk), enableTools bool, useMCPExecution bool) (*openai.ChatCompletion, error) {
	ctx := context.TODO()
	params := o.buildParams(messages, enableTools)
	for range o.toolPassLimit() {
		acc, err := o.streamAndAccumulate(ctx, params, chunkHandler)
		if err != nil {
			return nil, err
		}
		if !o.shouldExecuteTools(&acc.ChatCompletion, enableTools, useMCPExecution) {
			return &acc.ChatCompletion, nil
		}
		params.Messages = append(params.Messages, acc.Choices[0].Message.ToParam())
		idMapping := make(map[string]string)
		for _, toolCall := range acc.Choices[0].Message.ToolCalls {
			idMapping[toolCall.ID] = generateShortToolCallID(toolCall.ID)
		}
		for _, toolCall := range acc.Choices[0].Message.ToolCalls {
			shortID := idMapping[toolCall.ID]
			var arguments map[string]any
			if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &arguments); err != nil {
				params.Messages = append(params.Messages, openai.ToolMessage(fmt.Sprintf("Error parsing arguments: %v", err), shortID))
				continue
			}
			result, err := o.callAnyTool(ctx, toolCall.Function.Name, arguments)
			if err != nil {
				params.Messages = append(params.Messages, openai.ToolMessage(fmt.Sprintf("Error calling tool: %v", err), shortID))
				continue
			}
			params.Messages = append(params.Messages, openai.ToolMessage(result, shortID))
		}
	}
	return nil, fmt.Errorf("exceeded tool execution passes")
}
