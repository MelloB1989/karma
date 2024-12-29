package openai

import (
	"context"

	"github.com/MelloB1989/karma/config"
	"github.com/MelloB1989/karma/models"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

func createClient() *openai.Client {
	client := openai.NewClient(option.WithAPIKey(config.DefaultConfig().OPENAI_KEY))
	return client
}

func formatMessages(messages models.AIChatHistory) []openai.ChatCompletionMessageParamUnion {
	mgs := []openai.ChatCompletionMessageParamUnion{}
	for _, message := range messages.Messages {
		if message.Role == "user" {
			mgs = append(mgs, openai.UserMessage(message.Message))
		} else if message.Role == "assistant" {
			mgs = append(mgs, openai.AssistantMessage(message.Message))
		} else if message.Role == "system" {
			mgs = append(mgs, openai.SystemMessage(message.Message))
		}
	}
	return mgs
}

func CreateChat(messages models.AIChatHistory, model string) (*openai.ChatCompletion, error) {
	client := createClient()
	chatCompletion, err := client.Chat.Completions.New(context.TODO(), openai.ChatCompletionNewParams{
		Model:    openai.F(model),
		Messages: openai.F(formatMessages(messages)),
	})
	if err != nil {
		return nil, err
	}
	return chatCompletion, nil
}
