package memory

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/MelloB1989/karma/ai/parser"
	"github.com/upstash/vector-go"
)

func (k *KarmaMemory) generateSearchQuery(userPrompt string) (filters, error) {
	var filters filters
	p := parser.NewParser(parser.WithAIClient(k.retrievalAI))
	_, _, err := p.Parse(fmt.Sprintf("%s", userPrompt), "", &filters)
	return filters, err
}

func (k *KarmaMemory) selectRelevantMemories(vectorResults []vector.VectorScore, rules []map[string]any, topK int) []Memory {
	var result []Memory

	type scoredMemory struct {
		memory Memory
		score  float64
	}

	var scored []scoredMemory
	for _, vr := range vectorResults {
		if vr.Metadata == nil {
			continue
		}

		mem := metadataToMemory(vr.Metadata, vr.Id)
		if mem.Status != StatusActive || mem.Category == CategoryRule {
			continue
		}

		scored = append(scored, scoredMemory{memory: mem, score: float64(vr.Score)})
	}

	for i := 0; i < len(scored); i++ {
		for j := i + 1; j < len(scored); j++ {
			if scored[j].score > scored[i].score {
				scored[i], scored[j] = scored[j], scored[i]
			}
		}
	}

	if len(scored) > topK {
		scored = scored[:topK]
	}

	for _, s := range scored {
		result = append(result, s.memory)
	}

	for _, r := range rules {
		mem := metadataToMemory(r, "")
		if mem.Status == StatusActive {
			result = append(result, mem)
		}
	}

	return result
}

func metadataToMemory(metadata map[string]any, id string) Memory {
	mem := Memory{Id: id}

	if v, ok := metadata["id"].(string); ok && id == "" {
		mem.Id = v
	}
	if v, ok := metadata["summary"].(string); ok {
		mem.Summary = v
	}
	if v, ok := metadata["category"].(string); ok {
		mem.Category = MemoryCategory(v)
	}
	if v, ok := metadata["status"].(string); ok {
		mem.Status = MemoryStatus(v)
	} else {
		mem.Status = StatusActive
	}
	if v, ok := metadata["importance"].(float64); ok {
		mem.Importance = int(v)
	}
	if v, ok := metadata["lifespan"].(string); ok {
		mem.Lifespan = MemoryLifespan(v)
	}

	return mem
}

func (k *KarmaMemory) formatContext(memories []Memory, maxTokens int) string {
	if len(memories) == 0 {
		return ""
	}

	var rules []string
	var otherMemories []string

	// Separate rules from other memories
	for _, mem := range memories {
		if mem.Category == CategoryRule {
			// Condense rule: trim and shorten if needed
			rule := strings.TrimSpace(mem.Summary)
			if len(rule) > 80 {
				rule = rule[:77] + "..."
			}
			rules = append(rules, rule)
		} else {
			otherMemories = append(otherMemories, strings.TrimSpace(mem.Summary))
		}
	}

	var sb strings.Builder
	currentTokens := 0

	// Always include condensed rules first
	if len(rules) > 0 {
		sb.WriteString("[Rules] ")
		rulesStr := strings.Join(rules, "; ")
		sb.WriteString(rulesStr)
		sb.WriteString("\n")
		currentTokens += len(rulesStr) / 5
	}

	// Add other memories if token budget allows
	for _, summary := range otherMemories {
		entryTokens := len(summary) / 5
		if currentTokens+entryTokens > maxTokens {
			break
		}
		sb.WriteString("- ")
		sb.WriteString(summary)
		sb.WriteString("\n")
		currentTokens += entryTokens
	}

	return sb.String()
}

func (k *KarmaMemory) formatContextForIngest(memories []Memory) string {
	if len(memories) == 0 {
		return ""
	}

	var ms []string

	for _, mem := range memories {
		if mem.ExpiresAt == nil {
			ms = append(ms, fmt.Sprintf("MemoryId: %s; MemoryCreatedAt: %s; MemoryExpiry: noExpiry; MemorySummary: %s", mem.Id, mem.CreatedAt.Format(time.RFC3339), strings.TrimSpace(mem.Summary)))
		} else {
			ms = append(ms, fmt.Sprintf("MemoryId: %s; MemoryCreatedAt: %s; MemoryExpiry: %s; MemorySummary: %s", mem.Id, mem.CreatedAt.Format(time.RFC3339), mem.ExpiresAt.Format(time.RFC3339), strings.TrimSpace(mem.Summary)))
		}
	}

	return strings.Join(ms, "\n")
}

func computeExpiry(baseTime time.Time, lifespan MemoryLifespan, forgetScore float64) *time.Time {
	var baseDuration time.Duration

	switch lifespan {
	case LifespanShortTerm:
		baseDuration = 7 * 24 * time.Hour
	case LifespanMidTerm:
		baseDuration = 90 * 24 * time.Hour
	case LifespanLongTerm:
		baseDuration = 365 * 24 * time.Hour
	case LifespanLifelong:
		return nil
	default:
		return nil
	}

	adjustedDuration := time.Duration(float64(baseDuration) * (1.0 - forgetScore))
	expiryTime := baseTime.Add(adjustedDuration)
	return &expiryTime
}

func (k *KarmaMemory) getEmbeddings(text string) ([]float32, error) {
	resp, err := k.embeddingAI.GetEmbeddings(text)
	if err != nil {
		return nil, fmt.Errorf("embedding AI failed: %w", err)
	}
	return resp.GetEmbeddingsFloat32(), nil
}

func (m *Memory) ToMap() (map[string]any, error) {
	out := make(map[string]any)

	out["id"] = m.Id
	out["subject_key"] = m.SubjectKey
	out["namespace"] = m.Namespace
	out["category"] = string(m.Category)
	out["summary"] = m.Summary
	out["raw_text"] = m.RawText
	out["importance"] = m.Importance
	out["mutability"] = string(m.Mutability)
	out["lifespan"] = string(m.Lifespan)
	out["forget_score"] = m.ForgetScore
	out["status"] = string(m.Status)

	var supersedes any
	if len(m.SupersedesCanonicalKeys) > 0 {
		if err := json.Unmarshal(m.SupersedesCanonicalKeys, &supersedes); err != nil {
			return nil, fmt.Errorf("invalid SupersedesCanonicalKeys JSON: %w", err)
		}
	}
	out["supersedes_canonical_keys"] = supersedes

	if m.SupersededById != nil {
		out["superseded_by_id"] = *m.SupersededById
	} else {
		out["superseded_by_id"] = nil
	}

	var metadata any
	if len(m.Metadata) > 0 {
		if err := json.Unmarshal(m.Metadata, &metadata); err != nil {
			return nil, fmt.Errorf("invalid metadata JSON: %w", err)
		}
	}
	out["metadata"] = metadata

	out["created_at"] = m.CreatedAt
	out["updated_at"] = m.UpdatedAt

	if m.ExpiresAt != nil {
		out["expires_at"] = *m.ExpiresAt
	} else {
		out["expires_at"] = nil
	}

	out["entity_relationships"] = m.EntityRelationships

	return out, nil
}

func ptrStr(s string) *string {
	return &s
}
