package memory

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/MelloB1989/karma/ai/parser"
	"github.com/MelloB1989/karma/utils"
	"go.uber.org/zap"
)

type m struct {
	Operation               string           `json:"operation"`
	Id                      *string          `json:"id"`
	Category                MemoryCategory   `json:"category"`
	Summary                 string           `json:"summary"`
	RawText                 string           `json:"raw_text"`
	Importance              int              `json:"importance"`
	Mutability              MemoryMutability `json:"mutability"`
	Lifespan                MemoryLifespan   `json:"lifespan"`
	ForgetScore             float64          `json:"forget_score"`
	Status                  MemoryStatus     `json:"status"`
	SupersedesCanonicalKeys []string         `json:"supersedes_canonical_keys"`
	Metadata                json.RawMessage  `json:"metadata"`
	SupersedesMemoryId      *string          `json:"supersedes_memory_id,omitempty"`
}

type memoriesWrapper struct {
	Memories []m `json:"memories"`
}

func (k *KarmaMemory) ingest(convo struct {
	UserMessage          string
	AIResponse           string
	CurrentMemoryContext string
}) error {
	p := parser.NewParser(parser.WithAIClient(k.memoryAI), parser.WithDebug(false))

	prompt := fmt.Sprintf(`Extract memories from this conversation.
CurrentMemoryContext: %s
UserMessage: %s
AIResponse: %s
Return a JSON object with a "memories" array containing all extracted memories.
If no memories should be stored, return: {"memories": []}`, convo.CurrentMemoryContext, convo.UserMessage, convo.AIResponse)

	var wrapper memoriesWrapper
	if _, _, err := p.Parse(prompt, "", &wrapper); err != nil {
		k.logger.Error("karma_memory: failed to parse memories", zap.Error(err))
		return err
	}

	if len(wrapper.Memories) == 0 {
		k.logger.Debug("karma_memory: no memories extracted from conversation")
		return nil
	}

	k.logger.Debug("karma_memory: extracted memories", zap.Int("count", len(wrapper.Memories)))

	var vc []v
	var vd []string
	var memoriesIdsToSupersede []string
	var newMemories []Memory

	categoriesToInvalidate := make(map[MemoryCategory]bool)

	for _, memory := range wrapper.Memories {
		now := time.Now()
		memoryId := utils.GenerateID(7)

		if memory.Operation == "delete" || memory.Operation == "update" {
			if memory.Id != nil && *memory.Id != "" {
				memoryId = *memory.Id
			} else {
				foundId := k.findSimilarMemory(memory.Summary, memory.Category)
				if foundId != "" {
					if memory.Operation == "delete" {
						memoryId = foundId
					} else {
						memory.Operation = "create"
						memoriesIdsToSupersede = append(memoriesIdsToSupersede, foundId)
						k.logger.Debug("karma_memory: converting update to create+supersede",
							zap.String("supersededId", foundId))
					}
				} else {
					if memory.Operation == "delete" {
						k.logger.Warn("karma_memory: delete operation without ID and no similar memory found",
							zap.String("summary", memory.Summary))
						continue
					}
					memory.Operation = "create"
				}
			}
		}

		if memory.SupersedesMemoryId != nil && *memory.SupersedesMemoryId != "" {
			memoriesIdsToSupersede = append(memoriesIdsToSupersede, *memory.SupersedesMemoryId)
		}

		mem := &Memory{
			Category:                memory.Category,
			Summary:                 memory.Summary,
			RawText:                 memory.RawText,
			Importance:              memory.Importance,
			Mutability:              memory.Mutability,
			Lifespan:                memory.Lifespan,
			ForgetScore:             memory.ForgetScore,
			Status:                  memory.Status,
			SupersedesCanonicalKeys: memory.SupersedesCanonicalKeys,
			Metadata:                memory.Metadata,
			Id:                      memoryId,
			CreatedAt:               now,
			UpdatedAt:               now,
			ExpiresAt:               computeExpiry(now, memory.Lifespan, memory.ForgetScore),
			Namespace:               k.scope,
			SubjectKey:              k.userID,
		}

		if mem.Status == "" {
			mem.Status = StatusActive
		}

		categoriesToInvalidate[memory.Category] = true

		if memory.Operation == "delete" {
			vd = append(vd, memoryId)
			k.logger.Debug("karma_memory: queued memory for deletion",
				zap.String("id", memoryId),
				zap.String("summary", memory.Summary))
			continue
		}

		embeddingText := memory.Summary
		if memory.RawText != "" {
			embeddingText = memory.RawText + " " + memory.Summary
		}

		switch k.memorydb.currentService {
		case VectorServiceUpstash:
			embeddings, err := k.getEmbeddings(embeddingText)
			if err != nil {
				k.logger.Error("karma_memory: failed to generate embeddings",
					zap.String("memoryID", mem.Id),
					zap.Error(err))
				continue
			}
			if memory.Operation == "create" {
				vc = append(vc, v{memories: *mem, vector: embeddings})
				newMemories = append(newMemories, *mem)
			} else if memory.Operation == "update" {
				k.memorydb.client.updateVector(*mem, embeddings)
				newMemories = append(newMemories, *mem)
			}

		case VectorServicePinecone:
			if memory.Operation == "create" {
				vc = append(vc, v{memories: *mem})
				newMemories = append(newMemories, *mem)
			} else if memory.Operation == "update" {
				k.memorydb.client.updateVector(*mem)
				newMemories = append(newMemories, *mem)
			}
		}
	}

	for _, supersededId := range memoriesIdsToSupersede {
		if err := k.markMemoryAsSuperseded(supersededId); err != nil {
			k.logger.Warn("karma_memory: failed to mark memory as superseded",
				zap.String("memoryId", supersededId),
				zap.Error(err))
		}
	}

	if len(vc) > 0 {
		if err := k.memorydb.client.upsertVectors(vc); err != nil {
			k.logger.Error("karma_memory: failed to upsert vectors", zap.Error(err))
		} else {
			k.logger.Info("karma_memory: upserted memories", zap.Int("count", len(vc)))
		}
	}

	if len(vd) > 0 {
		if count, err := k.memorydb.client.deleteVectors(vd); err != nil {
			k.logger.Error("karma_memory: failed to delete vectors", zap.Error(err))
		} else {
			k.logger.Info("karma_memory: deleted memories", zap.Int("count", count))
		}
	}

	if k.IsCacheEnabled() {
		ctx := context.Background()

		for category := range categoriesToInvalidate {
			if err := k.cache.InvalidateCategoryCache(ctx, k.userID, k.scope, category); err != nil {
				k.logger.Warn("karma_memory: failed to invalidate category cache",
					zap.String("category", string(category)),
					zap.Error(err))
			} else {
				k.logger.Debug("karma_memory: invalidated category cache",
					zap.String("category", string(category)))
			}
		}

		if len(newMemories) > 0 {
			k.cacheNewMemories(ctx, newMemories)
		}
	}

	return nil
}

