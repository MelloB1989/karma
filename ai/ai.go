package ai

import (
	"errors"
	"fmt"

	"github.com/MelloB1989/karma/config"
	"github.com/MelloB1989/karma/internal/aws/bedrock_runtime"
	"github.com/MelloB1989/karma/models"
)

func (kai *KarmaAI) ChatCompletion(messages models.AIChatHistory) (*models.AIChatResponse, error) {
	kai.setBasicProperties()

	var response *models.AIChatResponse
	var err error

	switch kai.Model.GetModelProvider() {
	case OpenAI:
		response, err = kai.handleOpenAIChatCompletion(messages)
	case Bedrock:
		response, err = kai.handleBedrockChatCompletion(messages)
	case Anthropic:
		response, err = kai.handleAnthropicChatCompletion(messages)
	case XAI:
		response, err = kai.handleOpenAICompatibleChatCompletion(messages, XAI_API, config.GetEnvRaw("XAI_API_KEY"))
	default:
		return nil, errors.New("this model is not supported yet")
	}

	// Handle analytics and errors asynchronously after getting the response
	if response != nil {
		kai.captureResponse(messages, *response)
	}
	if err != nil {
		kai.SendErrorEvent(err)
	}

	return response, err
}

func (kai *KarmaAI) GenerateFromSinglePrompt(prompt string) (*models.AIChatResponse, error) {
	kai.setBasicProperties()

	singleMessage := models.AIChatHistory{
		Messages: []models.AIMessage{
			{
				Message: kai.UserPrePrompt + "\n" + prompt,
				Role:    models.User,
			},
		},
	}

	var response *models.AIChatResponse
	var err error

	switch kai.Model.GetModelProvider() {
	case OpenAI:
		response, err = kai.handleOpenAIChatCompletion(singleMessage)
	case Bedrock:
		response, err = kai.handleBedrockSinglePrompt(singleMessage)
	case Google:
		response, err = kai.handleGeminiSinglePrompt(prompt)
	case Anthropic:
		response, err = kai.handleAnthropicSinglePrompt(prompt)
	case XAI:
		response, err = kai.handleOpenAICompatibleChatCompletion(singleMessage, XAI_API, config.GetEnvRaw("XAI_API_KEY"))
	default:
		return nil, errors.New("this model is not supported yet")
	}

	// Handle analytics and errors asynchronously after getting the response
	if response != nil {
		kai.captureResponse(singleMessage, *response)
	}
	if err != nil {
		kai.SendErrorEvent(err)
	}

	return response, err
}

func (kai *KarmaAI) ChatCompletionStream(messages models.AIChatHistory, callback func(chunk models.StreamedResponse) error) (*models.AIChatResponse, error) {
	kai.setBasicProperties()

	var response *models.AIChatResponse
	var err error

	switch kai.Model.GetModelProvider() {
	case OpenAI:
		response, err = kai.handleOpenAIStreamCompletion(messages, callback)
	case Bedrock:
		response, err = kai.handleBedrockStreamCompletion(messages, callback)
	case Anthropic:
		response, err = kai.handleAnthropicStreamCompletion(messages, callback)
	case XAI:
		response, err = kai.handleOpenAICompatibleStreamCompletion(messages, callback, XAI_API, config.GetEnvRaw("XAI_API_KEY"))
	default:
		return nil, errors.New("this model is not supported yet")
	}

	// Handle analytics and errors asynchronously after getting the response
	if response != nil {
		kai.captureResponse(messages, *response)
	}
	if err != nil {
		kai.SendErrorEvent(err)
	}

	return response, err
}

func (kai *KarmaAI) GetEmbeddings(text string) (*bedrock_runtime.EmbeddingResponse, error) {
	embeddings, err := bedrock_runtime.CreateEmbeddings(text, kai.Model.GetModelString())
	if err != nil {
		return nil, fmt.Errorf("failed to get Bedrock embeddings: %w", err)
	}
	return &bedrock_runtime.EmbeddingResponse{
		Embedding: embeddings,
	}, nil
}
