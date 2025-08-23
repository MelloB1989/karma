package tests

import (
	"fmt"

	"github.com/MelloB1989/karma/ai"
)

func GetEmbedding(text string) ([]float32, error) {
	kai := ai.NewKarmaAI(ai.TitanEmbedText, ai.Bedrock)

	embeddingPrompt := fmt.Sprintf("Generate embedding for: %s", text)
	embeddingResponse, err := kai.GetEmbeddings(embeddingPrompt)
	if err != nil {
		return nil, err
	}

	return embeddingResponse.Embedding, nil
}
