package memory

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/MelloB1989/karma/config"
	"github.com/pinecone-io/go-pinecone/v4/pinecone"
	"github.com/upstash/vector-go"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/structpb"
)

type pineconeVectorClient struct {
	userId string
	scope  string
	client *pinecone.Client
	idx    *pinecone.IndexConnection
	logger *zap.Logger
	ctx    context.Context
}

func newPineconeClient(userId, scope string, logger *zap.Logger) vectorService {
	apiKey := config.GetEnvRaw("KARMA_MEMORY_PINECONE_API_KEY")
	indexHost := config.GetEnvRaw("KARMA_MEMORY_PINECONE_INDEX_HOST")

	if apiKey == "" || indexHost == "" {
		logger.Fatal("[KARMA_MEMORY] PINECONE CREDENTIALS NOT FOUND: Please setup environment variables for Karma Memory.")
		return nil
	}

	ctx := context.Background()

	pc, err := pinecone.NewClient(pinecone.NewClientParams{
		ApiKey: apiKey,
	})
	if err != nil {
		logger.Fatal("[KARMA_MEMORY] Failed to create Pinecone client", zap.Error(err))
		return nil
	}

	idxConnection, err := pc.Index(pinecone.NewIndexConnParams{
		Host:      indexHost,
		Namespace: userId,
	})
	if err != nil {
		logger.Fatal("[KARMA_MEMORY] Failed to create Pinecone IndexConnection", zap.Error(err))
		return nil
	}

	client := &pineconeVectorClient{
		userId: userId,
		scope:  scope,
		logger: logger,
		client: pc,
		idx:    idxConnection,
		ctx:    ctx,
	}

	return client
}

func (d *pineconeVectorClient) upsertVectors(vectors []v) error {
	useIntegratedRecords := len(vectors) > 0 && len(vectors[0].vector) == 0

	if useIntegratedRecords {
		records := make([]*pinecone.IntegratedRecord, 0, len(vectors))

		for _, v := range vectors {
			record := &pinecone.IntegratedRecord{
				"_id":  v.memories.Id,
				"text": v.memories.Summary,
			}

			// Add sanitized metadata fields (only primitive types supported)
			(*record)["subject_key"] = v.memories.SubjectKey
			(*record)["namespace"] = v.memories.Namespace
			(*record)["summary"] = v.memories.Summary
			(*record)["category"] = string(v.memories.Category)
			(*record)["importance"] = v.memories.Importance
			(*record)["mutability"] = string(v.memories.Mutability)
			(*record)["lifespan"] = string(v.memories.Lifespan)
			(*record)["forget_score"] = v.memories.ForgetScore
			(*record)["status"] = string(v.memories.Status)
			(*record)["created_at"] = v.memories.CreatedAt.Format(time.RFC3339)
			(*record)["updated_at"] = v.memories.UpdatedAt.Format(time.RFC3339)

			if v.memories.RawText != "" {
				(*record)["raw_text"] = v.memories.RawText
			}
			if v.memories.ExpiresAt != nil {
				(*record)["expires_at"] = v.memories.ExpiresAt.Format(time.RFC3339)
			}
			if len(v.memories.Metadata) > 0 {
				(*record)["metadata"] = string(v.memories.Metadata)
			}
			if len(v.memories.SupersedesCanonicalKeys) > 0 {
				(*record)["supersedes_canonical_keys"] = v.memories.SupersedesCanonicalKeys
			}
			if len(v.memories.EntityRelationships) > 0 {
				relBytes, err := json.Marshal(v.memories.EntityRelationships)
				if err == nil {
					(*record)["entity_relationships_json"] = string(relBytes)
				}
			}

			records = append(records, record)
		}

		err := d.idx.UpsertRecords(d.ctx, records)
		if err != nil {
			d.logger.Error("Pinecone integrated records upsert failed", zap.Error(err), zap.Int("record_count", len(records)))
			return fmt.Errorf("pinecone integrated records upsert failed: %w", err)
		}

		d.logger.Info("Pinecone integrated records upserted", zap.Int("record_count", len(records)))
		return nil
	}

	pineconeVectors := make([]*pinecone.Vector, 0, len(vectors))

	for _, v := range vectors {
		m, err := v.memories.ToMap()
		if err != nil {
			d.logger.Error("Failed to map memory metadata", zap.Error(err))
			continue
		}

		metadata, err := structpb.NewStruct(m)
		if err != nil {
			d.logger.Error("Failed to create metadata struct", zap.Error(err))
			continue
		}

		vectorCopy := make([]float32, len(v.vector))
		copy(vectorCopy, v.vector)
		pineconeVectors = append(pineconeVectors, &pinecone.Vector{
			Id:       v.memories.Id,
			Values:   &vectorCopy,
			Metadata: metadata,
		})
	}

	_, err := d.idx.UpsertVectors(d.ctx, pineconeVectors)
	if err != nil {
		return fmt.Errorf("pinecone upsert failed: %w", err)
	}

	return nil
}

