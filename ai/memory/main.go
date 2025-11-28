package memory

import (
	"context"
	"sync"

	"github.com/MelloB1989/karma/ai"
	"github.com/MelloB1989/karma/models"
	"github.com/upstash/vector-go"
	"go.uber.org/zap"
)

// RetrievalMode defines how memory context is retrieved
type RetrievalMode string

const (
	// RetrievalModeConscious uses AI to generate dynamic search queries based on user prompt.
	// It analyzes the prompt to determine relevant categories, lifespan, and search terms.
	// Best for: Complex queries where context matters, conversational AI.
	// Tradeoff: Higher latency due to LLM call, more tokens used.
	RetrievalModeConscious RetrievalMode = "conscious"

	// RetrievalModeAuto uses a fixed query strategy with the user prompt as the search text.
	// Always includes non-expired facts, preferences, rules, entities, and context.
	// Best for: Fast retrieval, predictable behavior, lower cost.
	// Tradeoff: Less intelligent filtering, may retrieve less relevant memories.
	RetrievalModeAuto RetrievalMode = "auto"
)

type KarmaMemory struct {
	messagesHistory      models.AIChatHistory
	kai                  *ai.KarmaAI
	memoryAI             *ai.KarmaAI
	embeddingAI          *ai.KarmaAI
	retrievalAI          *ai.KarmaAI
	memorydb             *vectorClient
	cache                MemoryCache
	userID               string
	scope                string
	logger               *zap.Logger
	retrievalMode        RetrievalMode
	currentMemoryContext string
	cacheEnabled         bool
}

func NewKarmaMemory(kai *ai.KarmaAI, userId string, sc ...string) *KarmaMemory {
	// We use "default" scope by default, you can change this by using the useScope function
	scope := "default"
	if len(sc) > 0 {
		scope = sc[0]
	}
	logger, _ := zap.NewProduction()
	memorydb := newVectorClient(userId, scope, logger)

	km := &KarmaMemory{
		messagesHistory: models.AIChatHistory{
			Messages: make([]models.AIMessage, 0),
		},
		kai: kai,
		// Uses GPT-4o-Mini by default, you can change this by using the useMemoryLLM function
		memoryAI: ai.NewKarmaAI(ai.GPT4oMini, ai.OpenAI,
			ai.WithSystemMessage(memoryLLMSystemPrompt),
			ai.WithMaxTokens(memoryLLMMaxTokens),
			ai.WithTemperature(1)),
		// Uses TextEmbedding3Small by default, you can change this by using the useEmbeddingLLM function
		embeddingAI: ai.NewKarmaAI(ai.TextEmbedding3Small, ai.OpenAI),
		retrievalAI: ai.NewKarmaAI(ai.GPT4oMini, ai.OpenAI,
			ai.WithSystemMessage(retrievalLLMSystemPrompt),
			ai.WithMaxTokens(retrievalLLMMaxTokens),
			ai.WithTemperature(0.3)),
		memorydb:      memorydb,
		userID:        userId,
		scope:         scope,
		logger:        logger,
		retrievalMode: RetrievalModeAuto,
	}

	km.EnableMemoryCache()

	return km
}

func (k *KarmaMemory) UseUser(userId string) bool {
	k.userID = userId
	u := k.memorydb.setUser(userId)
	return k.userID == u
}

func (k *KarmaMemory) UseScope(scope string) bool {
	k.scope = scope
	s := k.memorydb.setScope(scope)
	return k.scope == s
}

type MemoryLlmConfig struct {
	MaxTokens   *int
	Temperature *float32
}

