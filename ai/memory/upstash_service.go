package memory

import (
	"fmt"
	"strings"

	"github.com/MelloB1989/karma/config"
	"github.com/upstash/vector-go"
	"go.uber.org/zap"
)

type upstashVectorClient struct {
	userId string
	scope  string
	ns     *vector.Namespace
	idx    *vector.Index
	logger *zap.Logger
}

func newUpstashClient(userId, scope string, logger *zap.Logger) vectorService {
	if config.GetEnvRaw("KARMA_MEMORY_UPSTASH_VECTOR_REST_URL") == "" || config.GetEnvRaw("KARMA_MEMORY_UPSTASH_VECTOR_REST_TOKEN") == "" {
		logger.Fatal("[KARMA_MEMORY] UPSTASH CREDENTIALS NOT FOUND: Please setup environment variables for Karma Memory.")
		return nil
	}
	index := vector.NewIndex(config.GetEnvRaw("KARMA_MEMORY_UPSTASH_VECTOR_REST_URL"), config.GetEnvRaw("KARMA_MEMORY_UPSTASH_VECTOR_REST_TOKEN"))
	userIdx := index.Namespace(userId)

	client := &upstashVectorClient{
		userId: userId,
		scope:  scope,
		logger: logger,
		ns:     userIdx,
		idx:    index,
	}

	return client
}

func (d *upstashVectorClient) upsertVectors(vectors []v) error {
	vs := []vector.Upsert{}
	for _, v := range vectors {
		m, err := v.memories.ToMap()
		if err != nil {
			d.logger.Error("Failed to map memory metadata", zap.Error(err))
		}
		vs = append(vs, vector.Upsert{
			Id:       v.memories.Id,
			Vector:   v.vector,
			Metadata: m,
		})
	}
	return d.ns.UpsertMany(vs)
}

func (d *upstashVectorClient) buildMetadataFilter(f *filters, includeScope bool) string {
	var parts []string

	if includeScope && (f == nil || f.IncludeAllScopes == nil || !*f.IncludeAllScopes) {
		parts = append(parts, fmt.Sprintf("namespace = '%s'", d.scope))
	}

	if f != nil {
		if f.Category != nil && *f.Category != "" {
			categories := strings.Split(*f.Category, ",")
			for i := range categories {
				categories[i] = strings.TrimSpace(categories[i])
			}
			if len(categories) == 1 {
				parts = append(parts, fmt.Sprintf("category = '%s'", categories[0]))
			} else {
				// Build IN clause: category IN ('fact', 'preference', 'rule')
				quotedCats := make([]string, len(categories))
				for i, cat := range categories {
					quotedCats[i] = fmt.Sprintf("'%s'", cat)
				}
				parts = append(parts, fmt.Sprintf("category IN (%s)", strings.Join(quotedCats, ", ")))
			}
		}
		if f.Lifespan != nil && *f.Lifespan != "" {
			lifespans := strings.Split(*f.Lifespan, ",")
			for i := range lifespans {
				lifespans[i] = strings.TrimSpace(lifespans[i])
			}
			if len(lifespans) == 1 {
				parts = append(parts, fmt.Sprintf("lifespan = '%s'", lifespans[0]))
			} else {
				// Build IN clause: lifespan IN ('short_term', 'long_term')
				quotedLifespans := make([]string, len(lifespans))
				for i, ls := range lifespans {
					quotedLifespans[i] = fmt.Sprintf("'%s'", ls)
				}
				parts = append(parts, fmt.Sprintf("lifespan IN (%s)", strings.Join(quotedLifespans, ", ")))
			}
		}
		if f.Importance != nil && *f.Importance != 0 {
			parts = append(parts, fmt.Sprintf("importance = %d", *f.Importance))
		}
		if f.Expiry != nil {
			parts = append(parts, fmt.Sprintf("expiry = %s", *f.Expiry))
		}
		if f.Status != nil && *f.Status != "" {
			parts = append(parts, fmt.Sprintf("status = '%s'", *f.Status))
		}
	}

	return strings.Join(parts, " AND ")
}

