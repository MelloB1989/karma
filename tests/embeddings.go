package tests

import (
	"log"

	"github.com/MelloB1989/karma/ai"
)

func TestKAIEmbeddingGeneration() {
	kai := ai.NewKarmaAI(ai.TextEmbedding3Small, ai.OpenAI)

	embeddingString := "I like milkshakes."
	embeddingResponse, err := kai.GetEmbeddings(embeddingString)
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Embedding: ", embeddingResponse.GetEmbeddingsFloat32())
}