func (k *KarmaMemory) UseMemoryLLM(llm ai.BaseModel, provider ai.Provider, extraConfig ...MemoryLlmConfig) {
	var temp float32 = 1
	maxTokens := memoryLLMMaxTokens
	if len(extraConfig) > 0 {
		if extraConfig[0].MaxTokens != nil {
			maxTokens = *extraConfig[0].MaxTokens
		}
		if extraConfig[0].Temperature != nil {
			temp = *extraConfig[0].Temperature
		}
	}
	k.memoryAI = ai.NewKarmaAI(llm, provider,
		ai.WithSystemMessage(memoryLLMSystemPrompt),
		ai.WithMaxTokens(maxTokens),
		ai.WithTemperature(temp))
}

func (k *KarmaMemory) UseEmbeddingLLM(llm ai.BaseModel, provider ai.Provider) {
	k.embeddingAI = ai.NewKarmaAI(llm, provider)
}

func (k *KarmaMemory) UseService(service VectorServices) error {
	return k.memorydb.switchService(k.userID, k.scope, service)
}

func (k *KarmaMemory) UseLogger(logger *zap.Logger) {
	k.logger = logger
}

func (k *KarmaMemory) UseRetrievalMode(mode RetrievalMode) {
	k.retrievalMode = mode
}

func (k *KarmaMemory) EnableCache(cfg CacheConfig) {
	k.cache = NewCache(k.logger, cfg)
	k.cacheEnabled = k.cache.IsEnabled()
	k.logger.Info("karma_memory: cache enabled", zap.String("backend", string(k.cache.GetBackend())))
}

func (k *KarmaMemory) EnableRedisCache(cfg ...CacheConfig) {
	k.cache = NewRedisCache(k.logger, cfg...)
	k.cacheEnabled = k.cache.IsEnabled()
}

func (k *KarmaMemory) EnableMemoryCache(cfg ...CacheConfig) {
	k.cache = NewMemoryCache(k.logger, cfg...)
	k.cacheEnabled = k.cache.IsEnabled()
}

func (k *KarmaMemory) DisableCache() {
	k.cacheEnabled = false
	k.logger.Info("karma_memory: cache disabled")
}

func (k *KarmaMemory) IsCacheEnabled() bool {
	return k.cacheEnabled && k.cache != nil && k.cache.IsEnabled()
}

func (k *KarmaMemory) GetCacheMethod() CacheMethod {
	if k.cache == nil {
		return ""
	}
	return k.cache.GetBackend()
}

func (k *KarmaMemory) GetCacheStats() (map[string]any, error) {
	if !k.IsCacheEnabled() {
		return map[string]any{"enabled": false}, nil
	}
	return k.cache.GetCacheStats(context.Background(), k.userID)
}

func (k *KarmaMemory) WarmupCache() error {
	if !k.IsCacheEnabled() {
		return nil
	}
	return k.cache.WarmupCache(context.Background(), k.userID, k.scope, k.memorydb.client)
}

func (k *KarmaMemory) InvalidateCache() error {
	if !k.IsCacheEnabled() {
		return nil
	}
	return k.cache.InvalidateUserCache(context.Background(), k.userID)
}

func (k *KarmaMemory) GetHistory() models.AIChatHistory {
	return k.messagesHistory
}

func (k *KarmaMemory) ClearHistory() {
	k.messagesHistory = models.AIChatHistory{
		Messages: make([]models.AIMessage, 0),
	}
}

func (k *KarmaMemory) NumberOfMessages() int {
	return len(k.messagesHistory.Messages)
}