func (d *upstashVectorClient) queryVector(v []float32, topK int, fs ...filters) ([]vector.VectorScore, error) {
	var f *filters
	if len(fs) > 0 {
		f = &fs[0]
	}

	filter := d.buildMetadataFilter(f, true)

	q := vector.Query{
		Vector:          v,
		TopK:            topK,
		IncludeVectors:  false,
		IncludeMetadata: true,
		IncludeData:     false,
	}

	if filter != "" {
		q.Filter = filter
	}

	scores, err := d.ns.Query(q)
	if err != nil {
		return nil, fmt.Errorf("upstash vector query failed: %w", err)
	}

	return scores, nil
}

func (d *upstashVectorClient) queryVectorByMetadata(f filters) ([]map[string]any, error) {
	filter := d.buildMetadataFilter(&f, true)

	if filter == "" {
		return nil, fmt.Errorf("no metadata filters provided")
	}

	r := vector.Range{
		Cursor:          "",
		Limit:           1000,
		IncludeVectors:  false,
		IncludeMetadata: true,
		IncludeData:     false,
	}

	rangeVectors, err := d.ns.Range(r)
	if err != nil {
		return nil, fmt.Errorf("upstash range query failed: %w", err)
	}

	metadatas := make([]map[string]any, 0)
	for _, vec := range rangeVectors.Vectors {
		if d.matchesFilter(vec.Metadata, &f) {
			metadatas = append(metadatas, vec.Metadata)
		}
	}

	return metadatas, nil
}

func (d *upstashVectorClient) matchesFilter(metadata map[string]interface{}, f *filters) bool {
	if f.IncludeAllScopes == nil || !*f.IncludeAllScopes {
		if ns, ok := metadata["namespace"].(string); !ok || ns != d.scope {
			return false
		}
	}

	if f.Category != nil && *f.Category != "" {
		cat, ok := metadata["category"].(string)
		if !ok {
			return false
		}
		categories := strings.Split(*f.Category, ",")
		found := false
		for _, c := range categories {
			if strings.TrimSpace(c) == cat {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	if f.Lifespan != nil && *f.Lifespan != "" {
		ls, ok := metadata["lifespan"].(string)
		if !ok {
			return false
		}
		lifespans := strings.Split(*f.Lifespan, ",")
		found := false
		for _, l := range lifespans {
			if strings.TrimSpace(l) == ls {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	if f.Importance != nil && *f.Importance != 0 {
		if imp, ok := metadata["importance"].(float64); !ok || int(imp) != *f.Importance {
			return false
		}
	}

	if f.Status != nil && *f.Status != "" {
		if status, ok := metadata["status"].(string); !ok || status != string(*f.Status) {
			return false
		}
	}

	if f.Expiry != nil {
		if exp, ok := metadata["expiry"].(float64); !ok || int64(exp) > f.Expiry.Unix() {
			return false
		}
	}

	return true
}

func (d *upstashVectorClient) updateVector(memory Memory, v ...[]float32) (bool, error) {
	if len(v) == 0 {
		d.logger.Warn("no vector provided to updateVector")
		return false, fmt.Errorf("no vector provided")
	}

	m, err := memory.ToMap()
	if err != nil {
		d.logger.Error("Failed to map memory metadata", zap.Error(err))
	}

	vs := vector.Update{
		Id:       memory.Id,
		Vector:   v[0],
		Metadata: m,
	}
	return d.ns.Update(vs)
}

func (d *upstashVectorClient) deleteVectors(vectorsIds []string) (count int, err error) {
	return d.ns.DeleteMany(vectorsIds)
}

func (d *upstashVectorClient) shiftScope(scope string) string {
	d.scope = scope
	return d.scope
}

func (d *upstashVectorClient) shiftUser(userId string) string {
	d.userId = userId
	d.ns = d.idx.Namespace(d.userId)
	return d.userId
}
