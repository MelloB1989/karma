package memory

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/MelloB1989/karma/ai/parser"
	"github.com/MelloB1989/karma/utils"
	"go.uber.org/zap"
)

type m struct {
	Operation               string           `json:"operation" description:"Operation type of the memory. One of: 'create', 'update', 'delete'.\n- create: new memory being added.\n- update: existing memory being modified.\n- delete: memory being removed."`
	Id                      *string          `json:"id" description:"Unique identifier for the memory object. Include this when using update or delete operations to specify which memory to modify or remove."`
	Category                MemoryCategory   `json:"category" description:"High-level category of the memory. One of: 'fact', 'preference', 'skill', 'context', 'rule', 'entity', 'episodic'.\n- fact: objective info about the subject (e.g. 'I use PostgreSQL for databases').\n- preference: likes/dislikes or choices (e.g. 'I prefer clean, readable code', 'I like Adidas').\n- skill: abilities and expertise (e.g. 'Experienced with FastAPI').\n- context: project or situation info (e.g. 'Working on e-commerce platform').\n- rule: behavioral guidelines or constraints (e.g. 'Always write tests first', 'Never reply in Telugu').\n- entity: people/organizations in subject's life (e.g. 'Jane is my mom', 'Karthik is my lead developer').\n- episodic: specific events in time (e.g. 'Yesterday we deployed the new version')."`
	Summary                 string           `json:"summary" description:"Short, normalized natural-language summary of the memory. This should capture the essence in one concise sentence. Example: 'User uses PostgreSQL as their primary database.' or 'User no longer likes Adidas.'"`
	RawText                 string           `json:"raw_text" description:"Original or minimally normalized text span from which this memory was derived. Useful for audit and re-embedding. Example: 'I use PostgreSQL for databases' or 'I don't like Adidas anymore'."`
	Importance              int              `json:"importance" description:"Importance score from 1 to 5 indicating how critical this memory is for personalization and future recall.\n1 = low value, rarely needed.\n3 = normal.\n5 = very important, central to subject identity, long-term goals, or behavior.\nExamples: 'Never reply in Telugu' → 5; 'I like this one song' → 2 or 3."`
	Mutability              MemoryMutability `json:"mutability" description:"Indicates whether this memory is expected to change over time.\nCommon values (via MemoryMutability enum):\n- 'immutable': Core facts that typically do not change (e.g. 'I was born in 1995').\n- 'mutable': Preferences or settings that may change (e.g. 'I like Adidas', 'I prefer TypeScript').\nUsed to decide how to handle updates and conflict resolution."`
	Lifespan                MemoryLifespan   `json:"lifespan" description:"Intended lifespan category for this memory, typically via MemoryLifespan enum. One of:\n- 'short_term': ephemeral or near-term context (e.g. 'This week I am traveling').\n- 'mid_term': medium-lived preferences or context (e.g. 'Currently using Tailwind for styling').\n- 'long_term': persistent facts/skills (e.g. 'I use PostgreSQL', 'Experienced with FastAPI').\n- 'lifelong': identity-level traits (e.g. 'I love coding', 'I enjoy cooking').\nCombined with ForgetScore and ExpiresAt for decay and garbage collection strategies."`
	ForgetScore             float64          `json:"forget_score" description:"Float from 0.0 to 1.0 indicating how forgettable this memory is.\n0.0 = effectively never forget (highly critical).\n1.0 = very forgettable (highly ephemeral).\nGuidelines: identity-level traits and core rules ≈ 0.0–0.1; stable facts/skills ≈ 0.1–0.3; changing preferences ≈ 0.4–0.7; short-lived context ≈ 0.7–1.0."`
	Status                  MemoryStatus     `json:"status" description:"Current lifecycle state of the memory, usually via MemoryStatus enum. Typical values:\n- 'active': Memory is current and should be considered during retrieval.\n- 'superseded': Memory has been replaced by a newer memory for the same canonical_key (e.g. old preference 'I like Adidas' after 'I don't like Adidas anymore').\n- 'deleted': Soft-deleted or logically removed memory.\nRetrieval layers generally filter to active memories by default."`
	SupersedesCanonicalKeys []string         `json:"supersedes_canonical_keys" db:"supersedes_canonical_keys" description:"List of canonical_keys that this memory explicitly supersedes. Used primarily for mutable categories like preferences.\nExample: when creating a new preference 'I don't like Adidas anymore', this memory might have canonical_key 'brand.adidas' and SupersedesCanonicalKeys containing ['brand.adidas'], signaling that any previous Adidas preference should be marked as superseded."`
	Metadata                json.RawMessage  `json:"metadata" db:"metadata" description:"Arbitrary additional metadata stored as raw JSON. Can include tags, source, tool information, timestamps, app-specific fields, or vector service IDs. Examples: {\"tags\":[\"database\",\"technology\"],\"source\":\"chat\"} or {\"tool_name\":\"github_agent\",\"message_id\":\"abc123\"}."`
}

type memoriesWrapper struct {
	Memories []m `json:"memories" description:"Array of extracted memories from the conversation"`
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
		k.logger.Error("karmaMemory: failed to parse memories",
			zap.Error(err))
		return err
	}

	if len(wrapper.Memories) == 0 {
		k.logger.Debug("karmaMemory: no memories extracted from conversation")
		return nil
	}

	k.logger.Debug("karmaMemory: extracted memories",
		zap.Int("count", len(wrapper.Memories)))

	var vc []v
	var vd []string

	for _, memory := range wrapper.Memories {
		now := time.Now()

		memoryId := utils.GenerateID(7)
		if memory.Operation == "delete" || memory.Operation == "update" {
			if memory.Id != nil && *memory.Id != "" {
				memoryId = *memory.Id
			} else {
				k.logger.Warn("karmaMemory: delete/update operation without ID, skipping",
					zap.String("operation", memory.Operation),
					zap.String("summary", memory.Summary))
				continue
			}
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

		if memory.Operation == "delete" {
			vd = append(vd, memoryId)
			k.logger.Debug("karmaMemory: queued memory for deletion",
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
				k.logger.Error("karmaMemory: failed to generate embeddings for memory",
					zap.String("memoryID", mem.Id),
					zap.Error(err))
				continue
			}
			if memory.Operation == "create" {
				vc = append(vc, v{
					memories: *mem,
					vector:   embeddings,
				})
			} else if memory.Operation == "update" {
				k.memorydb.client.updateVector(*mem, embeddings)
			}

		case VectorServicePinecone:
			if memory.Operation == "create" {
				vc = append(vc, v{
					memories: *mem,
				})
			} else if memory.Operation == "update" {
				k.memorydb.client.updateVector(*mem)
			}
		}
	}

	if len(vc) > 0 {
		if err := k.memorydb.client.upsertVectors(vc); err != nil {
			k.logger.Error("karmaMemory: failed to upsert vector for memory",
				zap.Error(err))
		}
	}

	if len(vd) > 0 {
		if _, err := k.memorydb.client.deleteVectors(vd); err != nil {
			k.logger.Error("karmaMemory: failed to delete vector for memory",
				zap.Error(err))
		}
	}

	return nil
}
