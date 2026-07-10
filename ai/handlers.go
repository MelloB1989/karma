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
	"github.com/MelloB1989/karma/internal/openai"
	"github.com/MelloB1989/karma/models"
	"github.com/anthropics/anthropic-sdk-go"
	oai "github.com/openai/openai-go/v3"
	"google.golang.org/genai"
)

func (kai *KarmaAI) handleOpenAIChatCompletion(messages *models.AIChatHistory) (*models.AIChatResponse, error) {
	start := time.Now()
	o := openai.NewOpenAI(kai.Model.GetModelString(), kai.SystemMessage, float64(kai.Temperature), int64(kai.MaxTokens))
	kai.configureOpenAIClient(o)

	chat, err := o.CreateChat(messages, kai.ToolsEnabled, kai.UseMCPExecution)
	if err != nil {
		return nil, err
	}

	res, err := buildOpenAIChatResponse(chat, start)
	return finalizeOpenAIResponse(res, err, o)
}

func (kai *KarmaAI) handleOpenAICompatibleChatCompletion(messages *models.AIChatHistory, base_url string, apikey string) (*models.AIChatResponse, error) {
	start := time.Now()
	o := openai.NewOpenAICompatible(kai.Model.GetModelString(), kai.SystemMessage, float64(kai.Temperature), int64(kai.MaxTokens), base_url, apikey)
	kai.configureOpenAIClient(o)

	chat, err := o.CreateChat(messages, kai.ToolsEnabled, kai.UseMCPExecution)
	if err != nil {
		return nil, err
	}

	res, err := buildOpenAIChatResponse(chat, start)
	return finalizeOpenAIResponse(res, err, o)
}

func (kai *KarmaAI) handleBedrockChatCompletion(messages models.AIChatHistory) (*models.AIChatResponse, error) {
	start := time.Now()
	if err := kai.enforceRateLimit(); err != nil {
		return nil, err
	}
	ctx := context.Background()
	if kai.RequestTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(context.Background(), kai.RequestTimeout)
		defer cancel()
	}
	response, err := bedrock.Converse(ctx, kai.bedrockConverseParams(messages))
	if err != nil {
		return nil, err
	}
	if response.Text == "" {
		return nil, errors.New("No response from Bedrock")
	}
	return &models.AIChatResponse{
		AIResponse:   response.Text,
		Tokens:       response.TotalTokens,
		InputTokens:  response.InputTokens,
		OutputTokens: response.OutputTokens,
		TimeTaken:    int(time.Since(start).Milliseconds()),
	}, nil
}

// bedrockConverseParams builds the shared Converse parameters from the KarmaAI
// configuration.
func (kai *KarmaAI) bedrockConverseParams(messages models.AIChatHistory) bedrock.ConverseParams {
	return bedrock.ConverseParams{
		ModelID:     kai.Model.GetModelString(),
		System:      kai.SystemMessage,
		History:     messages,
		MaxTokens:   int(kai.MaxTokens),
		Temperature: float32(kai.Temperature),
		TopP:        float32(kai.TopP),
		TopK:        int(kai.TopK),
		APIKey:      kai.BedrockAPIKey,
		Region:      kai.BedrockRegion,
	}
}

func (kai *KarmaAI) handleAnthropicChatCompletion(messages models.AIChatHistory) (*models.AIChatResponse, error) {
	cc := claude.NewClaudeClient(int(kai.MaxTokens), anthropic.Model(kai.Model.GetModelString()), float64(kai.Temperature), float64(kai.TopP), float64(kai.TopK), kai.SystemMessage)
	kai.configureClaudeClientForMCP(cc)
	cc.RequestGate = kai.enforceRateLimit
	start := time.Now()
	response, err := cc.ClaudeChatCompletion(messages, kai.ToolsEnabled, kai.UseMCPExecution)
	if err != nil {
		return nil, fmt.Errorf("failed to get response from Claude: %w", err)
	}
	response.TimeTaken = int(time.Since(start).Milliseconds())
	return response, nil
}

