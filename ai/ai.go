package ai

import (
	"errors"

	"github.com/MelloB1989/karma/config"
	"github.com/MelloB1989/karma/models"
)

func (kai *KarmaAI) ChatCompletion(messages models.AIChatHistory) (*models.AIChatResponse, error) {
	kai.setBasicProperties()
	messages = kai.addUserPreprompt(messages)

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
	case Groq:
		response, err = kai.handleOpenAICompatibleChatCompletion(messages, GROQ_API, config.GetEnvRaw("GROQ_API_KEY"))
	case Sarvam:
		response, err = kai.handleOpenAICompatibleChatCompletion(messages, SARVAM_API, config.GetEnvRaw("SARVAM_API_KEY"))
	default:
		return nil, errors.New("this provider is not supported yet")
	}

	// Handle analytics and errors asynchronously after getting the response
	if response != nil {
		kai.captureResponse(messages, *response)
	}
	if err != nil {
		kai.SendErrorEvent(err)
	}

	kai.removeUserPrePrompt(messages)

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
	case Groq:
		response, err = kai.handleOpenAICompatibleChatCompletion(singleMessage, GROQ_API, config.GetEnvRaw("GROQ_API_KEY"))
	case Sarvam:
		response, err = kai.handleOpenAICompatibleChatCompletion(singleMessage, SARVAM_API, config.GetEnvRaw("SARVAM_API_KEY"))
	default:
		return nil, errors.New("this provider is not supported yet")
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
	messages = kai.addUserPreprompt(messages)

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
	case Groq:
		response, err = kai.handleOpenAICompatibleStreamCompletion(messages, callback, GROQ_API, config.GetEnvRaw("GROQ_API_KEY"))
	case Sarvam:
		response, err = kai.handleOpenAICompatibleStreamCompletion(messages, callback, SARVAM_API, config.GetEnvRaw("SARVAM_API_KEY"))
	default:
		return nil, errors.New("this provider is not supported yet")
	}

	// Handle analytics and errors asynchronously after getting the response
	if response != nil {
		kai.captureResponse(messages, *response)
	}
	if err != nil {
		kai.SendErrorEvent(err)
	}

	kai.removeUserPrePrompt(messages)

	return response, err
}

func (kai *KarmaAI) GetEmbeddings(text string) (*models.AIEmbeddingResponse, error) {
	kai.setBasicProperties()
	switch kai.Model.GetModelProvider() {
	case OpenAI:
		return kai.handleOpenAIEmbeddingGeneration(text)
	case Bedrock:
		return kai.handleBedrockEmbeddingGeneration(text)
	default:
		return nil, errors.New("this provider is not supported yet for embeddings")
	}
}
