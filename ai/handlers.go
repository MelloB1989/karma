package ai

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/MelloB1989/karma/apis/aws/bedrock"
	"github.com/MelloB1989/karma/apis/claude"
	"github.com/MelloB1989/karma/apis/gemini"
	"github.com/MelloB1989/karma/internal/aws/bedrock_runtime"
	"github.com/MelloB1989/karma/internal/openai"
	"github.com/MelloB1989/karma/models"
	"github.com/anthropics/anthropic-sdk-go"
	oai "github.com/openai/openai-go/v2"
	"google.golang.org/genai"
)

func (kai *KarmaAI) handleOpenAIChatCompletion(messages models.AIChatHistory) (*models.AIChatResponse, error) {
	start := time.Now()
	o := openai.NewOpenAI(kai.Model.GetModelString(), kai.SystemMessage, float64(kai.Temperature), int64(kai.MaxTokens))
	kai.configureOpenaiClientForMCP(o)
	o.ExtraFields = kai.Features.optionalFields
	chat, err := o.CreateChat(messages, kai.ToolsEnabled)
	if err != nil {
		return nil, err
	}
	if len(chat.Choices) == 0 {
		return nil, errors.New("No response from OpenAI")
	}
	res := &models.AIChatResponse{
		AIResponse:   chat.Choices[0].Message.Content,
		Tokens:       int(chat.Usage.TotalTokens),
		InputTokens:  int(chat.Usage.PromptTokens),
		OutputTokens: int(chat.Usage.CompletionTokens),
		TimeTaken:    int(time.Since(start).Milliseconds()),
	}
	return res, nil
}

func (kai *KarmaAI) handleOpenAICompatibleChatCompletion(messages models.AIChatHistory, base_url string, apikey string) (*models.AIChatResponse, error) {
	start := time.Now()
	o := openai.NewOpenAICompatible(kai.Model.GetModelString(), kai.SystemMessage, float64(kai.Temperature), int64(kai.MaxTokens), base_url, apikey)
	kai.configureOpenaiClientForMCP(o)
	o.ExtraFields = kai.Features.optionalFields
	chat, err := o.CreateChat(messages, kai.ToolsEnabled)
	if err != nil {
		return nil, err
	}
	if len(chat.Choices) == 0 {
		return nil, errors.New("No response from OpenAI")
	}
	return &models.AIChatResponse{
		AIResponse:   chat.Choices[0].Message.Content,
		Tokens:       int(chat.Usage.TotalTokens),
		InputTokens:  int(chat.Usage.PromptTokens),
		OutputTokens: int(chat.Usage.CompletionTokens),
		TimeTaken:    int(time.Since(start).Milliseconds()),
	}, nil
}

func (kai *KarmaAI) handleBedrockChatCompletion(messages models.AIChatHistory) (*models.AIChatResponse, error) {
	start := time.Now()
	response, err := bedrock_runtime.InvokeBedrockConverseAPI(
		kai.Model.GetModelString(),
		bedrock_runtime.CreateBedrockRequest(int(kai.MaxTokens), float64(kai.Temperature), float64(kai.TopP), messages, kai.SystemMessage),
	)
	if err != nil {
		return nil, err
	}
	if len(response.Output.Message.Content) == 0 {
		return nil, errors.New("No response from Bedrock")
	}
	return &models.AIChatResponse{
		AIResponse:   response.Output.Message.Content[0].Text,
		Tokens:       response.Usage.TotalTokens,
		InputTokens:  response.Usage.InputTokens,
		OutputTokens: response.Usage.OutputTokens,
		TimeTaken:    int(time.Since(start).Milliseconds()),
	}, nil
}

func (kai *KarmaAI) handleAnthropicChatCompletion(messages models.AIChatHistory) (*models.AIChatResponse, error) {
	cc := claude.NewClaudeClient(int(kai.MaxTokens), anthropic.Model(kai.Model.GetModelString()), float64(kai.Temperature), float64(kai.TopP), float64(kai.TopK), kai.SystemMessage)
	kai.configureClaudeClientForMCP(cc)
	start := time.Now()
	response, err := cc.ClaudeChatCompletion(messages, kai.ToolsEnabled)
	if err != nil {
		return nil, fmt.Errorf("failed to get response from Claude: %w", err)
	}
	response.TimeTaken = int(time.Since(start).Milliseconds())
	return response, nil
}