func (k *KarmaMemory) cacheNewMemories(ctx context.Context, memories []Memory) {
	categoryMap := make(map[MemoryCategory][]Memory)

	for _, mem := range memories {
		switch mem.Category {
		case CategoryRule, CategoryFact, CategorySkill, CategoryContext:
			categoryMap[mem.Category] = append(categoryMap[mem.Category], mem)
		}
	}

	for category, mems := range categoryMap {
		if len(mems) > 0 {
			if err := k.cache.CacheMemoriesByCategory(ctx, k.userID, k.scope, category, mems); err != nil {
				k.logger.Warn("karma_memory: failed to cache new memories",
					zap.String("category", string(category)),
					zap.Error(err))
			} else {
				k.logger.Debug("karma_memory: cached new memories",
					zap.String("category", string(category)),
					zap.Int("count", len(mems)))
			}
		}
	}

	// Also cache all memories for dynamic filtering in conscious mode
	if len(memories) > 0 {
		if err := k.cache.CacheAllMemories(ctx, k.userID, k.scope, memories); err != nil {
			k.logger.Warn("karma_memory: failed to cache all memories for dynamic filtering",
				zap.Error(err))
		} else {
			k.logger.Debug("karma_memory: cached all memories for dynamic filtering",
				zap.Int("count", len(memories)))
		}
	}
}