// Advanced implementations require custom logic to manage message history, in such cases below function can be used to update the message history
func (k *KarmaMemory) UpdateMessageHistory(messages []models.AIMessage) {
	k.messagesHistory.Messages = messages

	if len(messages) < 2 {
		return
	}

	lastUserMsg := ""
	lastAIMsg := ""

	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == models.User && lastUserMsg == "" {
			lastUserMsg = messages[i].Message
		}
		if messages[i].Role == models.Assistant && lastAIMsg == "" {
			lastAIMsg = messages[i].Message
		}
		if lastUserMsg != "" && lastAIMsg != "" {
			break
		}
	}

	if k.currentMemoryContext == "" {
		k.GetContext(lastUserMsg)
	}

	if lastUserMsg != "" && lastAIMsg != "" {
		go func() {
			if err := k.ingest(struct {
				UserMessage          string
				AIResponse           string
				CurrentMemoryContext string
			}{
				UserMessage:          lastUserMsg,
				AIResponse:           lastAIMsg,
				CurrentMemoryContext: k.currentMemoryContext,
			}); err != nil {
				k.logger.Error("karma_memory: memory ingestion failed",
					zap.String("userID", k.userID),
					zap.String("scope", k.scope),
					zap.Error(err))
			} else {
				k.logger.Info("karma_memory: memory ingested",
					zap.String("userID", k.userID),
					zap.String("scope", k.scope))
			}
		}()
	}
}

func (k *KarmaMemory) GetContext(userPrompt string) (string, error) {
	mode := k.retrievalMode
	ctx := context.Background()

	var maxTokens int
	var topK int
	var searchQuery filters
	var sq string
	var err error

	switch mode {
	case RetrievalModeConscious:
		maxTokens = 400
		topK = 3

		searchQuery, err = k.generateSearchQuery(userPrompt)
		sq = searchQuery.SearchQuery
		if err != nil {
			k.logger.Warn("karma_memory: search query generation failed, using original prompt", zap.Error(err))
			sq = userPrompt
			searchQuery.SearchQuery = sq
		}

	case RetrievalModeAuto:
		maxTokens = 800
		topK = 5

		sq = userPrompt
		activeStatus := StatusActive
		searchQuery = filters{
			SearchQuery: userPrompt,
			Category:    ptrStr("fact, preference, rule, entity, context, skill, episodic"),
			Lifespan:    ptrStr("short_term, lifelong, long_term, mid_term"),
			Status:      &activeStatus,
		}

	default:
		maxTokens = 300
		topK = 5

		sq = userPrompt
		activeStatus := StatusActive
		searchQuery = filters{
			SearchQuery: userPrompt,
			Status:      &activeStatus,
		}
	}

	var allMemories []Memory

	if mode == RetrievalModeConscious {
		allMemories = k.getContextConscious(ctx, sq, searchQuery, topK)
	} else {
		allMemories = k.getContextAuto(ctx, sq, searchQuery, topK)
	}

	formattedContext := k.formatContext(allMemories, maxTokens)
	k.currentMemoryContext = k.formatContextForIngest(allMemories)

	k.logger.Info("karma_memory: context retrieved",
		zap.String("mode", string(mode)),
		zap.Int("total_memories", len(allMemories)),
		zap.Int("context_length", len(formattedContext)),
		zap.Bool("cache_enabled", k.IsCacheEnabled()))

	return formattedContext, nil
}

