package claude

import (
	"context"

	"github.com/MelloB1989/karma/config"
	"github.com/MelloB1989/karma/models"
	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/anthropics/anthropic-sdk-go/packages/param"
)

type ClaudeClient struct {
	Client       *anthropic.Client
	MaxTokens    int
	Model        anthropic.Model
	Temp         float64
	TopP         float64
	TopK         int64
	SystemPrompt string
}

func NewClaudeClient(maxTokens int, model anthropic.Model, temp float64, topP float64, topK float64, systemPrompt string) *ClaudeClient {
	client := anthropic.NewClient(
		option.WithAPIKey(config.GetEnvRaw("ANTHROPIC_API_KEY")),
	)
	return &ClaudeClient{
		Client:       &client,
		MaxTokens:    maxTokens,
		Model:        model,
		Temp:         temp,
		TopP:         topP,
		TopK:         int64(topK),
		SystemPrompt: systemPrompt,
	}
}

func (cc *ClaudeClient) ClaudeSinglePrompt(prompt string) (string, error) {
	mgsParam := anthropic.MessageNewParams{
		MaxTokens: int64(cc.MaxTokens),
		Messages: []anthropic.MessageParam{{
			Content: []anthropic.ContentBlockParamUnion{{
				OfText: &anthropic.TextBlockParam{Text: prompt},
			}},
			Role: anthropic.MessageParamRoleUser,
		}},
		Model: cc.Model,
	}
	if cc.Temp > 0 {
		mgsParam.Temperature = param.NewOpt(cc.Temp)
	}
	if cc.TopP > 0 {
		mgsParam.TopP = param.NewOpt(cc.TopP)
	}
	if cc.TopK > 0 {
		mgsParam.TopK = param.NewOpt(cc.TopK)
	}
	if cc.SystemPrompt != "" {
		mgsParam.System = []anthropic.TextBlockParam{{Text: cc.SystemPrompt}}
	}
	message, err := cc.Client.Messages.New(context.TODO(), mgsParam)
	if err != nil {
		return "", err
	}
	return message.Content[0].Text, nil
}

func (cc *ClaudeClient) ClaudeChatCompletion(messages models.AIChatHistory) (string, error) {
	processedMessages := processMessages(messages)
	mgsParam := anthropic.MessageNewParams{
		MaxTokens: int64(cc.MaxTokens),
		Messages:  processedMessages,
		Model:     cc.Model,
	}
	if cc.Temp > 0 {
		mgsParam.Temperature = param.NewOpt(cc.Temp)
	}
	if cc.TopP > 0 {
		mgsParam.TopP = param.NewOpt(cc.TopP)
	}
	if cc.TopK > 0 {
		mgsParam.TopK = param.NewOpt(cc.TopK)
	}
	if cc.SystemPrompt != "" {
		mgsParam.System = []anthropic.TextBlockParam{{Text: cc.SystemPrompt}}
	}
	message, err := cc.Client.Messages.New(context.TODO(), mgsParam)
	if err != nil {
		return "", err
	}
	return message.Content[0].Text, nil
}

func (cc *ClaudeClient) ClaudeStreamCompletion(messages models.AIChatHistory, callback func(chunck models.StreamedResponse) error) (string, error) {
	processedMessages := processMessages(messages)
	streamParams := anthropic.MessageNewParams{
		MaxTokens: int64(cc.MaxTokens),
		Messages:  processedMessages,
		Model:     cc.Model,
	}
	if cc.Temp > 0 {
		streamParams.Temperature = param.NewOpt(cc.Temp)
	}
	if cc.TopP > 0 {
		streamParams.TopP = param.NewOpt(cc.TopP)
	}
	if cc.TopK > 0 {
		streamParams.TopK = param.NewOpt(cc.TopK)
	}
	if cc.SystemPrompt != "" {
		streamParams.System = []anthropic.TextBlockParam{{Text: cc.SystemPrompt}}
	}
	stream := cc.Client.Messages.NewStreaming(context.TODO(), streamParams)
	message := anthropic.Message{}
	for stream.Next() {
		event := stream.Current()
		err := message.Accumulate(event)
		if err != nil {
			return "", err
		}

		switch eventVariant := event.AsAny().(type) {
		case anthropic.ContentBlockDeltaEvent:
			switch deltaVariant := eventVariant.Delta.AsAny().(type) {
			case anthropic.TextDelta:
				// print(deltaVariant.Text)
				chunk := models.StreamedResponse{
					AIResponse: deltaVariant.Text,
				}
				if err := callback(chunk); err != nil {
					return "", err
				}
			}
		}
	}

	if stream.Err() != nil {
		return "", stream.Err()
	}
	return message.Content[0].Text, nil
}
