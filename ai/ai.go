package ai

import (
	"errors"
	"time"

	"github.com/MelloB1989/karma/config"
	"github.com/MelloB1989/karma/models"
	"github.com/MelloB1989/karma/utils"
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

// ChatCompletionManaged performs a chat completion and automatically manages the chat history.
// It appends the assistant's response (including any tool calls) to the provided history.
func (kai *KarmaAI) ChatCompletionManaged(history *models.AIChatHistory) (*models.AIChatResponse, error) {
	response, err := kai.ChatCompletion(*history)
	if err != nil {
		return nil, err
	}

	// Create the assistant message
	assistantMsg := models.AIMessage{
		Role:      models.Assistant,
		Message:   response.AIResponse,
		Timestamp: time.Now(),
		UniqueId:  utils.GenerateID(16),
	}

	// If there are tool calls, convert them to OpenAIToolCall format
	if len(response.ToolCalls) > 0 {
		toolCalls := make([]models.OpenAIToolCall, len(response.ToolCalls))
		for i, tc := range response.ToolCalls {
			toolCalls[i] = models.OpenAIToolCall{
				Index: tc.Index,
				ID:    tc.ID,
				Type:  tc.Type,
				Function: struct {
					Name      string `json:"name"`
					Arguments string `json:"arguments"`
				}{
					Name:      tc.Function.Name,
					Arguments: tc.Function.Arguments,
				},
			}
		}
		assistantMsg.ToolCalls = toolCalls
	}

	// Append the assistant message to history
	history.Messages = append(history.Messages, assistantMsg)

	return response, nil
}

// ChatCompletionStreamManaged performs a streaming chat completion and automatically manages the chat history.
// It appends the assistant's response (including any tool calls) to the provided history.
func (kai *KarmaAI) ChatCompletionStreamManaged(history *models.AIChatHistory, callback func(chunk models.StreamedResponse) error) (*models.AIChatResponse, error) {
	response, err := kai.ChatCompletionStream(*history, callback)
	if err != nil {
		return nil, err
	}

	// Create the assistant message
	assistantMsg := models.AIMessage{
		Role:      models.Assistant,
		Message:   response.AIResponse,
		Timestamp: time.Now(),
		UniqueId:  utils.GenerateID(16),
	}

	// If there are tool calls, convert them to OpenAIToolCall format
	if len(response.ToolCalls) > 0 {
		toolCalls := make([]models.OpenAIToolCall, len(response.ToolCalls))
		for i, tc := range response.ToolCalls {
			toolCalls[i] = models.OpenAIToolCall{
				Index: tc.Index,
				ID:    tc.ID,
				Type:  tc.Type,
				Function: struct {
					Name      string `json:"name"`
					Arguments string `json:"arguments"`
				}{
					Name:      tc.Function.Name,
					Arguments: tc.Function.Arguments,
				},
			}
		}
		assistantMsg.ToolCalls = toolCalls
	}

	// Append the assistant message to history
	history.Messages = append(history.Messages, assistantMsg)

	return response, nil
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
