package memory

import "github.com/upstash/vector-go"

func (k *KarmaMemory) generateSearchQuery(userPrompt string) (string, error) {
	response, err := k.retrievalAI.GenerateFromSinglePrompt(userPrompt)
	if err != nil {
		return "", err
	}
	return response.AIResponse, nil
}

func (k *KarmaMemory) selectRelevantMemories(dbMemories []Memory, vectorResults []vector.VectorScore, topK int) []Memory {
	vectorMemoryMap := make(map[string]float64)
	for _, vr := range vectorResults {
		vectorMemoryMap[vr.Id] = float64(vr.Score)
	}

	type scoredMemory struct {
		memory Memory
		score  float64
	}

	var scored []scoredMemory
	for _, mem := range dbMemories {
		if mem.Status != StatusActive {
			continue
		}

		baseScore := float64(mem.Importance) * 0.3

		if vectorScore, exists := vectorMemoryMap[mem.Id]; exists {
			baseScore += vectorScore * 0.7
		}

		lifespanBoost := 0.0
		switch mem.Lifespan {
		case LifespanLifelong:
			lifespanBoost = 0.3
		case LifespanLongTerm:
			lifespanBoost = 0.2
		case LifespanMidTerm:
			lifespanBoost = 0.1
		}
		baseScore += lifespanBoost

		baseScore *= (1.0 - mem.ForgetScore*0.5)

		scored = append(scored, scoredMemory{memory: mem, score: baseScore})
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

	var result []Memory
	for _, s := range scored {
		result = append(result, s.memory)
	}

	return result
}

func (k *KarmaMemory) formatContext(memories []Memory, maxTokens int) string {
	if len(memories) == 0 {
		return ""
	}

	context := "# Relevant Context\n\n"
	currentTokens := 20

	categoryGroups := make(map[MemoryCategory][]Memory)
	for _, mem := range memories {
		categoryGroups[mem.Category] = append(categoryGroups[mem.Category], mem)
	}

	categoryOrder := []MemoryCategory{
		CategoryRule,
		CategoryPreference,
		CategoryFact,
		CategoryEntity,
		CategorySkill,
		CategoryContext,
		CategoryEpisodic,
	}

	for _, cat := range categoryOrder {
		mems, exists := categoryGroups[cat]
		if !exists || len(mems) == 0 {
			continue
		}

		for _, mem := range mems {
			entry := mem.Summary + "\n"
			entryTokens := len(entry) / 4

			if currentTokens+entryTokens > maxTokens {
				break
			}

			context += entry
			currentTokens += entryTokens
		}

		if currentTokens >= maxTokens {
			break
		}
	}

	return context
}
