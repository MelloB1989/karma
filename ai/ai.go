package ai

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/MelloB1989/karma/apis/aws/bedrock"
	"github.com/MelloB1989/karma/apis/claude"
	"github.com/MelloB1989/karma/apis/gemini"
	"github.com/MelloB1989/karma/internal/aws/bedrock_runtime"
	"github.com/MelloB1989/karma/internal/openai"
	"github.com/MelloB1989/karma/models"
	oai "github.com/openai/openai-go"
	"google.golang.org/genai"
)

func (kai *KarmaAI) ChatCompletion(messages models.AIChatHistory) (*models.AIChatResponse, error) {
	//Check if model is OpenAI
	if kai.Model.IsOpenAIModel() {
		o := openai.NewOpenAI(string(kai.Model), kai.Temperature, kai.MaxTokens)
		chat, err := o.CreateChat(messages)
		if err != nil {
			return nil, err
		}
		return &models.AIChatResponse{
			AIResponse: chat.Choices[0].Message.Content,
			Tokens:     int(chat.Usage.TotalTokens),
			TimeTaken:  int(chat.Created),
		}, nil
	} else if kai.Model.IsBedrockModel() {
		response, err := bedrock_runtime.InvokeBedrockConverseAPI(string(kai.Model), bedrock_runtime.CreateBedrockRequest(int(kai.MaxTokens), kai.Temperature, kai.TopP, messages, kai.SystemMessage))
		if err != nil {
			return nil, err
		}
		return &models.AIChatResponse{
			AIResponse: response.Output.Message.Content[0].Text,
			Tokens:     response.Usage.TotalTokens,
			TimeTaken:  0,
		}, nil
	} else if kai.Model.IsAnthropicModel() {
		cc := claude.NewClaudeClient(int(kai.MaxTokens), kai.Model.ToClaudeModel(), kai.Temperature, kai.TopP, kai.TopK, kai.SystemMessage)
		response, err := cc.ClaudeChatCompletion(messages)
		if err != nil {
			return nil, fmt.Errorf("failed to get response from Claude: %w", err)
		}
		return &models.AIChatResponse{
			AIResponse: response,
		}, nil
	} else {
		return nil, errors.New("This model is not supported yet.")
	}
}

func (kai *KarmaAI) GenerateFromSinglePrompt(prompt string) (*models.AIChatResponse, error) {
	//Check if model is OpenAI
	if kai.Model.IsOpenAIModel() {
		o := openai.NewOpenAI(string(kai.Model), kai.Temperature, kai.MaxTokens)
		chat, err := o.CreateChat(models.AIChatHistory{
			Messages: []models.AIMessage{
				{
					Message: kai.UserPrePrompt + " " + prompt,
					Role:    models.User,
				},
			}})
		if err != nil {
			return nil, err
		}
		return &models.AIChatResponse{
			AIResponse: chat.Choices[0].Message.Content,
			Tokens:     int(chat.Usage.TotalTokens),
			TimeTaken:  int(chat.Created),
		}, nil
	} else if kai.Model.IsBedrockModel() {
		response, err := bedrock_runtime.InvokeBedrockConverseAPI(string(kai.Model), bedrock_runtime.CreateBedrockRequest(int(kai.MaxTokens), kai.Temperature, kai.TopP, models.AIChatHistory{
			Messages: []models.AIMessage{
				{
					Message: kai.UserPrePrompt + " " + prompt,
					Role:    models.User,
				},
			}}, kai.SystemMessage))
		if err != nil {
			return nil, err
		}
		return &models.AIChatResponse{
			AIResponse: response.Output.Message.Content[0].Text,
			Tokens:     response.Usage.TotalTokens,
			TimeTaken:  0,
		}, nil
	} else if kai.Model.IsGeminiModel() {
		var response *genai.GenerateContentResponse
		var err error
		if kai.ResponseType != "" {
			response, err = gemini.RunGemini(kai.UserPrePrompt+" "+prompt, string(kai.Model), kai.SystemMessage, kai.Temperature, kai.TopP, kai.TopK, kai.MaxTokens, kai.ResponseType)
			if err != nil {
				return nil, fmt.Errorf("failed to get response from Gemini: %w", err)
			}
		} else {
			response, err = gemini.RunGemini(kai.UserPrePrompt+" "+prompt, string(kai.Model), kai.SystemMessage, kai.Temperature, kai.TopP, kai.TopK, kai.MaxTokens)
			if err != nil {
				return nil, fmt.Errorf("failed to get response from Gemini: %w", err)
			}
		}

		return &models.AIChatResponse{
			AIResponse: response.Text(),
			Tokens:     int(response.UsageMetadata.TotalTokenCount),
			TimeTaken:  int(time.Since(response.CreateTime).Milliseconds()),
		}, nil
	} else if kai.Model.IsAnthropicModel() {
		cc := claude.NewClaudeClient(int(kai.MaxTokens), kai.Model.ToClaudeModel(), kai.Temperature, kai.TopP, kai.TopK, kai.SystemMessage)
		response, err := cc.ClaudeSinglePrompt(kai.UserPrePrompt + " " + prompt)
		if err != nil {
			return nil, fmt.Errorf("failed to get response from Claude: %w", err)
		}
		return &models.AIChatResponse{
			AIResponse: response,
		}, nil
	} else {
		return nil, errors.New("This model is not supported yet.")
	}
}

