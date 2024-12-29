package tests

import (
	"fmt"

	"github.com/MelloB1989/karma/ai"
	"github.com/MelloB1989/karma/models"
)

func TestKai() {
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