func (k *KarmaMemory) getContextConscious(ctx context.Context, sq string, searchQuery filters, topK int) []Memory {
	var wg sync.WaitGroup
	var mu sync.Mutex

	var cachedRules, cachedFacts, cachedSkills, cachedContext []Memory
	var rulesFound, factsFound, skillsFound, contextFound bool
	var dynamicMemories []Memory
	var dynamicFound bool

	var vectorRules []Memory
	var vectorFacts []Memory
	var vectorRelevant []Memory
	var needVectorRules, needVectorFacts, needVectorRelevant bool

	if k.IsCacheEnabled() {
		wg.Add(5)

		go func() {
			defer wg.Done()
			cachedRules, rulesFound = k.cache.GetCachedRules(ctx, k.userID, k.scope)
		}()

		go func() {
			defer wg.Done()
			cachedFacts, factsFound = k.cache.GetCachedFacts(ctx, k.userID, k.scope)
		}()

		go func() {
			defer wg.Done()
			cachedSkills, skillsFound = k.cache.GetCachedSkills(ctx, k.userID, k.scope)
		}()

		go func() {
			defer wg.Done()
			cachedContext, contextFound = k.cache.GetCachedContext(ctx, k.userID, k.scope)
		}()

		go func() {
			defer wg.Done()
			dynamicFilter := k.buildMemoryFilter(searchQuery)
			dynamicMemories, dynamicFound = k.cache.GetCachedMemoriesWithFilter(ctx, k.userID, k.scope, dynamicFilter)
		}()

		wg.Wait()

		needVectorRules = !rulesFound || len(cachedRules) == 0
		needVectorFacts = !factsFound || len(cachedFacts) == 0
		needVectorRelevant = !dynamicFound || len(dynamicMemories) == 0

		if needVectorRules || needVectorFacts || needVectorRelevant {
			var vectorWg sync.WaitGroup

			if needVectorRules {
				vectorWg.Add(1)
				go func() {
					defer vectorWg.Done()
					rules, err := k.memorydb.client.queryVectorByMetadata(filters{
						Category: ptrStr("rule"),
					})
					if err != nil {
						k.logger.Warn("karma_memory: rules query failed", zap.Error(err))
						return
					}
					mu.Lock()
					vectorRules = make([]Memory, 0, len(rules))
					for _, r := range rules {
						vectorRules = append(vectorRules, metadataToMemory(r, ""))
					}
					mu.Unlock()
				}()
			}

			if needVectorFacts {
				vectorWg.Add(1)
				go func() {
					defer vectorWg.Done()
					facts, err := k.memorydb.client.queryVectorByMetadata(filters{
						Category: ptrStr("fact"),
					})
					if err != nil {
						k.logger.Warn("karma_memory: facts query failed", zap.Error(err))
						return
					}
					mu.Lock()
					vectorFacts = make([]Memory, 0, len(facts))
					for _, f := range facts {
						vectorFacts = append(vectorFacts, metadataToMemory(f, ""))
					}
					mu.Unlock()
				}()
			}

			if needVectorRelevant {
				vectorWg.Add(1)
				go func() {
					defer vectorWg.Done()
					relevant, err := k.queryVectorService(sq, topK, searchQuery)
					if err != nil {
						k.logger.Warn("karma_memory: relevant memories query failed", zap.Error(err))
						return
					}
					mu.Lock()
					vectorRelevant = relevant
					mu.Unlock()
				}()
			}

			vectorWg.Wait()
		}

		go func() {
			cacheCtx := context.Background()
			if needVectorRules && len(vectorRules) > 0 {
				k.cache.CacheRules(cacheCtx, k.userID, k.scope, vectorRules)
				k.logger.Debug("karma_memory: cached rules from vector", zap.Int("count", len(vectorRules)))
			}
			if needVectorFacts && len(vectorFacts) > 0 {
				k.cache.CacheFacts(cacheCtx, k.userID, k.scope, vectorFacts)
				k.logger.Debug("karma_memory: cached facts from vector", zap.Int("count", len(vectorFacts)))
			}
			if needVectorRelevant && len(vectorRelevant) > 0 {
				k.cacheMemoriesInBackground(cacheCtx, vectorRelevant)
			}
		}()

		allMemories := make([]Memory, 0)

		if rulesFound && len(cachedRules) > 0 {
			allMemories = append(allMemories, cachedRules...)
		} else if len(vectorRules) > 0 {
			allMemories = append(allMemories, vectorRules...)
		}

		if factsFound && len(cachedFacts) > 0 {
			allMemories = append(allMemories, cachedFacts...)
		} else if len(vectorFacts) > 0 {
			allMemories = append(allMemories, vectorFacts...)
		}

		if skillsFound && len(cachedSkills) > 0 {
			allMemories = append(allMemories, cachedSkills...)
		}
		if contextFound && len(cachedContext) > 0 {
			allMemories = append(allMemories, cachedContext...)
		}

		if dynamicFound && len(dynamicMemories) > 0 {
			allMemories = k.mergeMemories(allMemories, dynamicMemories)
		} else if len(vectorRelevant) > 0 {
			allMemories = k.mergeMemories(allMemories, vectorRelevant)
		}

		return k.deduplicateMemories(allMemories)

	} else {
		wg.Add(3)

		go func() {
			defer wg.Done()
			rules, err := k.memorydb.client.queryVectorByMetadata(filters{
				Category: ptrStr("rule"),
			})
			if err != nil {
				k.logger.Warn("karma_memory: rules query failed", zap.Error(err))
				return
			}
			mu.Lock()
			vectorRules = make([]Memory, 0, len(rules))
			for _, r := range rules {
				vectorRules = append(vectorRules, metadataToMemory(r, ""))
			}
			mu.Unlock()
		}()

		go func() {
			defer wg.Done()
			facts, err := k.memorydb.client.queryVectorByMetadata(filters{
				Category: ptrStr("fact"),
			})
			if err != nil {
				k.logger.Warn("karma_memory: facts query failed", zap.Error(err))
				return
			}
			mu.Lock()
			vectorFacts = make([]Memory, 0, len(facts))
			for _, f := range facts {
				vectorFacts = append(vectorFacts, metadataToMemory(f, ""))
			}
			mu.Unlock()
		}()

		go func() {
			defer wg.Done()
			relevant, err := k.queryVectorService(sq, topK, searchQuery)
			if err != nil {
				k.logger.Warn("karma_memory: relevant memories query failed", zap.Error(err))
				return
			}
			mu.Lock()
			vectorRelevant = relevant
			mu.Unlock()
		}()

		wg.Wait()

		allMemories := make([]Memory, 0)
		allMemories = append(allMemories, vectorRules...)
		allMemories = append(allMemories, vectorFacts...)
		allMemories = k.mergeMemories(allMemories, vectorRelevant)

		return k.deduplicateMemories(allMemories)
	}
}