func (d *pineconeVectorClient) queryVector(v []float32, topK int, fs ...filters) ([]vector.VectorScore, error) {
	var f *filters
	if len(fs) > 0 {
		f = &fs[0]
	}

	useIntegratedSearch := len(v) == 0

	if useIntegratedSearch && f != nil && f.SearchQuery != "" {
		return d.queryByText(f.SearchQuery, topK, fs...)
	}

	var metadataFilter map[string]any

	if f == nil || f.IncludeAllScopes == nil || !*f.IncludeAllScopes {
		metadataFilter = map[string]any{
			"namespace": map[string]any{
				"$eq": d.scope,
			},
		}
	}

	if f != nil {
		filterConditions := []any{}
		if metadataFilter != nil {
			if namespaceFilter, ok := metadataFilter["namespace"]; ok {
				filterConditions = append(filterConditions, map[string]any{
					"namespace": namespaceFilter,
				})
			}
		}
		if f.Category != nil && *f.Category != "" {
			categories := strings.Split(*f.Category, ",")
			for i := range categories {
				categories[i] = strings.TrimSpace(categories[i])
			}
			if len(categories) == 1 {
				filterConditions = append(filterConditions, map[string]any{
					"category": map[string]any{
						"$eq": categories[0],
					},
				})
			} else {
				catInterface := make([]any, len(categories))
				for i, c := range categories {
					catInterface[i] = c
				}
				filterConditions = append(filterConditions, map[string]any{
					"category": map[string]any{
						"$in": catInterface,
					},
				})
			}
		}
		if f.Lifespan != nil && *f.Lifespan != "" {
			lifespans := strings.Split(*f.Lifespan, ",")
			for i := range lifespans {
				lifespans[i] = strings.TrimSpace(lifespans[i])
			}
			if len(lifespans) == 1 {
				filterConditions = append(filterConditions, map[string]any{
					"lifespan": map[string]any{
						"$eq": lifespans[0],
					},
				})
			} else {
				// Convert to []any for protobuf compatibility
				lsInterface := make([]any, len(lifespans))
				for i, l := range lifespans {
					lsInterface[i] = l
				}
				filterConditions = append(filterConditions, map[string]any{
					"lifespan": map[string]any{
						"$in": lsInterface,
					},
				})
			}
		}
		if f.Importance != nil && *f.Importance != 0 {
			filterConditions = append(filterConditions, map[string]any{
				"importance": map[string]any{
					"$eq": *f.Importance,
				},
			})
		}
		if f.Expiry != nil {
			filterConditions = append(filterConditions, map[string]any{
				"expiry": map[string]any{
					"$lte": f.Expiry.Unix(),
				},
			})
		}
		if f.Status != nil && *f.Status != "" {
			filterConditions = append(filterConditions, map[string]any{
				"status": map[string]any{
					"$eq": string(*f.Status),
				},
			})
		}
		if len(filterConditions) > 1 {
			metadataFilter = map[string]any{
				"$and": filterConditions,
			}
		} else if len(filterConditions) == 1 {
			metadataFilter = filterConditions[0].(map[string]any)
		}
	}

	var filterStruct *structpb.Struct
	if metadataFilter != nil {
		var err error
		filterStruct, err = structpb.NewStruct(d.sanitizeFilterForProto(metadataFilter))
		if err != nil {
			d.logger.Warn("Failed to create filter struct, querying without filter", zap.Error(err))
			filterStruct = nil
		}
	}

	queryReq := &pinecone.QueryByVectorValuesRequest{
		Vector:          v,
		TopK:            uint32(topK),
		IncludeValues:   false,
		IncludeMetadata: true,
		MetadataFilter:  filterStruct,
	}

	res, err := d.idx.QueryByVectorValues(d.ctx, queryReq)
	if err != nil {
		return nil, fmt.Errorf("pinecone query failed: %w", err)
	}

	scores := make([]vector.VectorScore, 0, len(res.Matches))
	for _, match := range res.Matches {
		metadataMap := make(map[string]any)
		if match.Vector.Metadata != nil {
			metadataMap = match.Vector.Metadata.AsMap()
		}
		var vectorValues []float32
		if match.Vector.Values != nil {
			vectorValues = *match.Vector.Values
		}
		scores = append(scores, vector.VectorScore{
			Id:       match.Vector.Id,
			Score:    match.Score,
			Vector:   vectorValues,
			Metadata: metadataMap,
		})
	}

	return scores, nil
}

