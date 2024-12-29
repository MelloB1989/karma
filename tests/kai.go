package tests

import (
	"fmt"

	"github.com/MelloB1989/karma/ai"
	"github.com/MelloB1989/karma/models"
	"github.com/openai/openai-go"
)

func TestKai() {
	// testChatCompletion()
	testGenerateFromSinglePrompt()
	// testChatCompletionStream()
}

func testChatCompletion() {
	kai := ai.NewKarmaAI(ai.ChatModelChatgpt4oLatest, ai.WithUserPrePrompt("I am Kartik Deshmukh. "))
	response, err := kai.ChatCompletion(models.AIChatHistory{
		Messages: []models.AIMessage{
			{
				Message: "Hello",
				Role:    models.User,
			},
		},
	})
	if err != nil {
		panic(err)
	}
	fmt.Println(response.AIResponse)
}

func testGenerateFromSinglePrompt() {
	kai := ai.NewKarmaAI(ai.ChatModelChatgpt4oLatest, ai.WithUserPrePrompt("This is Kartik Deshmukh. "), ai.WithTemperature(0.5), ai.WithMaxTokens(512), ai.WithTopP(0.9))
	response, err := kai.GenerateFromSinglePrompt("Hello!")
	if err != nil {
		panic(err)
	}
	fmt.Println(response.AIResponse)
}

func testChatCompletionStream() {
	chuckHandler := func(chuck openai.ChatCompletionChunk) {
		fmt.Print(chuck.Choices[0].Delta.Content)
	}
	kai := ai.NewKarmaAI(ai.ChatModelChatgpt4oLatest, ai.WithUserPrePrompt("I am Kartik Deshmukh. "))
	response, err := kai.ChatCompletionStream(models.AIChatHistory{
		Messages: []models.AIMessage{
			{
				Message: "Hello, how to create a new file in Go?",
				Role:    models.User,
			},
		},
	}, chuckHandler)
	if err != nil {
		panic(err)
	}
	fmt.Println(response.AIResponse)
}
