package memory

import (
	"encoding/json"
	"fmt"

	"github.com/MelloB1989/karma/ai/parser"
)

type m struct {
	Category                MemoryCategory   `json:"category" description:"High-level category of the memory. One of: 'fact', 'preference', 'skill', 'context', 'rule', 'entity', 'episodic'.\n- fact: objective info about the subject (e.g. 'I use PostgreSQL for databases').\n- preference: likes/dislikes or choices (e.g. 'I prefer clean, readable code', 'I like Adidas').\n- skill: abilities and expertise (e.g. 'Experienced with FastAPI').\n- context: project or situation info (e.g. 'Working on e-commerce platform').\n- rule: behavioral guidelines or constraints (e.g. 'Always write tests first', 'Never reply in Telugu').\n- entity: people/organizations in subject's life (e.g. 'Jane is my mom', 'Karthik is my lead developer').\n- episodic: specific events in time (e.g. 'Yesterday we deployed the new version')."`
	Summary                 string           `json:"summary" description:"Short, normalized natural-language summary of the memory. This should capture the essence in one concise sentence. Example: 'User uses PostgreSQL as their primary database.' or 'User no longer likes Adidas.'"`
	RawText                 string           `json:"raw_text" description:"Original or minimally normalized text span from which this memory was derived. Useful for audit and re-embedding. Example: 'I use PostgreSQL for databases' or 'I don't like Adidas anymore'."`
	CanonicalKey            *string          `json:"canonical_key,omitempty" description:"Optional stable semantic key that identifies the concept this memory refers to. Used mainly for facts, preferences, and skills to support updates and supersession. Examples: 'db.primary', 'brand.adidas', 'interest.coding', 'stack.frontend', 'language.primary'. Nil for purely episodic memories where no canonical dimension is needed."`
	Value                   *string          `json:"value,omitempty" description:"Optional normalized value for the canonical_key. Examples: for canonical_key 'db.primary' → 'postgresql'; for 'brand.adidas' → 'like' or 'dislike'; for 'interest.coding' → 'love'; for 'stack.frontend' → 'typescript+react'. Nil for memories that do not express a key-value fact (e.g. some episodic entries)."`
	Importance              int              `json:"importance" description:"Importance score from 1 to 5 indicating how critical this memory is for personalization and future recall.\n1 = low value, rarely needed.\n3 = normal.\n5 = very important, central to subject identity, long-term goals, or behavior.\nExamples: 'Never reply in Telugu' → 5; 'I like this one song' → 2 or 3."`
	Mutability              MemoryMutability `json:"mutability" description:"Indicates whether this memory is expected to change over time.\nCommon values (via MemoryMutability enum):\n- 'immutable': Core facts that typically do not change (e.g. 'I was born in 1995').\n- 'mutable': Preferences or settings that may change (e.g. 'I like Adidas', 'I prefer TypeScript').\nUsed to decide how to handle updates and conflict resolution."`
	Lifespan                MemoryLifespan   `json:"lifespan" description:"Intended lifespan category for this memory, typically via MemoryLifespan enum. One of:\n- 'short_term': ephemeral or near-term context (e.g. 'This week I am traveling').\n- 'mid_term': medium-lived preferences or context (e.g. 'Currently using Tailwind for styling').\n- 'long_term': persistent facts/skills (e.g. 'I use PostgreSQL', 'Experienced with FastAPI').\n- 'lifelong': identity-level traits (e.g. 'I love coding', 'I enjoy cooking').\nCombined with ForgetScore and ExpiresAt for decay and garbage collection strategies."`
	ForgetScore             float64          `json:"forget_score" description:"Float from 0.0 to 1.0 indicating how forgettable this memory is.\n0.0 = effectively never forget (highly critical).\n1.0 = very forgettable (highly ephemeral).\nGuidelines: identity-level traits and core rules ≈ 0.0–0.1; stable facts/skills ≈ 0.1–0.3; changing preferences ≈ 0.4–0.7; short-lived context ≈ 0.7–1.0."`
	Status                  MemoryStatus     `json:"status" description:"Current lifecycle state of the memory, usually via MemoryStatus enum. Typical values:\n- 'active': Memory is current and should be considered during retrieval.\n- 'superseded': Memory has been replaced by a newer memory for the same canonical_key (e.g. old preference 'I like Adidas' after 'I don't like Adidas anymore').\n- 'deleted': Soft-deleted or logically removed memory.\nRetrieval layers generally filter to active memories by default."`
	SupersedesCanonicalKeys []string         `json:"supersedes_canonical_keys" db:"supersedes_canonical_keys" description:"List of canonical_keys that this memory explicitly supersedes. Used primarily for mutable categories like preferences.\nExample: when creating a new preference 'I don't like Adidas anymore', this memory might have canonical_key 'brand.adidas' and SupersedesCanonicalKeys containing ['brand.adidas'], signaling that any previous Adidas preference should be marked as superseded."`
	SupersededByID          *string          `json:"superseded_by_id,omitempty" description:"Optional ID of a newer memory that superseded this one. Set when this memory's Status transitions to 'superseded'. Allows forward linkage from old memories to the new canonical version. Example: the old 'I like Adidas' memory points to the newer 'I don't like Adidas anymore' memory."`
	Metadata                json.RawMessage  `json:"metadata" db:"metadata" description:"Arbitrary additional metadata stored as raw JSON. Can include tags, source, tool information, timestamps, app-specific fields, or vector service IDs. Examples: {\"tags\":[\"database\",\"technology\"],\"source\":\"chat\"} or {\"tool_name\":\"github_agent\",\"message_id\":\"abc123\"}."`
	ShouldVectorize         bool             `json:"should_vectorize" description:"Indicates whether this memory should be vectorized and stored in a vector database. Useful for mutable categories like preferences or for memories that need to be searched for similarity."`
}

func (k *KarmaMemory) ingest(convo struct {
	UserMessage string
	AIResponse  string
}) error {
	parser := parser.NewParser(parser.WithAIClient(k.memoryAI))
	var memories []m
	if _, _, err := parser.Parse(fmt.Sprintf("UserMessage: %s\nAIResponse: %s\nGenerate the memory array\njson_mode = true", convo.UserMessage, convo.AIResponse), "", &memories); err != nil || len(memories) == 0 {
		return err
	}

	for _, memory := range memories {
		mem := &Memory{
			Category:                memory.Category,
			Summary:                 memory.Summary,
			RawText:                 memory.RawText,
			CanonicalKey:            memory.CanonicalKey,
			Value:                   memory.Value,
			Importance:              memory.Importance,
			Mutability:              memory.Mutability,
			Lifespan:                memory.Lifespan,
			ForgetScore:             memory.ForgetScore,
			Status:                  memory.Status,
			SupersedesCanonicalKeys: memory.SupersedesCanonicalKeys,
			SupersededByID:          memory.SupersededByID,
			Metadata:                memory.Metadata,
		}
		k.memorydb.createMemory(mem)
	}

	return nil
}