func (k *KarmaMemory) getContextAuto(ctx context.Context, sq string, searchQuery filters, topK int) []Memory {
	var wg sync.WaitGroup
	var mu sync.Mutex

	var cachedRules, cachedFacts, cachedSkills, cachedContext []Memory
	var rulesFound, factsFound, skillsFound, contextFound bool

	var vectorRules []Memory
	var vectorFacts []Memory
	var vectorRelevant []Memory
	var needVectorRules, needVectorFacts bool

	if k.IsCacheEnabled() {
		wg.Add(4)

		go func() {
			defer wg.Done()
			cachedRules, rulesFound = k.cache.GetCachedRules(ctx, k.userID, k.scope)
		}()

		go func() {
			defer wg.Done()
			cachedFacts, factsFound = k.cache.GetCachedFacts(ctx, k.userID, k.scope)
		}()

		go func() {
			defer wg.Done()
			cachedSkills, skillsFound = k.cache.GetCachedSkills(ctx, k.userID, k.scope)
		}()

		go func() {
			defer wg.Done()
			cachedContext, contextFound = k.cache.GetCachedContext(ctx, k.userID, k.scope)
		}()

		wg.Wait()

		needVectorRules = !rulesFound || len(cachedRules) == 0
		needVectorFacts = !factsFound || len(cachedFacts) == 0

		var vectorWg sync.WaitGroup

		vectorWg.Add(1)
		go func() {
			defer vectorWg.Done()
			relevant, err := k.queryVectorService(sq, topK, searchQuery)
			if err != nil {
				k.logger.Warn("karma_memory: relevant memories query failed", zap.Error(err))
				return
			}
			mu.Lock()
			vectorRelevant = relevant
			mu.Unlock()
		}()

		if needVectorRules {
			vectorWg.Add(1)
			go func() {
				defer vectorWg.Done()
				rules, err := k.memorydb.client.queryVectorByMetadata(filters{
					Category: ptrStr("rule"),
				})
				if err != nil {
					k.logger.Warn("karma_memory: rules query failed", zap.Error(err))
					return
				}
				mu.Lock()
				vectorRules = make([]Memory, 0, len(rules))
				for _, r := range rules {
					vectorRules = append(vectorRules, metadataToMemory(r, ""))
				}
				mu.Unlock()
			}()
		}

		if needVectorFacts {
			vectorWg.Add(1)
			go func() {
				defer vectorWg.Done()
				facts, err := k.memorydb.client.queryVectorByMetadata(filters{
					Category: ptrStr("fact"),
				})
				if err != nil {
					k.logger.Warn("karma_memory: facts query failed", zap.Error(err))
					return
				}
				mu.Lock()
				vectorFacts = make([]Memory, 0, len(facts))
				for _, f := range facts {
					vectorFacts = append(vectorFacts, metadataToMemory(f, ""))
				}
				mu.Unlock()
			}()
		}

		vectorWg.Wait()

		go func() {
			cacheCtx := context.Background()
			if needVectorRules && len(vectorRules) > 0 {
				k.cache.CacheRules(cacheCtx, k.userID, k.scope, vectorRules)
				k.logger.Debug("karma_memory: cached rules from vector", zap.Int("count", len(vectorRules)))
			}
			if needVectorFacts && len(vectorFacts) > 0 {
				k.cache.CacheFacts(cacheCtx, k.userID, k.scope, vectorFacts)
				k.logger.Debug("karma_memory: cached facts from vector", zap.Int("count", len(vectorFacts)))
			}
			if len(vectorRelevant) > 0 {
				k.cacheMemoriesInBackground(cacheCtx, vectorRelevant)
			}
		}()

		allMemories := make([]Memory, 0)

		if rulesFound && len(cachedRules) > 0 {
			allMemories = append(allMemories, cachedRules...)
		} else if len(vectorRules) > 0 {
			allMemories = append(allMemories, vectorRules...)
		}

		if factsFound && len(cachedFacts) > 0 {
			allMemories = append(allMemories, cachedFacts...)
		} else if len(vectorFacts) > 0 {
			allMemories = append(allMemories, vectorFacts...)
		}

		if skillsFound && len(cachedSkills) > 0 {
			allMemories = append(allMemories, cachedSkills...)
		}
		if contextFound && len(cachedContext) > 0 {
			allMemories = append(allMemories, cachedContext...)
		}

		allMemories = k.mergeMemories(allMemories, vectorRelevant)

		return k.deduplicateMemories(allMemories)

	} else {
		wg.Add(3)

		go func() {
			defer wg.Done()
			rules, err := k.memorydb.client.queryVectorByMetadata(filters{
				Category: ptrStr("rule"),
			})
			if err != nil {
				k.logger.Warn("karma_memory: rules query failed", zap.Error(err))
				return
			}
			mu.Lock()
			vectorRules = make([]Memory, 0, len(rules))
			for _, r := range rules {
				vectorRules = append(vectorRules, metadataToMemory(r, ""))
			}
			mu.Unlock()
		}()

		go func() {
			defer wg.Done()
			facts, err := k.memorydb.client.queryVectorByMetadata(filters{
				Category: ptrStr("fact"),
			})
			if err != nil {
				k.logger.Warn("karma_memory: facts query failed", zap.Error(err))
				return
			}
			mu.Lock()
			vectorFacts = make([]Memory, 0, len(facts))
			for _, f := range facts {
				vectorFacts = append(vectorFacts, metadataToMemory(f, ""))
			}
			mu.Unlock()
		}()

		go func() {
			defer wg.Done()
			relevant, err := k.queryVectorService(sq, topK, searchQuery)
			if err != nil {
				k.logger.Warn("karma_memory: relevant memories query failed", zap.Error(err))
				return
			}
			mu.Lock()
			vectorRelevant = relevant
			mu.Unlock()
		}()

		wg.Wait()

		allMemories := make([]Memory, 0)
		allMemories = append(allMemories, vectorRules...)
		allMemories = append(allMemories, vectorFacts...)
		allMemories = k.mergeMemories(allMemories, vectorRelevant)

		return k.deduplicateMemories(allMemories)
	}
}

