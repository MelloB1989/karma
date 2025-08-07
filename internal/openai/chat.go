package openai

import (
	"context"
	"log"

	"github.com/MelloB1989/karma/config"
	"github.com/MelloB1989/karma/models"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

type compatibleOptions struct {
	BaseURL string
	API_Key string
}

func createClient(opts ...compatibleOptions) *openai.Client {
	if len(opts) > 0 {
		return openai.NewClient(option.WithAPIKey(opts[0].API_Key), option.WithBaseURL(opts[0].BaseURL))
	}
	return openai.NewClient(option.WithAPIKey(config.DefaultConfig().OPENAI_KEY))
}

type OpenAI struct {
	Client        *openai.Client
	Model         string
	Temperature   float64
	MaxTokens     int64
	SystemMessage string
}

func NewOpenAI(model string, temperature float64, maxTokens int64) *OpenAI {
	return &OpenAI{
		Client:        createClient(),
		Model:         model,
		Temperature:   temperature,
		MaxTokens:     maxTokens,
		SystemMessage: "",
	}
}

func NewOpenAICompatible(model string, temperature float64, maxTokens int64, base_url, apikey string) *OpenAI {
	return &OpenAI{
		Client: createClient(compatibleOptions{
			BaseURL: base_url,
			API_Key: apikey,
		}),
		Model:         model,
		Temperature:   temperature,
		MaxTokens:     maxTokens,
		SystemMessage: "",
	}
}

func (o *OpenAI) CreateChat(messages models.AIChatHistory) (*openai.ChatCompletion, error) {
	mgs := formatMessages(messages)
	mgs = append(mgs, openai.SystemMessage(o.SystemMessage))
	chatCompletion, err := o.Client.Chat.Completions.New(context.TODO(), openai.ChatCompletionNewParams{
		Model:    openai.F(o.Model),
		Messages: openai.F(mgs),
	})
	if err != nil {
		return nil, err
	}
	return chatCompletion, nil
}

func (o *OpenAI) CreateChatStream(messages models.AIChatHistory, chunkHandler func(chuck openai.ChatCompletionChunk)) (*openai.ChatCompletion, error) {
	mgs := formatMessages(messages)
	mgs = append(mgs, openai.SystemMessage(o.SystemMessage))
	stream := o.Client.Chat.Completions.NewStreaming(context.TODO(), openai.ChatCompletionNewParams{
		Model:    openai.F(o.Model),
		Messages: openai.F(mgs),
		Seed:     openai.Int(69),
	})
	// optionally, an accumulator helper can be used
	acc := openai.ChatCompletionAccumulator{}

	for stream.Next() {
		chunk := stream.Current()
		acc.AddChunk(chunk)
		chunkHandler(chunk)
	}

	if err := stream.Err(); err != nil {
		log.Println(err)
	}

	// After the stream is finished, acc can be used like a ChatCompletion
	return &acc.ChatCompletion, nil
}

func formatMessages(messages models.AIChatHistory) []openai.ChatCompletionMessageParamUnion {
	mgs := []openai.ChatCompletionMessageParamUnion{}
	for _, message := range messages.Messages {
		switch message.Role {
		case "user":
			mgs = append(mgs, openai.UserMessage(message.Message))
		case "assistant":
			mgs = append(mgs, openai.AssistantMessage(message.Message))
		case "system":
			mgs = append(mgs, openai.SystemMessage(message.Message))
		}
	}
	return mgs
}