func (d *pineconeVectorClient) queryVectorByMetadata(f filters) ([]map[string]any, error) {
	var metadataFilter map[string]any

	if f.IncludeAllScopes == nil || !*f.IncludeAllScopes {
		metadataFilter = map[string]any{
			"namespace": map[string]any{
				"$eq": d.scope,
			},
		}
	}

	filterConditions := []any{}
	if metadataFilter != nil {
		if namespaceFilter, ok := metadataFilter["namespace"]; ok {
			filterConditions = append(filterConditions, map[string]any{
				"namespace": namespaceFilter,
			})
		}
	}
	if f.Category != nil && *f.Category != "" {
		categories := strings.Split(*f.Category, ",")
		for i := range categories {
			categories[i] = strings.TrimSpace(categories[i])
		}
		if len(categories) == 1 {
			filterConditions = append(filterConditions, map[string]any{
				"category": map[string]any{
					"$eq": categories[0],
				},
			})
		} else {
			// Convert to []any for protobuf compatibility
			catInterface := make([]any, len(categories))
			for i, c := range categories {
				catInterface[i] = c
			}
			filterConditions = append(filterConditions, map[string]any{
				"category": map[string]any{
					"$in": catInterface,
				},
			})
		}
	}
	if f.Lifespan != nil && *f.Lifespan != "" {
		lifespans := strings.Split(*f.Lifespan, ",")
		for i := range lifespans {
			lifespans[i] = strings.TrimSpace(lifespans[i])
		}
		if len(lifespans) == 1 {
			filterConditions = append(filterConditions, map[string]any{
				"lifespan": map[string]any{
					"$eq": lifespans[0],
				},
			})
		} else {
			// Convert to []any for protobuf compatibility
			lsInterface := make([]any, len(lifespans))
			for i, l := range lifespans {
				lsInterface[i] = l
			}
			filterConditions = append(filterConditions, map[string]any{
				"lifespan": map[string]any{
					"$in": lsInterface,
				},
			})
		}
	}
	if f.Importance != nil && *f.Importance != 0 {
		filterConditions = append(filterConditions, map[string]any{
			"importance": map[string]any{
				"$eq": *f.Importance,
			},
		})
	}
	if f.Expiry != nil {
		filterConditions = append(filterConditions, map[string]any{
			"expiry": map[string]any{
				"$lte": f.Expiry.Unix(),
			},
		})
	}
	if f.Status != nil && *f.Status != "" {
		filterConditions = append(filterConditions, map[string]any{
			"status": map[string]any{
				"$eq": string(*f.Status),
			},
		})
	}
	if len(filterConditions) == 0 {
		return nil, fmt.Errorf("no metadata filters provided")
	}
	if len(filterConditions) > 1 {
		metadataFilter = map[string]any{
			"$and": filterConditions,
		}
	} else {
		metadataFilter = filterConditions[0].(map[string]any)
	}

	filterStruct, err := structpb.NewStruct(d.sanitizeFilterForProto(metadataFilter))
	if err != nil {
		return nil, fmt.Errorf("failed to create filter struct: %w", err)
	}

	limit := uint32(100)
	var allMemories []map[string]any
	var paginationToken *string

	for {
		listReq := &pinecone.ListVectorsRequest{
			Limit:           &limit,
			PaginationToken: paginationToken,
		}

		listRes, err := d.idx.ListVectors(d.ctx, listReq)
		if err != nil {
			return nil, fmt.Errorf("pinecone list vectors failed: %w", err)
		}

		if len(listRes.VectorIds) == 0 {
			break
		}

		vectorIds := make([]string, len(listRes.VectorIds))
		for i, id := range listRes.VectorIds {
			if id != nil {
				vectorIds[i] = *id
			}
		}

		fetchRes, err := d.idx.FetchVectors(d.ctx, vectorIds)
		if err != nil {
			d.logger.Warn("Failed to fetch vectors", zap.Error(err))
			if listRes.NextPaginationToken == nil {
				break
			}
			paginationToken = listRes.NextPaginationToken
			continue
		}

		for _, vec := range fetchRes.Vectors {
			if d.matchesPineconeFilter(vec.Metadata, filterStruct) {
				metadataMap := make(map[string]any)
				if vec.Metadata != nil {
					metadataMap = vec.Metadata.AsMap()
				}

				allMemories = append(allMemories, metadataMap)
			}
		}

		if listRes.NextPaginationToken == nil {
			break
		}
		paginationToken = listRes.NextPaginationToken
	}

	return allMemories, nil
}

