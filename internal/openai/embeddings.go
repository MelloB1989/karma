package openai

import (
	"context"

	"github.com/openai/openai-go/v3"
)

func GenerateEmbeddings(ofstring, model string, com ...CompatibleOptions) (*openai.CreateEmbeddingResponse, error) {
	return GenerateEmbeddingsWithContext(context.TODO(), ofstring, model, com...)
}

func GenerateEmbeddingsWithContext(ctx context.Context, ofstring, model string, com ...CompatibleOptions) (*openai.CreateEmbeddingResponse, error) {
	client := createClient(com...)
	resp, err := client.Embeddings.New(ctx, openai.EmbeddingNewParams{
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