func (kai *KarmaAI) handleBedrockSinglePrompt(messages models.AIChatHistory) (*models.AIChatResponse, error) {
	return kai.handleBedrockChatCompletion(messages)
}

func (kai *KarmaAI) handleGeminiSinglePrompt(prompt string) (*models.AIChatResponse, error) {
	fullPrompt := kai.UserPrePrompt + "\n" + prompt
	var response *genai.GenerateContentResponse
	var err error

	if err := kai.enforceRateLimit(); err != nil {
		return nil, err
	}
	ctx := context.Background()
	if kai.RequestTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(context.Background(), kai.RequestTimeout)
		defer cancel()
	}
	if kai.ResponseType != "" {
		response, err = gemini.RunGeminiWithContext(ctx, fullPrompt, kai.Model.GetModelString(), kai.SystemMessage, float64(kai.Temperature), float64(kai.TopP), float64(kai.TopK), int64(kai.MaxTokens), kai.ResponseType)
	} else {
		response, err = gemini.RunGeminiWithContext(ctx, fullPrompt, kai.Model.GetModelString(), kai.SystemMessage, float64(kai.Temperature), float64(kai.TopP), float64(kai.TopK), int64(kai.MaxTokens))
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

func (kai *KarmaAI) handleGeminiChatCompletion(messages *models.AIChatHistory) (*models.AIChatResponse, error) {
	start := time.Now()
	g, err := kai.createGeminiClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create Gemini client: %w", err)
	}

	kai.configureGeminiClient(g)

	chat, err := g.CreateChat(messages, kai.ToolsEnabled, kai.UseMCPExecution)
	if err != nil {
		return nil, err
	}

	return buildGeminiChatResponse(chat, start)
}

func (kai *KarmaAI) handleGeminiStreamCompletion(messages *models.AIChatHistory, callback func(chunk models.StreamedResponse) error) (*models.AIChatResponse, error) {
	start := time.Now()
	g, err := kai.createGeminiClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create Gemini client: %w", err)
	}

	kai.configureGeminiClient(g)

	chunkHandler := createGeminiChunkHandler(callback)
	chat, err := g.CreateChatStream(messages, chunkHandler, kai.ToolsEnabled, kai.UseMCPExecution)
	if err != nil {
		return nil, err
	}

	return buildGeminiChatResponse(chat, start)
}

func (kai *KarmaAI) handleAnthropicSinglePrompt(prompt string) (*models.AIChatResponse, error) {
	cc := claude.NewClaudeClient(int(kai.MaxTokens), anthropic.Model(kai.Model.GetModelString()), float64(kai.Temperature), float64(kai.TopP), float64(kai.TopK), kai.SystemMessage)
	cc.RequestGate = kai.enforceRateLimit
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

func (kai *KarmaAI) handleOpenAIStreamCompletion(messages *models.AIChatHistory, callback func(chunk models.StreamedResponse) error) (*models.AIChatResponse, error) {
	start := time.Now()
	o := openai.NewOpenAI(kai.Model.GetModelString(), kai.SystemMessage, float64(kai.Temperature), int64(kai.MaxTokens))
	kai.configureOpenAIClient(o)

	chunkHandler := createOpenAIChunkHandler(callback)
	chat, err := o.CreateChatStream(messages, chunkHandler, kai.ToolsEnabled, kai.UseMCPExecution)
	if err != nil {
		return nil, err
	}

	res, err := buildOpenAIChatResponse(chat, start)
	return finalizeOpenAIResponse(res, err, o)
}

func (kai *KarmaAI) handleOpenAICompatibleStreamCompletion(messages *models.AIChatHistory, callback func(chunk models.StreamedResponse) error, base_url string, apikey string) (*models.AIChatResponse, error) {
	start := time.Now()
	o := openai.NewOpenAICompatible(kai.Model.GetModelString(), kai.SystemMessage, float64(kai.Temperature), int64(kai.MaxTokens), base_url, apikey)
	kai.configureOpenAIClient(o)

	chunkHandler := createOpenAIChunkHandler(callback)
	chat, err := o.CreateChatStream(messages, chunkHandler, kai.ToolsEnabled, kai.UseMCPExecution)
	if err != nil {
		return nil, err
	}

	res, err := buildOpenAIChatResponse(chat, start)
	return finalizeOpenAIResponse(res, err, o)
}

func (kai *KarmaAI) handleBedrockStreamCompletion(messages models.AIChatHistory, callback func(chunk models.StreamedResponse) error) (*models.AIChatResponse, error) {
	if err := kai.enforceRateLimit(); err != nil {
		return nil, err
	}
	ctx := context.Background()
	if kai.RequestTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(context.Background(), kai.RequestTimeout)
		defer cancel()
	}
	generationStart := time.Now()

	onText := func(text string) error {
		return callback(models.StreamedResponse{
			AIResponse: text,
			TimeTaken:  -1,
		})
	}

	result, err := bedrock.ConverseStream(ctx, kai.bedrockConverseParams(messages), onText)
	if err != nil {
		return nil, err
	}

	return &models.AIChatResponse{
		AIResponse:   result.Text,
		Tokens:       result.TotalTokens,
		InputTokens:  result.InputTokens,
		OutputTokens: result.OutputTokens,
		TimeTaken:    int(time.Since(generationStart).Milliseconds()),
	}, nil
}

