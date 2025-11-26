package memory

import (
	"context"
	"fmt"

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
			m, err := v.memories.ToMap()
			if err != nil {
				d.logger.Error("Failed to map memory metadata", zap.Error(err))
				continue
			}

			record := &pinecone.IntegratedRecord{
				"_id":        v.memories.Id,
				"chunk_text": v.memories.Summary,
			}

			for k, val := range m {
				if k != "id" && k != "summary" {
					(*record)[k] = val
				}
			}

			records = append(records, record)
		}

		err := d.idx.UpsertRecords(d.ctx, records)
		if err != nil {
			return fmt.Errorf("pinecone integrated records upsert failed: %w", err)
		}

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

	var metadataFilter map[string]any

	if f == nil || f.IncludeAllScopes == nil || !*f.IncludeAllScopes {

		metadataFilter = map[string]any{
			"namespace": map[string]any{
				"$eq": d.scope,
			},
		}
	}

	if f != nil {
		filterConditions := []map[string]any{}
		if metadataFilter != nil {
			if namespaceFilter, ok := metadataFilter["namespace"]; ok {
				filterConditions = append(filterConditions, map[string]any{
					"namespace": namespaceFilter,
				})
			}
		}
		if f.Category != nil && *f.Category != "" {
			filterConditions = append(filterConditions, map[string]any{
				"category": map[string]any{
					"$eq": *f.Category,
				},
			})
		}
		if f.Lifespan != nil && *f.Lifespan != "" {
			filterConditions = append(filterConditions, map[string]any{
				"lifespan": map[string]any{
					"$eq": *f.Lifespan,
				},
			})
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
					"$eq": *f.Status,
				},
			})
		}
		if len(filterConditions) > 1 {
			metadataFilter = map[string]any{
				"$and": filterConditions,
			}
		} else if len(filterConditions) == 1 {
			metadataFilter = filterConditions[0]
		}
	}

	var filterStruct *structpb.Struct
	if metadataFilter != nil {
		var err error
		filterStruct, err = structpb.NewStruct(metadataFilter)
		if err != nil {
			return nil, fmt.Errorf("failed to create filter struct: %w", err)
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

	filterConditions := []map[string]any{}
	if metadataFilter != nil {
		if namespaceFilter, ok := metadataFilter["namespace"]; ok {
			filterConditions = append(filterConditions, map[string]any{
				"namespace": namespaceFilter,
			})
		}
	}
	if f.Category != nil && *f.Category != "" {
		filterConditions = append(filterConditions, map[string]any{
			"category": map[string]any{
				"$eq": *f.Category,
			},
		})
	}
	if f.Lifespan != nil && *f.Lifespan != "" {
		filterConditions = append(filterConditions, map[string]any{
			"lifespan": map[string]any{
				"$eq": *f.Lifespan,
			},
		})
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
				"$eq": *f.Status,
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
		metadataFilter = filterConditions[0]
	}

	filterStruct, err := structpb.NewStruct(metadataFilter)
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

	if andConditions, ok := filterMap["$and"].([]interface{}); ok {
		for _, condition := range andConditions {
			condMap, ok := condition.(map[string]interface{})
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

func (d *pineconeVectorClient) matchesSingleCondition(metadataMap map[string]interface{}, condMap map[string]interface{}) bool {
	for field, operator := range condMap {
		opMap, ok := operator.(map[string]interface{})
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
	if len(v) == 0 {
		d.logger.Warn("no vector provided to updateVector")
		return false, fmt.Errorf("no vector provided")
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
