package memory

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/MelloB1989/karma/ai/parser"
	"github.com/upstash/vector-go"
	"go.uber.org/zap"
)

func normalizeSummary(s string) string {
	s = strings.ToLower(s)
	reg := regexp.MustCompile(`[.,!?;:'"\-_()[\]{}]`)
	s = reg.ReplaceAllString(s, "")
	s = strings.Join(strings.Fields(s), " ")
	return strings.TrimSpace(s)
}

func (k *KarmaMemory) generateSearchQuery(userPrompt string) (filters, error) {
	var filters filters
	p := parser.NewParser(parser.WithAIClient(k.retrievalAI))
	_, _, err := p.Parse(fmt.Sprintf("%s", userPrompt), "", &filters)
	return filters, err
}

func (k *KarmaMemory) selectRelevantMemories(vectorResults []vector.VectorScore, rules []map[string]any, topK int) []Memory {
	var result []Memory

	seenIds := make(map[string]bool)
	seenSummaries := make(map[string]bool)

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

		if mem.Id != "" && seenIds[mem.Id] {
			continue
		}

		normalizedSummary := normalizeSummary(mem.Summary)
		if normalizedSummary != "" && seenSummaries[normalizedSummary] {
			continue
		}

		if mem.Id != "" {
			seenIds[mem.Id] = true
		}
		if normalizedSummary != "" {
			seenSummaries[normalizedSummary] = true
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
		if mem.Status != StatusActive {
			continue
		}

		if mem.Id != "" && seenIds[mem.Id] {
			continue
		}

		normalizedSummary := normalizeSummary(mem.Summary)
		if normalizedSummary != "" && seenSummaries[normalizedSummary] {
			continue
		}

		if mem.Id != "" {
			seenIds[mem.Id] = true
		}
		if normalizedSummary != "" {
			seenSummaries[normalizedSummary] = true
		}

		result = append(result, mem)
	}

	return result
}