func (k *KarmaMemory) queryVectorService(sq string, topK int, searchQuery filters) ([]Memory, error) {
	var vectorResults []vector.VectorScore
	var err error

	switch k.memorydb.currentService {
	case VectorServicePinecone:
		vectorResults, err = k.memorydb.client.queryVector(nil, topK, searchQuery)
	case VectorServiceUpstash:
		embeddings, embErr := k.getEmbeddings(sq)
		if embErr != nil {
			return nil, embErr
		}
		vectorResults, err = k.memorydb.client.queryVector(embeddings, topK, searchQuery)
	default:
		embeddings, embErr := k.getEmbeddings(sq)
		if embErr != nil {
			return nil, embErr
		}
		vectorResults, err = k.memorydb.client.queryVector(embeddings, topK, searchQuery)
	}

	if err != nil {
		return nil, err
	}

	memories := make([]Memory, 0, len(vectorResults))
	for _, result := range vectorResults {
		if result.Metadata != nil {
			mem := metadataToMemory(result.Metadata, result.Id)
			memories = append(memories, mem)
		}
	}

	return memories, nil
}

func (k *KarmaMemory) cacheMemoriesInBackground(ctx context.Context, memories []Memory) {
	categoryMap := make(map[MemoryCategory][]Memory)
	for _, mem := range memories {
		switch mem.Category {
		case CategoryRule, CategoryFact, CategorySkill, CategoryContext:
			categoryMap[mem.Category] = append(categoryMap[mem.Category], mem)
		}
	}

	for category, mems := range categoryMap {
		if len(mems) > 0 {
			k.cache.CacheMemoriesByCategory(ctx, k.userID, k.scope, category, mems)
		}
	}

	k.cache.CacheAllMemories(ctx, k.userID, k.scope, memories)
}