func (d *pineconeVectorClient) matchesPineconeFilter(metadata *structpb.Struct, filter *structpb.Struct) bool {
	if metadata == nil {
		return false
	}

	metadataMap := metadata.AsMap()
	filterMap := filter.AsMap()

	if andConditions, ok := filterMap["$and"].([]any); ok {
		for _, condition := range andConditions {
			condMap, ok := condition.(map[string]any)
			if !ok {
				return false
			}
			if !d.matchesSingleCondition(metadataMap, condMap) {
				return false
			}
		}
		return true
	}

	return d.matchesSingleCondition(metadataMap, filterMap)
}

func (d *pineconeVectorClient) matchesSingleCondition(metadataMap map[string]any, condMap map[string]any) bool {
	for field, operator := range condMap {
		opMap, ok := operator.(map[string]any)
		if !ok {
			continue
		}

		metadataValue, hasField := metadataMap[field]
		if !hasField {
			return false
		}

		for op, expectedValue := range opMap {
			switch op {
			case "$eq":
				if metadataValue != expectedValue {
					return false
				}
			case "$in":
				values, ok := expectedValue.([]any)
				if !ok {
					if strValues, ok := expectedValue.([]string); ok {
						found := false
						metaStr, _ := metadataValue.(string)
						for _, v := range strValues {
							if v == metaStr {
								found = true
								break
							}
						}
						if !found {
							return false
						}
					} else {
						return false
					}
				} else {
					found := false
					for _, v := range values {
						if metadataValue == v {
							found = true
							break
						}
					}
					if !found {
						return false
					}
				}
			case "$ne":
				if metadataValue == expectedValue {
					return false
				}
			case "$gt":
				metaFloat, ok1 := metadataValue.(float64)
				expFloat, ok2 := expectedValue.(float64)
				if !ok1 || !ok2 || metaFloat <= expFloat {
					return false
				}
			case "$gte":
				metaFloat, ok1 := metadataValue.(float64)
				expFloat, ok2 := expectedValue.(float64)
				if !ok1 || !ok2 || metaFloat < expFloat {
					return false
				}
			case "$lt":
				metaFloat, ok1 := metadataValue.(float64)
				expFloat, ok2 := expectedValue.(float64)
				if !ok1 || !ok2 || metaFloat >= expFloat {
					return false
				}
			case "$lte":
				metaFloat, ok1 := metadataValue.(float64)
				expFloat, ok2 := expectedValue.(float64)
				if !ok1 || !ok2 || metaFloat > expFloat {
					return false
				}
			}
		}
	}
	return true
}