func (kai *KarmaAI) handleBedrockSinglePrompt(messages models.AIChatHistory) (*models.AIChatResponse, error) {
	start := time.Now()
	response, err := bedrock_runtime.InvokeBedrockConverseAPI(
		kai.Model.GetModelString(),
		bedrock_runtime.CreateBedrockRequest(int(kai.MaxTokens), float64(kai.Temperature), float64(kai.TopP), messages, kai.SystemMessage),
	)
	if err != nil {
		return nil, err
	}
	if len(response.Output.Message.Content) == 0 {
		return nil, errors.New("No response from Bedrock")
	}
	return &models.AIChatResponse{
		AIResponse:   response.Output.Message.Content[0].Text,
		Tokens:       response.Usage.TotalTokens,
		InputTokens:  response.Usage.InputTokens,
		OutputTokens: response.Usage.OutputTokens,
		TimeTaken:    int(time.Since(start).Milliseconds()),
	}, nil
}

func (kai *KarmaAI) handleGeminiSinglePrompt(prompt string) (*models.AIChatResponse, error) {
	fullPrompt := kai.UserPrePrompt + "\n" + prompt
	var response *genai.GenerateContentResponse
	var err error

	if kai.ResponseType != "" {
		response, err = gemini.RunGemini(fullPrompt, kai.Model.GetModelString(), kai.SystemMessage, float64(kai.Temperature), float64(kai.TopP), float64(kai.TopK), int64(kai.MaxTokens), kai.ResponseType)
	} else {
		response, err = gemini.RunGemini(fullPrompt, kai.Model.GetModelString(), kai.SystemMessage, float64(kai.Temperature), float64(kai.TopP), float64(kai.TopK), int64(kai.MaxTokens))
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get response from Gemini: %w", err)
	}

	return &models.AIChatResponse{
		AIResponse:   response.Text(),
		Tokens:       int(response.UsageMetadata.TotalTokenCount),
		TimeTaken:    int(time.Since(response.CreateTime).Milliseconds()),
		InputTokens:  int(response.UsageMetadata.PromptTokenCount),
		OutputTokens: int(response.UsageMetadata.TotalTokenCount) - int(response.UsageMetadata.PromptTokenCount),
	}, nil
}

func (kai *KarmaAI) handleAnthropicSinglePrompt(prompt string) (*models.AIChatResponse, error) {
	cc := claude.NewClaudeClient(int(kai.MaxTokens), anthropic.Model(kai.Model.GetModelString()), float64(kai.Temperature), float64(kai.TopP), float64(kai.TopK), kai.SystemMessage)
	if len(kai.MCPTools) > 0 {
		log.Println("MCPTools are not supported for Single Prompts, please create a conversation!")
	}
	start := time.Now()
	response, err := cc.ClaudeSinglePrompt(kai.UserPrePrompt + "\n" + prompt)
	if err != nil {
		return nil, fmt.Errorf("failed to get response from Claude: %w", err)
	}
	response.TimeTaken = int(time.Since(start).Milliseconds())
	return response, nil
}

func (kai *KarmaAI) handleOpenAIStreamCompletion(messages models.AIChatHistory, callback func(chunk models.StreamedResponse) error) (*models.AIChatResponse, error) {
	start := time.Now()
	o := openai.NewOpenAI(kai.Model.GetModelString(), kai.SystemMessage, float64(kai.Temperature), int64(kai.MaxTokens))
	kai.configureOpenaiClientForMCP(o)
	o.ExtraFields = kai.Features.optionalFields

	chunkHandler := func(chuck oai.ChatCompletionChunk) {
		if len(chuck.Choices) == 0 {
			log.Println("No choices in chunk")
			return
		}
		callback(models.StreamedResponse{
			AIResponse: chuck.Choices[0].Delta.Content,
			TokenUsed:  int(chuck.Usage.TotalTokens),
			TimeTaken:  int(chuck.Created),
		})
	}
	chat, err := o.CreateChatStream(messages, chunkHandler, kai.ToolsEnabled)
	if err != nil {
		return nil, err
	}
	if len(chat.Choices) == 0 {
		return nil, errors.New("No response from OpenAI")
	}
	return &models.AIChatResponse{
		AIResponse:   chat.Choices[0].Message.Content,
		Tokens:       int(chat.Usage.TotalTokens),
		TimeTaken:    int(time.Since(start).Milliseconds()),
		InputTokens:  int(chat.Usage.PromptTokens),
		OutputTokens: int(chat.Usage.CompletionTokens),
	}, nil
}

