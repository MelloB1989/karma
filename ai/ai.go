package ai

import (
	"errors"
	"fmt"

	"github.com/MelloB1989/karma/config"
	"github.com/MelloB1989/karma/internal/aws/bedrock_runtime"
	"github.com/MelloB1989/karma/models"
)

func (kai *KarmaAI) ChatCompletion(messages models.AIChatHistory) (*models.AIChatResponse, error) {
	switch kai.Model.GetModelProvider() {
	case OpenAI:
		return kai.handleOpenAIChatCompletion(messages)
	case Bedrock:
		return kai.handleBedrockChatCompletion(messages)
	case Anthropic:
		return kai.handleAnthropicChatCompletion(messages)
	case XAI:
		return kai.handleOpenAICompatibleChatCompletion(messages, XAI_API, config.GetEnvRaw("XAI_API_KEY"))
	default:
		return nil, errors.New("this model is not supported yet")
	}
}

func (kai *KarmaAI) GenerateFromSinglePrompt(prompt string) (*models.AIChatResponse, error) {
	singleMessage := models.AIChatHistory{
		Messages: []models.AIMessage{
			{
				Message: kai.UserPrePrompt + "\n" + prompt,
				Role:    models.User,
			},
		},
	}

	switch kai.Model.GetModelProvider() {
	case OpenAI:
		return kai.handleOpenAIChatCompletion(singleMessage)
	case Bedrock:
		return kai.handleBedrockSinglePrompt(singleMessage)
	case Google:
		return kai.handleGeminiSinglePrompt(prompt)
	case Anthropic:
		return kai.handleAnthropicSinglePrompt(prompt)
	case XAI:
		return kai.handleOpenAICompatibleChatCompletion(singleMessage, XAI_API, config.GetEnvRaw("XAI_API_KEY"))
	default:
		return nil, errors.New("this model is not supported yet")
	}
}

func (kai *KarmaAI) ChatCompletionStream(messages models.AIChatHistory, callback func(chunk models.StreamedResponse) error) (*models.AIChatResponse, error) {
	switch kai.Model.GetModelProvider() {
	case OpenAI:
		return kai.handleOpenAIStreamCompletion(messages, callback)
	case Bedrock:
		return kai.handleBedrockStreamCompletion(messages, callback)
	case Anthropic:
		return kai.handleAnthropicStreamCompletion(messages, callback)
	case XAI:
		return kai.handleOpenAICompatibleStreamCompletion(messages, callback, XAI_API, config.GetEnvRaw("XAI_API_KEY"))
	default:
		return nil, errors.New("this model is not supported yet")
	}
}

func (kai *KarmaAI) GetEmbeddings(text string) (*bedrock_runtime.EmbeddingResponse, error) {
	embeddings, err := bedrock_runtime.CreateEmbeddings(text, string(kai.Model))
	if err != nil {
		return nil, fmt.Errorf("failed to get Bedrock embeddings: %w", err)
	}
	return &bedrock_runtime.EmbeddingResponse{
		Embedding: embeddings,
	}, nil
}