func (kai *KarmaAI) GenerateFromSinglePromptWithStream(prompt string) (*models.AIChatResponse, error) {
	//Check if model is OpenAI
	if kai.Model.IsOpenAIModel() {
		o := openai.NewOpenAI(string(kai.Model), kai.Temperature, kai.MaxTokens)
		chat, err := o.CreateChat(models.AIChatHistory{
			Messages: []models.AIMessage{
				{
					Message: kai.UserPrePrompt + " " + prompt,
					Role:    models.User,
				},
			}})
		if err != nil {
			return nil, err
		}
		return &models.AIChatResponse{
			AIResponse: chat.Choices[0].Message.Content,
			Tokens:     int(chat.Usage.TotalTokens),
			TimeTaken:  int(chat.Created),
		}, nil
	} else if kai.Model.IsBedrockModel() {
		response, err := bedrock_runtime.InvokeBedrockConverseAPI(string(kai.Model), bedrock_runtime.CreateBedrockRequest(int(kai.MaxTokens), kai.Temperature, kai.TopP, models.AIChatHistory{
			Messages: []models.AIMessage{
				{
					Message: kai.UserPrePrompt + " " + prompt,
					Role:    models.User,
				},
			}}, kai.SystemMessage))
		if err != nil {
			return nil, err
		}
		return &models.AIChatResponse{
			AIResponse: response.Output.Message.Content[0].Text,
			Tokens:     response.Usage.TotalTokens,
			TimeTaken:  0,
		}, nil
	} else {
		return nil, errors.New("This model is not supported yet.")
	}
}

func (kai *KarmaAI) ChatCompletionStream(messages models.AIChatHistory, callback func(chunck models.StreamedResponse) error) (*models.AIChatResponse, error) {
	//Check if model is OpenAI
	if kai.Model.IsOpenAIModel() {
		o := openai.NewOpenAI(string(kai.Model), kai.Temperature, kai.MaxTokens)
		chunkHandler := func(chuck oai.ChatCompletionChunk) {
			callback(models.StreamedResponse{
				AIResponse: chuck.Choices[0].Delta.Content,
				TokenUsed:  int(chuck.Usage.TotalTokens),
				TimeTaken:  int(chuck.Created),
			})
		}
		chat, err := o.CreateChatStream(messages, chunkHandler)
		if err != nil {
			return nil, err
		}
		return &models.AIChatResponse{
			AIResponse: chat.Choices[0].Message.Content,
			Tokens:     int(chat.Usage.TotalTokens),
			TimeTaken:  int(chat.Created),
		}, nil
	} else if kai.Model.IsBedrockModel() {
		stream, err := bedrock.PromptModelStream(kai.processMessagesForLlamaBedrockSystemPrompt(messages), float32(kai.Temperature), float32(kai.TopP), int(kai.MaxTokens), string(kai.Model))
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
	} else if kai.Model.IsAnthropicModel() {
		cc := claude.NewClaudeClient(int(kai.MaxTokens), kai.Model.ToClaudeModel(), kai.Temperature, kai.TopP, kai.TopK, kai.SystemMessage)
		response, err := cc.ClaudeStreamCompletion(messages, callback)
		if err != nil {
			return nil, fmt.Errorf("failed to get response from Claude: %w", err)
		}
		return &models.AIChatResponse{
			AIResponse: response,
		}, nil
	} else {
		return nil, errors.New("This model is not supported yet.")
	}
}

func (kai *KarmaAI) GetEmbeddings(text string) (*bedrock_runtime.EmbeddingResponse, error) {
	modelID := string(kai.Model)
	embeddings, err := bedrock_runtime.CreateEmbeddings(text, modelID)
	if err != nil {
		return nil, fmt.Errorf("failed to get Bedrock embeddings: %w", err)
	}
	return &bedrock_runtime.EmbeddingResponse{
		Embedding: embeddings,
	}, nil
}