func (kai *KarmaAI) handleOpenAICompatibleStreamCompletion(messages models.AIChatHistory, callback func(chunk models.StreamedResponse) error, base_url string, apikey string) (*models.AIChatResponse, error) {
	start := time.Now()
	o := openai.NewOpenAICompatible(kai.Model.GetModelString(), kai.SystemMessage, float64(kai.Temperature), int64(kai.MaxTokens), base_url, apikey)
	kai.configureOpenaiClientForMCP(o)
	o.ExtraFields = kai.Features.optionalFields
	chunkHandler := func(chunk oai.ChatCompletionChunk) {
		if len(chunk.Choices) == 0 {
			log.Println("No choices in chunk")
			return
		}
		callback(models.StreamedResponse{
			AIResponse: chunk.Choices[0].Delta.Content,
			TokenUsed:  int(chunk.Usage.TotalTokens),
			TimeTaken:  int(chunk.Created),
		})
	}
	chat, err := o.CreateChatStream(messages, chunkHandler, kai.ToolsEnabled)
	if err != nil {
		return nil, err
	}
	if len(chat.Choices) == 0 {
		return nil, errors.New("No response from OpenAI")
	}
	return &models.AIChatResponse{
		AIResponse:   chat.Choices[0].Message.Content,
		Tokens:       int(chat.Usage.TotalTokens),
		TimeTaken:    int(time.Since(start).Milliseconds()),
		InputTokens:  int(chat.Usage.PromptTokens),
		OutputTokens: int(chat.Usage.CompletionTokens),
	}, nil
}

func (kai *KarmaAI) handleBedrockStreamCompletion(messages models.AIChatHistory, callback func(chunk models.StreamedResponse) error) (*models.AIChatResponse, error) {
	stream, err := bedrock.PromptModelStream(
		kai.processMessagesForLlamaBedrockSystemPrompt(messages),
		float32(kai.Temperature),
		float32(kai.TopP),
		int(kai.MaxTokens),
		kai.Model.GetModelString(),
	)
	if err != nil {
		return nil, err
	}

	var response string
	var totalTokens int
	generationStart := time.Now()

	chunkHandler := func(ctx context.Context, part bedrock.Generation) error {
		response += string(part.Generation)
		totalTokens += part.GenerationTokenCount
		return callback(models.StreamedResponse{
			AIResponse: part.Generation,
			TokenUsed:  part.GenerationTokenCount,
			TimeTaken:  -1,
		})
	}

	_, err = bedrock.ProcessStreamingOutput(stream, chunkHandler)
	if err != nil {
		return nil, err
	}

	return &models.AIChatResponse{
		AIResponse: response,
		Tokens:     totalTokens,
		TimeTaken:  int(time.Since(generationStart).Milliseconds()),
	}, nil
}

func (kai *KarmaAI) handleAnthropicStreamCompletion(messages models.AIChatHistory, callback func(chunk models.StreamedResponse) error) (*models.AIChatResponse, error) {
	start := time.Now()
	cc := claude.NewClaudeClient(int(kai.MaxTokens), anthropic.Model(kai.Model.GetModelString()), float64(kai.Temperature), float64(kai.TopP), float64(kai.TopK), kai.SystemMessage)
	kai.configureClaudeClientForMCP(cc)
	response, err := cc.ClaudeStreamCompletionWithTools(messages, callback, kai.ToolsEnabled)
	if err != nil {
		return nil, fmt.Errorf("failed to get response from Claude: %w", err)
	}
	response.TimeTaken = int(time.Since(start).Milliseconds())
	return response, nil
}
