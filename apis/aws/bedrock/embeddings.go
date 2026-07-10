package bedrock

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
)

// CreateEmbeddings generates an embedding vector for text using a Bedrock
// embedding model (Amazon Titan or Cohere) via the official AWS SDK v2
// InvokeModel API. It honours the same authentication modes as Converse,
// including Bedrock API keys.
func CreateEmbeddings(ctx context.Context, text, modelID string, opts ClientOptions) ([]float32, error) {
	client, err := NewRuntimeClient(ctx, opts)
	if err != nil {
		return nil, err
	}

	var payload []byte
	switch {
	case strings.Contains(modelID, "titan-embed"):
		payload, err = json.Marshal(map[string]string{"inputText": text})
	case strings.Contains(modelID, "cohere"):
		payload, err = json.Marshal(map[string]interface{}{
			"texts":      []string{text},
			"input_type": "search_document",
		})
	default:
		return nil, fmt.Errorf("unsupported embedding model: %s", modelID)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to marshal embedding request: %w", err)
	}

	out, err := client.InvokeModel(ctx, &bedrockruntime.InvokeModelInput{
		ModelId:     aws.String(modelID),
		ContentType: aws.String("application/json"),
		Accept:      aws.String("application/json"),
		Body:        payload,
	})
	if err != nil {
		return nil, fmt.Errorf("bedrock invoke model (embeddings) failed: %w", err)
	}

	if strings.Contains(modelID, "titan-embed") {
		var resp struct {
			Embedding []float32 `json:"embedding"`
		}
		if err := json.Unmarshal(out.Body, &resp); err != nil {
			return nil, fmt.Errorf("failed to unmarshal Titan response: %w", err)
		}
		return resp.Embedding, nil
	}

	var resp struct {
		Embeddings [][]float32 `json:"embeddings"`
	}
	if err := json.Unmarshal(out.Body, &resp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal Cohere response: %w", err)
	}
	if len(resp.Embeddings) == 0 {
		return nil, fmt.Errorf("no embeddings returned from model %s", modelID)
	}
	return resp.Embeddings[0], nil
}
