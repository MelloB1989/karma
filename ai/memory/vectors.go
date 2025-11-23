package memory

import (
	"fmt"
	"strings"

	"github.com/upstash/vector-go"
)

func (d *dbClient) upsertVector(memId, category, lifespan string, importance int, v []float32) error {
	return d.vn.Upsert(vector.Upsert{
		Id:     memId,
		Vector: v,
		Metadata: map[string]any{
			"namespace":  d.scope,
			"category":   category,
			"lifespan":   lifespan,
			"importance": importance,
		},
	})
}

type filters struct {
	category         *string
	lifespan         *string
	importance       *int
	includeAllScopes *bool
}

func (d *dbClient) queryVector(v []float32, fs ...filters) ([]vector.VectorScore, error) {
	var f *filters
	if len(fs) > 0 {
		f = &fs[0]
	}

	var parts []string

	if f == nil || f.includeAllScopes == nil || !*f.includeAllScopes {
		parts = append(parts, fmt.Sprintf("namespace = '%s'", d.scope))
	}

	if f != nil {
		if f.category != nil && *f.category != "" {
			parts = append(parts, fmt.Sprintf("category = '%s'", *f.category))
		}
		if f.lifespan != nil && *f.lifespan != "" {
			parts = append(parts, fmt.Sprintf("lifespan = '%s'", *f.lifespan))
		}
		if f.importance != nil && *f.importance != 0 {
			parts = append(parts, fmt.Sprintf("importance = %d", *f.importance))
		}
	}

	filter := strings.Join(parts, " AND ")

	q := vector.Query{
		Vector:          v,
		TopK:            5,
		IncludeVectors:  false,
		IncludeMetadata: true,
		IncludeData:     false,
	}

	if filter != "" {
		q.Filter = filter
	}

	scores, err := d.vn.Query(q)
	if err != nil {
		return nil, fmt.Errorf("upstash vector query failed: %w", err)
	}

	return scores, nil
}

func (k *KarmaMemory) getEmbeddings(text string) ([]float32, error) {
	resp, err := k.embeddingAI.GetEmbeddings(text)
	if err != nil {
		return nil, fmt.Errorf("embedding AI failed: %w", err)
	}
	return resp.GetEmbeddingsFloat32(), nil
}