func (d *pineconeVectorClient) updateVector(memory Memory, v ...[]float32) (bool, error) {
	useIntegratedRecords := len(v) == 0 || len(v[0]) == 0

	if useIntegratedRecords {
		err := d.idx.DeleteVectorsById(d.ctx, []string{memory.Id})
		if err != nil {
			d.logger.Warn("Failed to delete old vector for update", zap.Error(err), zap.String("id", memory.Id))
		}

		record := &pinecone.IntegratedRecord{
			"_id":  memory.Id,
			"text": memory.Summary,
		}

		(*record)["subject_key"] = memory.SubjectKey
		(*record)["namespace"] = memory.Namespace
		(*record)["category"] = string(memory.Category)
		(*record)["importance"] = memory.Importance
		(*record)["mutability"] = string(memory.Mutability)
		(*record)["lifespan"] = string(memory.Lifespan)
		(*record)["forget_score"] = memory.ForgetScore
		(*record)["status"] = string(memory.Status)
		(*record)["created_at"] = memory.CreatedAt.Format(time.RFC3339)
		(*record)["updated_at"] = memory.UpdatedAt.Format(time.RFC3339)

		if memory.RawText != "" {
			(*record)["raw_text"] = memory.RawText
		}
		if memory.ExpiresAt != nil {
			(*record)["expires_at"] = memory.ExpiresAt.Format(time.RFC3339)
		}
		if len(memory.Metadata) > 0 {
			(*record)["metadata_json"] = string(memory.Metadata)
		}
		if len(memory.SupersedesCanonicalKeys) > 0 {
			keysBytes, err := json.Marshal(memory.SupersedesCanonicalKeys)
			if err == nil {
				(*record)["supersedes_canonical_keys_json"] = string(keysBytes)
			}
		}
		if len(memory.EntityRelationships) > 0 {
			relBytes, err := json.Marshal(memory.EntityRelationships)
			if err == nil {
				(*record)["entity_relationships_json"] = string(relBytes)
			}
		}

		err = d.idx.UpsertRecords(d.ctx, []*pinecone.IntegratedRecord{record})
		if err != nil {
			d.logger.Error("Pinecone integrated record update failed", zap.Error(err))
			return false, fmt.Errorf("pinecone integrated record update failed: %w", err)
		}

		d.logger.Info("Pinecone integrated record updated", zap.String("id", memory.Id))
		return true, nil
	}

	m, err := memory.ToMap()
	if err != nil {
		d.logger.Error("Failed to map memory metadata", zap.Error(err))
		return false, err
	}

	metadata, err := structpb.NewStruct(m)
	if err != nil {
		d.logger.Error("Failed to create metadata struct", zap.Error(err))
		return false, err
	}

	err = d.idx.UpdateVector(d.ctx, &pinecone.UpdateVectorRequest{
		Id:       memory.Id,
		Values:   v[0],
		Metadata: metadata,
	})

	if err != nil {
		return false, fmt.Errorf("pinecone update failed: %w", err)
	}

	return true, nil
}

func (d *pineconeVectorClient) deleteVectors(vectorsIds []string) (count int, err error) {
	err = d.idx.DeleteVectorsById(d.ctx, vectorsIds)
	if err != nil {
		return 0, fmt.Errorf("pinecone delete failed: %w", err)
	}

	return len(vectorsIds), nil
}

func (d *pineconeVectorClient) shiftScope(scope string) string {
	d.scope = scope
	return d.scope
}

func (d *pineconeVectorClient) shiftUser(userId string) string {
	d.userId = userId
	indexHost := config.GetEnvRaw("KARMA_MEMORY_PINECONE_INDEX_HOST")
	idxConnection, err := d.client.Index(pinecone.NewIndexConnParams{
		Host:      indexHost,
		Namespace: userId,
	})
	if err != nil {
		d.logger.Error("[KARMA_MEMORY] Failed to reconnect to Pinecone with new namespace", zap.Error(err))
		return d.userId
	}

	d.idx = idxConnection
	return d.userId
}