func metadataToMemory(metadata map[string]any, id string) Memory {
	mem := Memory{Id: id}

	if v, ok := metadata["id"].(string); ok && id == "" {
		mem.Id = v
	}
	if v, ok := metadata["_id"].(string); ok && mem.Id == "" {
		mem.Id = v
	}
	if v, ok := metadata["subject_key"].(string); ok {
		mem.SubjectKey = v
	}
	if v, ok := metadata["namespace"].(string); ok {
		mem.Namespace = v
	}
	// Check both "summary" and "text" fields (Pinecone uses "text" for integrated records)
	if v, ok := metadata["summary"].(string); ok {
		mem.Summary = v
	}
	if v, ok := metadata["text"].(string); ok && mem.Summary == "" {
		mem.Summary = v
	}
	if v, ok := metadata["category"].(string); ok {
		mem.Category = MemoryCategory(v)
	}
	if v, ok := metadata["raw_text"].(string); ok {
		mem.RawText = v
	}
	if v, ok := metadata["forget_score"].(float64); ok {
		mem.ForgetScore = v
	}
	// Handle metadata as string (from Pinecone) or json.RawMessage
	if v, ok := metadata["metadata"].(string); ok {
		mem.Metadata = json.RawMessage(v)
	} else if v, ok := metadata["metadata_json"].(string); ok {
		mem.Metadata = json.RawMessage(v)
	} else if v, ok := metadata["metadata"].(json.RawMessage); ok {
		mem.Metadata = v
	}
	if v, ok := metadata["mutability"].(string); ok {
		mem.Mutability = MemoryMutability(v)
	}
	// Handle supersedes_canonical_keys as JSON string or slice
	if v, ok := metadata["supersedes_canonical_keys"].(string); ok {
		var keys []string
		if err := json.Unmarshal([]byte(v), &keys); err == nil {
			mem.SupersedesCanonicalKeys = keys
		}
	} else if v, ok := metadata["supersedes_canonical_keys_json"].(string); ok {
		var keys []string
		if err := json.Unmarshal([]byte(v), &keys); err == nil {
			mem.SupersedesCanonicalKeys = keys
		}
	} else if v, ok := metadata["supersedes_canonical_keys"].([]string); ok {
		mem.SupersedesCanonicalKeys = v
	} else if v, ok := metadata["supersedes_canonical_keys"].([]interface{}); ok {
		keys := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok {
				keys = append(keys, s)
			}
		}
		mem.SupersedesCanonicalKeys = keys
	}
	// Handle time fields - can be time.Time or RFC3339 string
	if v, ok := metadata["created_at"].(time.Time); ok {
		mem.CreatedAt = v
	} else if v, ok := metadata["created_at"].(string); ok {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			mem.CreatedAt = t
		}
	}
	if v, ok := metadata["updated_at"].(time.Time); ok {
		mem.UpdatedAt = v
	} else if v, ok := metadata["updated_at"].(string); ok {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			mem.UpdatedAt = t
		}
	}
	if v, ok := metadata["expires_at"].(*time.Time); ok {
		mem.ExpiresAt = v
	} else if v, ok := metadata["expires_at"].(string); ok {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			mem.ExpiresAt = &t
		}
	}
	// Handle entity_relationships as string or slice
	if v, ok := metadata["entity_relationships_json"].(string); ok {
		var rels []EntityRelationship
		if err := json.Unmarshal([]byte(v), &rels); err == nil {
			mem.EntityRelationships = rels
		}
	} else if v, ok := metadata["entity_relationships"].([]EntityRelationship); ok {
		mem.EntityRelationships = v
	}
	if v, ok := metadata["status"].(string); ok {
		mem.Status = MemoryStatus(v)
	} else {
		mem.Status = StatusActive
	}
	if v, ok := metadata["importance"].(float64); ok {
		mem.Importance = int(v)
	} else if v, ok := metadata["importance"].(int); ok {
		mem.Importance = v
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

	ms = append(ms, "EXISTING_MEMORIES (use these IDs for updates/supersedes):")

	for _, mem := range memories {
		var entry string
		if mem.ExpiresAt == nil {
			entry = fmt.Sprintf("[ID:%s] [Category:%s] [Status:%s] %s",
				mem.Id,
				string(mem.Category),
				string(mem.Status),
				strings.TrimSpace(mem.Summary))
		} else {
			entry = fmt.Sprintf("[ID:%s] [Category:%s] [Status:%s] [Expires:%s] %s",
				mem.Id,
				string(mem.Category),
				string(mem.Status),
				mem.ExpiresAt.Format("2006-01-02"),
				strings.TrimSpace(mem.Summary))
		}
		ms = append(ms, entry)
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
		supersedes = m.SupersedesCanonicalKeys
	}
	out["supersedes_canonical_keys"] = supersedes

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

func (k *KarmaMemory) findSimilarMemory(summary string, category MemoryCategory) string {
	categoryStr := string(category)
	existingMemories, err := k.memorydb.client.queryVectorByMetadata(filters{
		Category: &categoryStr,
	})
	if err != nil {
		k.logger.Debug("karmaMemory: failed to query existing memories for similarity check",
			zap.Error(err))
		return ""
	}

	normalizedNew := normalizeSummary(summary)

	for _, mem := range existingMemories {
		existingSummary, ok := mem["summary"].(string)
		if !ok {
			continue
		}

		normalizedExisting := normalizeSummary(existingSummary)

		if isSimilarMemory(normalizedNew, normalizedExisting) {
			if id, ok := mem["id"].(string); ok && id != "" {
				return id
			}
			if id, ok := mem["_id"].(string); ok && id != "" {
				return id
			}
		}
	}

	return ""
}

func isSimilarMemory(new, existing string) bool {
	if new == existing {
		return true
	}

	if strings.Contains(new, existing) || strings.Contains(existing, new) {
		return true
	}

	newWords := extractKeyWords(new)
	existingWords := extractKeyWords(existing)

	if len(newWords) == 0 || len(existingWords) == 0 {
		return false
	}

	overlap := 0
	for _, word := range newWords {
		for _, existingWord := range existingWords {
			if word == existingWord {
				overlap++
				break
			}
		}
	}

	minLen := len(newWords)
	if len(existingWords) < minLen {
		minLen = len(existingWords)
	}

	if minLen > 0 && float64(overlap)/float64(minLen) >= 0.6 {
		return true
	}

	return false
}

//go:embed stopwords
var stopwordsFile string

func loadStopWords() map[string]bool {
	stopWords := make(map[string]bool)
	lines := strings.Split(stopwordsFile, "\n")
	for _, w := range lines {
		w = strings.TrimSpace(w)
		if w != "" {
			stopWords[w] = true
		}
	}
	return stopWords
}

func extractKeyWords(s string) []string {
	stopWords := loadStopWords()

	words := strings.Fields(s)
	var keyWords []string

	for _, word := range words {
		word = strings.TrimSpace(strings.ToLower(word))
		if len(word) > 2 && !stopWords[word] {
			keyWords = append(keyWords, word)
		}
	}

	return keyWords
}

func (k *KarmaMemory) markMemoryAsSuperseded(memoryId string) error {
	supersededMem := Memory{
		Id:        memoryId,
		Status:    StatusSuperseded,
		UpdatedAt: time.Now(),
	}

	switch k.memorydb.currentService {
	case VectorServiceUpstash:
		_, err := k.memorydb.client.updateVector(supersededMem)
		return err
	case VectorServicePinecone:
		_, err := k.memorydb.client.updateVector(supersededMem)
		return err
	}

	return nil
}

func intPtr(i int) *int {
	return &i
}
