package ai

import (
	"errors"

	"github.com/MelloB1989/karma/config"
	"github.com/MelloB1989/karma/models"
)

func (kai *KarmaAI) ChatCompletion(messages models.AIChatHistory) (*models.AIChatResponse, error) {
	kai.setBasicProperties()
	m := kai.addUserPreprompt(&messages)

	var response *models.AIChatResponse
	var err error

	switch kai.Model.GetModelProvider() {
	case OpenAI:
		response, err = kai.handleOpenAIChatCompletion(m)
	case Bedrock:
		response, err = kai.handleBedrockChatCompletion(*m)
	case Google:
		response, err = kai.handleGeminiChatCompletion(m)
	case Anthropic:
		response, err = kai.handleAnthropicChatCompletion(*m)
	case XAI:
		response, err = kai.handleOpenAICompatibleChatCompletion(m, XAI_API, config.GetEnvRaw("XAI_API_KEY"))
	case Groq:
		response, err = kai.handleOpenAICompatibleChatCompletion(m, GROQ_API, config.GetEnvRaw("GROQ_API_KEY"))
	case Sarvam:
		response, err = kai.handleOpenAICompatibleChatCompletion(m, SARVAM_API, config.GetEnvRaw("SARVAM_API_KEY"))
	default:
		return nil, errors.New("this provider is not supported yet")
	}

	// Handle analytics and errors asynchronously after getting the response
	if response != nil {
		kai.captureResponse(*m, *response)
	}
	if err != nil {
		kai.SendErrorEvent(err)
	}

	kai.removeUserPrePrompt(m)

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
		response, err = kai.handleOpenAIChatCompletion(&singleMessage)
	case Bedrock:
		response, err = kai.handleBedrockSinglePrompt(singleMessage)
	case Google:
		response, err = kai.handleGeminiSinglePrompt(prompt)
	case Anthropic:
		response, err = kai.handleAnthropicSinglePrompt(prompt)
	case XAI:
		response, err = kai.handleOpenAICompatibleChatCompletion(&singleMessage, XAI_API, config.GetEnvRaw("XAI_API_KEY"))
	case Groq:
		response, err = kai.handleOpenAICompatibleChatCompletion(&singleMessage, GROQ_API, config.GetEnvRaw("GROQ_API_KEY"))
	case Sarvam:
		response, err = kai.handleOpenAICompatibleChatCompletion(&singleMessage, SARVAM_API, config.GetEnvRaw("SARVAM_API_KEY"))
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
	m := kai.addUserPreprompt(&messages)

	var response *models.AIChatResponse
	var err error

	switch kai.Model.GetModelProvider() {
	case OpenAI:
		response, err = kai.handleOpenAIStreamCompletion(m, callback)
	case Bedrock:
		response, err = kai.handleBedrockStreamCompletion(*m, callback)
	case Google:
		response, err = kai.handleGeminiStreamCompletion(m, callback)
	case Anthropic:
		response, err = kai.handleAnthropicStreamCompletion(*m, callback)
	case XAI:
		response, err = kai.handleOpenAICompatibleStreamCompletion(m, callback, XAI_API, config.GetEnvRaw("XAI_API_KEY"))
	case Groq:
		response, err = kai.handleOpenAICompatibleStreamCompletion(m, callback, GROQ_API, config.GetEnvRaw("GROQ_API_KEY"))
	case Sarvam:
		response, err = kai.handleOpenAICompatibleStreamCompletion(m, callback, SARVAM_API, config.GetEnvRaw("SARVAM_API_KEY"))
	default:
		return nil, errors.New("this provider is not supported yet")
	}

	// Handle analytics and errors asynchronously after getting the response
	if response != nil {
		kai.captureResponse(*m, *response)
	}
	if err != nil {
		kai.SendErrorEvent(err)
	}

	kai.removeUserPrePrompt(m)

	return response, err
}

func (kai *KarmaAI) ChatCompletionManaged(history *models.AIChatHistory) (*models.AIChatResponse, error) {
	if history == nil {
		return nil, errors.New("history is nil")
	}
	kai.setBasicProperties()
	kai.addUserPreprompt(history)

	var response *models.AIChatResponse
	var err error

	switch kai.Model.GetModelProvider() {
	case OpenAI:
		response, err = kai.handleOpenAIChatCompletion(history)
	case Bedrock:
		response, err = kai.handleBedrockChatCompletion(*history)
	case Google:
		response, err = kai.handleGeminiChatCompletion(history)
	case Anthropic:
		response, err = kai.handleAnthropicChatCompletion(*history)
	case XAI:
		response, err = kai.handleOpenAICompatibleChatCompletion(history, XAI_API, config.GetEnvRaw("XAI_API_KEY"))
	case Groq:
		response, err = kai.handleOpenAICompatibleChatCompletion(history, GROQ_API, config.GetEnvRaw("GROQ_API_KEY"))
	case Sarvam:
		response, err = kai.handleOpenAICompatibleChatCompletion(history, SARVAM_API, config.GetEnvRaw("SARVAM_API_KEY"))
	default:
		return nil, errors.New("this provider is not supported yet")
	}

	// Handle analytics and errors asynchronously after getting the response
	if response != nil {
		kai.captureResponse(*history, *response)
	}
	if err != nil {
		kai.SendErrorEvent(err)
	}

	kai.removeUserPrePrompt(history)

	return response, err
}

func (kai *KarmaAI) ChatCompletionStreamManaged(history *models.AIChatHistory, callback func(chunk models.StreamedResponse) error) (*models.AIChatResponse, error) {
	if history == nil {
		return nil, errors.New("history is nil")
	}
	kai.setBasicProperties()
	kai.addUserPreprompt(history)

	var response *models.AIChatResponse
	var err error

	switch kai.Model.GetModelProvider() {
	case OpenAI:
		response, err = kai.handleOpenAIStreamCompletion(history, callback)
	case Bedrock:
		response, err = kai.handleBedrockStreamCompletion(*history, callback)
	case Google:
		response, err = kai.handleGeminiStreamCompletion(history, callback)
	case Anthropic:
		response, err = kai.handleAnthropicStreamCompletion(*history, callback)
	case XAI:
		response, err = kai.handleOpenAICompatibleStreamCompletion(history, callback, XAI_API, config.GetEnvRaw("XAI_API_KEY"))
	case Groq:
		response, err = kai.handleOpenAICompatibleStreamCompletion(history, callback, GROQ_API, config.GetEnvRaw("GROQ_API_KEY"))
	case Sarvam:
		response, err = kai.handleOpenAICompatibleStreamCompletion(history, callback, SARVAM_API, config.GetEnvRaw("SARVAM_API_KEY"))
	default:
		return nil, errors.New("this provider is not supported yet")
	}

	// Handle analytics and errors asynchronously after getting the response
	if response != nil {
		kai.captureResponse(*history, *response)
	}
	if err != nil {
		kai.SendErrorEvent(err)
	}

	kai.removeUserPrePrompt(history)

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