func (kai *KarmaAI) handleAnthropicStreamCompletion(messages models.AIChatHistory, callback func(chunk models.StreamedResponse) error) (*models.AIChatResponse, error) {
	start := time.Now()
	cc := claude.NewClaudeClient(int(kai.MaxTokens), anthropic.Model(kai.Model.GetModelString()), float64(kai.Temperature), float64(kai.TopP), float64(kai.TopK), kai.SystemMessage)
	kai.configureClaudeClientForMCP(cc)
	cc.RequestGate = kai.enforceRateLimit
	response, err := cc.ClaudeStreamCompletionWithTools(messages, callback, kai.ToolsEnabled, kai.UseMCPExecution)
	if err != nil {
		return nil, fmt.Errorf("failed to get response from Claude: %w", err)
	}
	response.TimeTaken = int(time.Since(start).Milliseconds())
	return response, nil
}

func (kai *KarmaAI) handleOpenAIEmbeddingGeneration(text string) (*models.AIEmbeddingResponse, error) {
	if err := kai.enforceRateLimit(); err != nil {
		return nil, err
	}
	ctx := context.Background()
	if kai.RequestTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(context.Background(), kai.RequestTimeout)
		defer cancel()
	}
	embeddings, err := openai.GenerateEmbeddingsWithContext(ctx, text, string(kai.Model.BaseModel))
	if err != nil {
		return nil, fmt.Errorf("failed to generate embeddings: %w", err)
	}

	// Extract the embedding vector from the first data element
	var embeddingVector []float64
	if len(embeddings.Data) > 0 {
		embeddingVector = embeddings.Data[0].Embedding
	}

	return &models.AIEmbeddingResponse{
		Embeddings: embeddingVector,
		Usage: struct {
			PromptTokens int `json:"prompt_tokens"`
			TotalTokens  int `json:"total_tokens"`
		}{
			PromptTokens: int(embeddings.Usage.PromptTokens),
			TotalTokens:  int(embeddings.Usage.TotalTokens),
		},
	}, nil
}

func (kai *KarmaAI) handleBedrockEmbeddingGeneration(text string) (*models.AIEmbeddingResponse, error) {
	if err := kai.enforceRateLimit(); err != nil {
		return nil, err
	}
	embeddings, err := bedrock.CreateEmbeddings(
		context.Background(),
		text,
		kai.Model.GetModelString(),
		bedrock.ClientOptions{Region: kai.BedrockRegion, APIKey: kai.BedrockAPIKey},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get Bedrock embeddings: %w", err)
	}
	// Convert the embeddings to a slice of float64
	embeddingVector := make([]float64, 0, len(embeddings))
	for _, v := range embeddings {
		embeddingVector = append(embeddingVector, float64(v))
	}

	// TODO: Include usage details.
	return &models.AIEmbeddingResponse{
		Embeddings: embeddingVector,
	}, nil
}

