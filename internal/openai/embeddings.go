package openai

import (
	"context"

	"github.com/openai/openai-go/v3"
)

func GenerateEmbeddings(ofstring, model string, com ...CompatibleOptions) (*openai.CreateEmbeddingResponse, error) {
	client := createClient(com...)
	resp, err := client.Embeddings.New(context.TODO(), openai.EmbeddingNewParams{
		Input: openai.EmbeddingNewParamsInputUnion{
			OfString: openai.String(ofstring),
		},
		Model:          openai.EmbeddingModel(model),
		EncodingFormat: openai.EmbeddingNewParamsEncodingFormatFloat,
	})
	if err != nil {
		return nil, err
	}
	return resp, nil
}