func (k *KarmaMemory) deduplicateMemories(memories []Memory) []Memory {
	seen := make(map[string]bool)
	result := make([]Memory, 0, len(memories))

	for _, mem := range memories {
		key := mem.Id
		if key == "" {
			key = normalizeSummary(mem.Summary)
		}
		if !seen[key] {
			seen[key] = true
			result = append(result, mem)
		}
	}

	return result
}

func (k *KarmaMemory) buildMemoryFilter(f filters) MemoryFilter {
	filter := MemoryFilter{
		NotExpired: true,
	}

	if f.Category != nil && *f.Category != "" {
		filter.Categories = parseCategories(*f.Category)
	}

	if f.Lifespan != nil && *f.Lifespan != "" {
		filter.Lifespans = parseLifespans(*f.Lifespan)
	}

	if f.Status != nil {
		filter.Status = f.Status
	} else {
		activeStatus := StatusActive
		filter.Status = &activeStatus
	}

	if f.Importance != nil {
		filter.MinImportance = f.Importance
	}

	return filter
}

func (k *KarmaMemory) mergeMemories(cached, vector []Memory) []Memory {
	seen := make(map[string]bool)
	result := make([]Memory, 0, len(cached)+len(vector))

	for _, mem := range cached {
		if mem.Id != "" && !seen[mem.Id] {
			seen[mem.Id] = true
			result = append(result, mem)
		} else if mem.Id == "" {
			normalizedSummary := normalizeSummary(mem.Summary)
			if !seen[normalizedSummary] {
				seen[normalizedSummary] = true
				result = append(result, mem)
			}
		}
	}

	for _, mem := range vector {
		if mem.Id != "" && !seen[mem.Id] {
			seen[mem.Id] = true
			result = append(result, mem)
		} else if mem.Id == "" {
			normalizedSummary := normalizeSummary(mem.Summary)
			if !seen[normalizedSummary] {
				seen[normalizedSummary] = true
				result = append(result, mem)
			}
		}
	}

	return result
}