func (kai *KarmaAI) configureOpenAIClient(o *openai.OpenAI) {
	kai.configureOpenaiClientForMCP(o)
	o.ExtraFields = kai.Features.optionalFields
	o.ReasoningEffort = kai.ReasoningEffort
	o.RequestGate = kai.enforceRateLimit
	o.RequestTimeout = kai.RequestTimeout
	o.ApplyRequestTimeout()
}

func buildToolCallsFromOpenAI(toolCalls []oai.ChatCompletionMessageToolCallUnion) []models.ToolCall {
	result := make([]models.ToolCall, len(toolCalls))
	for i, tc := range toolCalls {
		result[i] = models.ToolCall{
			ID:   tc.ID,
			Type: string(tc.Type),
			Function: models.ToolCallFunction{
				Name:      tc.Function.Name,
				Arguments: tc.Function.Arguments,
			},
		}
	}
	return result
}

// finalizeOpenAIResponse restores any sanitized tool-call names (e.g. dotted
// KARMAX names rewritten for the OpenAI/Codex name constraint) back to their
// originals so callers dispatching on the name find the right tool. No-op when
// no names were rewritten.
func finalizeOpenAIResponse(res *models.AIChatResponse, err error, o *openai.OpenAI) (*models.AIChatResponse, error) {
	if res != nil {
		for i := range res.ToolCalls {
			res.ToolCalls[i].Function.Name = o.RestoreToolName(res.ToolCalls[i].Function.Name)
		}
	}
	return res, err
}

func buildOpenAIChatResponse(chat *oai.ChatCompletion, startTime time.Time) (*models.AIChatResponse, error) {
	if len(chat.Choices) == 0 {
		return nil, errors.New("No response from OpenAI")
	}

	res := &models.AIChatResponse{
		AIResponse:   chat.Choices[0].Message.Content,
		Tokens:       int(chat.Usage.TotalTokens),
		InputTokens:  int(chat.Usage.PromptTokens),
		OutputTokens: int(chat.Usage.CompletionTokens),
		TimeTaken:    int(time.Since(startTime).Milliseconds()),
	}

	if len(chat.Choices[0].Message.ToolCalls) > 0 {
		res.ToolCalls = buildToolCallsFromOpenAI(chat.Choices[0].Message.ToolCalls)
	}

	return res, nil
}

func createOpenAIChunkHandler(callback func(chunk models.StreamedResponse) error) func(oai.ChatCompletionChunk) {
	return func(chunk oai.ChatCompletionChunk) {
		if len(chunk.Choices) == 0 {
			log.Println("No choices in chunk")
			return
		}

		streamResp := models.StreamedResponse{
			AIResponse: chunk.Choices[0].Delta.Content,
			TokenUsed:  int(chunk.Usage.TotalTokens),
			TimeTaken:  int(chunk.Created),
		}

		if len(chunk.Choices[0].Delta.ToolCalls) > 0 {
			streamResp.ToolCalls = buildStreamToolCallsFromOpenAI(chunk.Choices[0].Delta.ToolCalls)
		}

		callback(streamResp)
	}
}

func buildStreamToolCallsFromOpenAI(toolCalls []oai.ChatCompletionChunkChoiceDeltaToolCall) []models.ToolCall {
	result := make([]models.ToolCall, len(toolCalls))
	for i, tc := range toolCalls {
		idx := int(tc.Index)
		result[i] = models.ToolCall{
			Index: &idx,
			ID:    tc.ID,
			Type:  string(tc.Type),
			Function: models.ToolCallFunction{
				Name:      tc.Function.Name,
				Arguments: tc.Function.Arguments,
			},
		}
	}
	return result
}
