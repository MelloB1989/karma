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
	client := createClient()
	mgs := formatMessages(messages)
	mgs = append(mgs, openai.SystemMessage(o.SystemMessage))
	stream := client.Chat.Completions.NewStreaming(context.TODO(), openai.ChatCompletionNewParams{
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

		// if content, ok := acc.JustFinishedContent(); ok {
		// 	println("Content stream finished:", content)
		// }

		// // if using tool calls
		// if tool, ok := acc.JustFinishedToolCall(); ok {
		// 	println("Tool call stream finished:", tool.Index, tool.Name, tool.Arguments)
		// }

		// if refusal, ok := acc.JustFinishedRefusal(); ok {
		// 	println("Refusal stream finished:", refusal)
		// }

		// it's best to use chunks after handling JustFinished events
		// if len(chunk.Choices) > 0 {
		// 	println(chunk.Choices[0].Delta.Content)
		// }
	}

	if err := stream.Err(); err != nil {
		panic(err)
	}

	// After the stream is finished, acc can be used like a ChatCompletion
	return &acc.ChatCompletion, nil
}