// queryByText queries using Pinecone's integrated inference (for indexes with integrated embeddings)
func (d *pineconeVectorClient) queryByText(query string, topK int, fs ...filters) ([]vector.VectorScore, error) {
	var f *filters
	if len(fs) > 0 {
		f = &fs[0]
	}

	var metadataFilter map[string]any

	if f == nil || f.IncludeAllScopes == nil || !*f.IncludeAllScopes {
		metadataFilter = map[string]any{
			"namespace": map[string]any{
				"$eq": d.scope,
			},
		}
	}

	if f != nil {
		filterConditions := []any{}
		if metadataFilter != nil {
			if namespaceFilter, ok := metadataFilter["namespace"]; ok {
				filterConditions = append(filterConditions, map[string]any{
					"namespace": namespaceFilter,
				})
			}
		}
		if f.Category != nil && *f.Category != "" {
			categories := strings.Split(*f.Category, ",")
			for i := range categories {
				categories[i] = strings.TrimSpace(categories[i])
			}
			if len(categories) == 1 {
				filterConditions = append(filterConditions, map[string]any{
					"category": map[string]any{
						"$eq": categories[0],
					},
				})
			} else {
				catInterface := make([]any, len(categories))
				for i, c := range categories {
					catInterface[i] = c
				}
				filterConditions = append(filterConditions, map[string]any{
					"category": map[string]any{
						"$in": catInterface,
					},
				})
			}
		}
		if f.Lifespan != nil && *f.Lifespan != "" {
			lifespans := strings.Split(*f.Lifespan, ",")
			for i := range lifespans {
				lifespans[i] = strings.TrimSpace(lifespans[i])
			}
			if len(lifespans) == 1 {
				filterConditions = append(filterConditions, map[string]any{
					"lifespan": map[string]any{
						"$eq": lifespans[0],
					},
				})
			} else {
				lsInterface := make([]any, len(lifespans))
				for i, l := range lifespans {
					lsInterface[i] = l
				}
				filterConditions = append(filterConditions, map[string]any{
					"lifespan": map[string]any{
						"$in": lsInterface,
					},
				})
			}
		}
		if f.Status != nil && *f.Status != "" {
			filterConditions = append(filterConditions, map[string]any{
				"status": map[string]any{
					"$eq": string(*f.Status),
				},
			})
		}
		if len(filterConditions) > 1 {
			metadataFilter = map[string]any{
				"$and": filterConditions,
			}
		} else if len(filterConditions) == 1 {
			metadataFilter = filterConditions[0].(map[string]any)
		}
	}

	// Use SearchRecords for integrated inference
	inputs := map[string]any{
		"text": query,
	}
	var filterMap *map[string]any
	if metadataFilter != nil {
		sanitized := d.sanitizeFilterForProto(metadataFilter)
		filterMap = &sanitized
	}
	searchReq := &pinecone.SearchRecordsRequest{
		Query: pinecone.SearchRecordsQuery{
			TopK:   int32(topK),
			Inputs: &inputs,
			Filter: filterMap,
		},
	}

	res, err := d.idx.SearchRecords(d.ctx, searchReq)
	if err != nil {
		return nil, fmt.Errorf("pinecone text search failed: %w", err)
	}

	scores := make([]vector.VectorScore, 0, len(res.Result.Hits))
	for _, hit := range res.Result.Hits {
		metadataMap := make(map[string]any)
		if hit.Fields != nil {
			for k, v := range hit.Fields {
				metadataMap[k] = v
			}
		}
		scores = append(scores, vector.VectorScore{
			Id:       hit.Id,
			Score:    float32(hit.Score),
			Metadata: metadataMap,
		})
	}

	return scores, nil
}

// sanitizeFilterForProto converts filter map to a format compatible with structpb.NewStruct
func (d *pineconeVectorClient) sanitizeFilterForProto(filter map[string]any) map[string]any {
	result := make(map[string]any)

	for k, v := range filter {
		switch val := v.(type) {
		case []any:
			// Convert slice elements recursively
			sanitized := make([]any, len(val))
			for i, item := range val {
				if m, ok := item.(map[string]any); ok {
					sanitized[i] = d.sanitizeFilterForProto(m)
				} else {
					sanitized[i] = item
				}
			}
			result[k] = sanitized
		case map[string]any:
			result[k] = d.sanitizeFilterForProto(val)
		default:
			result[k] = v
		}
	}

	return result
}
