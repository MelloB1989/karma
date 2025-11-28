package memory

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/MelloB1989/karma/utils"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

type CacheMethod string

const (
	CacheMethodMemory CacheMethod = "memory"
	CacheMethodRedis  CacheMethod = "redis"
)

const (
	rulesCachePrefix       = "karma:memory:rules:"
	factsCachePrefix       = "karma:memory:facts:"
	skillsCachePrefix      = "karma:memory:skills:"
	contextCachePrefix     = "karma:memory:context:"
	allMemoriesCachePrefix = "karma:memory:all:"

	defaultRulesTTL       = 30 * time.Minute
	defaultFactsTTL       = 20 * time.Minute
	defaultSkillsTTL      = 25 * time.Minute
	defaultContextTTL     = 10 * time.Minute
	defaultAllMemoriesTTL = 15 * time.Minute
)

type CacheConfig struct {
	Backend          CacheMethod
	RulesTTL         time.Duration
	FactsTTL         time.Duration
	SkillsTTL        time.Duration
	ContextTTL       time.Duration
	AllMemoriesTTL   time.Duration
	LocalCacheMaxAge time.Duration
	Enabled          bool
}

// MemoryFilter represents dynamic filter criteria for conscious mode retrieval
type MemoryFilter struct {
	Categories    []MemoryCategory // Filter by categories (e.g., fact, rule, skill)
	Lifespans     []MemoryLifespan // Filter by lifespans (e.g., short_term, lifelong)
	Status        *MemoryStatus    // Filter by status (e.g., active)
	MinImportance *int             // Filter by minimum importance
	NotExpired    bool             // Only include non-expired memories
}

type cacheEntry struct {
	data      any
	expiresAt time.Time
}

type CachedMemories struct {
	Memories  []Memory  `json:"memories"`
	CachedAt  time.Time `json:"cached_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

type MemoryCache interface {
	IsEnabled() bool
	GetBackend() CacheMethod
	CacheMemoriesByCategory(ctx context.Context, userID, scope string, category MemoryCategory, memories []Memory) error
	GetCachedMemoriesByCategory(ctx context.Context, userID, scope string, category MemoryCategory) ([]Memory, bool)
	CacheRules(ctx context.Context, userID, scope string, rules []Memory) error
	GetCachedRules(ctx context.Context, userID, scope string) ([]Memory, bool)
	CacheFacts(ctx context.Context, userID, scope string, facts []Memory) error
	GetCachedFacts(ctx context.Context, userID, scope string) ([]Memory, bool)
	CacheSkills(ctx context.Context, userID, scope string, skills []Memory) error
	GetCachedSkills(ctx context.Context, userID, scope string) ([]Memory, bool)
	CacheContext(ctx context.Context, userID, scope string, contextMemories []Memory) error
	GetCachedContext(ctx context.Context, userID, scope string) ([]Memory, bool)
	// Dynamic filtering for conscious mode
	CacheAllMemories(ctx context.Context, userID, scope string, memories []Memory) error
	GetCachedMemoriesWithFilter(ctx context.Context, userID, scope string, filter MemoryFilter) ([]Memory, bool)
	InvalidateCategoryCache(ctx context.Context, userID, scope string, category MemoryCategory) error
	InvalidateUserCache(ctx context.Context, userID string) error
	GetCacheStats(ctx context.Context, userID string) (map[string]any, error)
	WarmupCache(ctx context.Context, userID, scope string, vectorClient vectorService) error
	Close() error
}

type inMemoryCache struct {
	logger         *zap.Logger
	enabled        bool
	rulesTTL       time.Duration
	factsTTL       time.Duration
	skillsTTL      time.Duration
	contextTTL     time.Duration
	allMemoriesTTL time.Duration
	mu             sync.RWMutex
	cache          map[string]*cacheEntry
	stopCleanup    chan struct{}
}

type redisCache struct {
	client         *redis.Client
	logger         *zap.Logger
	enabled        bool
	rulesTTL       time.Duration
	factsTTL       time.Duration
	skillsTTL      time.Duration
	contextTTL     time.Duration
	allMemoriesTTL time.Duration
	mu             sync.RWMutex
	localCache     map[string]*cacheEntry
	localTTL       time.Duration
	stopCleanup    chan struct{}
}

func NewCache(logger *zap.Logger, cfg ...CacheConfig) MemoryCache {
	config := CacheConfig{
		Backend:          CacheMethodMemory,
		RulesTTL:         defaultRulesTTL,
		FactsTTL:         defaultFactsTTL,
		SkillsTTL:        defaultSkillsTTL,
		ContextTTL:       defaultContextTTL,
		LocalCacheMaxAge: 5 * time.Minute,
		Enabled:          true,
	}

	if len(cfg) > 0 {
		c := cfg[0]
		config.Backend = c.Backend
		config.Enabled = c.Enabled
		if c.RulesTTL > 0 {
			config.RulesTTL = c.RulesTTL
		}
		if c.FactsTTL > 0 {
			config.FactsTTL = c.FactsTTL
		}
		if c.SkillsTTL > 0 {
			config.SkillsTTL = c.SkillsTTL
		}
		if c.ContextTTL > 0 {
			config.ContextTTL = c.ContextTTL
		}
		if c.LocalCacheMaxAge > 0 {
			config.LocalCacheMaxAge = c.LocalCacheMaxAge
		}
	}

	if config.Backend == CacheMethodRedis {
		return newRedisCache(logger, config)
	}
	return newInMemoryCache(logger, config)
}

func NewRedisCache(logger *zap.Logger, cfg ...CacheConfig) MemoryCache {
	config := CacheConfig{
		Backend:          CacheMethodRedis,
		RulesTTL:         defaultRulesTTL,
		FactsTTL:         defaultFactsTTL,
		SkillsTTL:        defaultSkillsTTL,
		ContextTTL:       defaultContextTTL,
		LocalCacheMaxAge: 5 * time.Minute,
		Enabled:          true,
	}

	if len(cfg) > 0 {
		c := cfg[0]
		config.Enabled = c.Enabled
		if c.RulesTTL > 0 {
			config.RulesTTL = c.RulesTTL
		}
		if c.FactsTTL > 0 {
			config.FactsTTL = c.FactsTTL
		}
		if c.SkillsTTL > 0 {
			config.SkillsTTL = c.SkillsTTL
		}
		if c.ContextTTL > 0 {
			config.ContextTTL = c.ContextTTL
		}
		if c.LocalCacheMaxAge > 0 {
			config.LocalCacheMaxAge = c.LocalCacheMaxAge
		}
	}

	return newRedisCache(logger, config)
}

func NewMemoryCache(logger *zap.Logger, cfg ...CacheConfig) MemoryCache {
	config := CacheConfig{
		Backend:    CacheMethodMemory,
		RulesTTL:   defaultRulesTTL,
		FactsTTL:   defaultFactsTTL,
		SkillsTTL:  defaultSkillsTTL,
		ContextTTL: defaultContextTTL,
		Enabled:    true,
	}

	if len(cfg) > 0 {
		c := cfg[0]
		config.Enabled = c.Enabled
		if c.RulesTTL > 0 {
			config.RulesTTL = c.RulesTTL
		}
		if c.FactsTTL > 0 {
			config.FactsTTL = c.FactsTTL
		}
		if c.SkillsTTL > 0 {
			config.SkillsTTL = c.SkillsTTL
		}
		if c.ContextTTL > 0 {
			config.ContextTTL = c.ContextTTL
		}
	}

	return newInMemoryCache(logger, config)
}

func newInMemoryCache(logger *zap.Logger, config CacheConfig) *inMemoryCache {
	allMemTTL := config.AllMemoriesTTL
	if allMemTTL == 0 {
		allMemTTL = defaultAllMemoriesTTL
	}

	c := &inMemoryCache{
		logger:         logger,
		enabled:        config.Enabled,
		rulesTTL:       config.RulesTTL,
		factsTTL:       config.FactsTTL,
		skillsTTL:      config.SkillsTTL,
		contextTTL:     config.ContextTTL,
		allMemoriesTTL: allMemTTL,
		cache:          make(map[string]*cacheEntry),
		stopCleanup:    make(chan struct{}),
	}

	if c.enabled {
		go c.cleanup()
	}

	return c
}

func newRedisCache(logger *zap.Logger, config CacheConfig) *redisCache {
	var client *redis.Client
	defer func() {
		if r := recover(); r != nil {
			logger.Warn("karma_memory: failed to connect to Redis, cache disabled")
			client = nil
		}
	}()

	client = utils.RedisConnect()

	allMemTTL := config.AllMemoriesTTL
	if allMemTTL == 0 {
		allMemTTL = defaultAllMemoriesTTL
	}

	c := &redisCache{
		client:         client,
		logger:         logger,
		enabled:        config.Enabled && client != nil,
		rulesTTL:       config.RulesTTL,
		factsTTL:       config.FactsTTL,
		skillsTTL:      config.SkillsTTL,
		contextTTL:     config.ContextTTL,
		allMemoriesTTL: allMemTTL,
		localCache:     make(map[string]*cacheEntry),
		localTTL:       config.LocalCacheMaxAge,
		stopCleanup:    make(chan struct{}),
	}

	if c.enabled {
		go c.cleanup()
		logger.Info("karma_memory: Redis cache enabled")
	}

	return c
}

func (c *inMemoryCache) cleanup() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.mu.Lock()
			now := time.Now()
			for key, entry := range c.cache {
				if now.After(entry.expiresAt) {
					delete(c.cache, key)
				}
			}
			c.mu.Unlock()
		case <-c.stopCleanup:
			return
		}
	}
}

func (c *redisCache) cleanup() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.mu.Lock()
			now := time.Now()
			for key, entry := range c.localCache {
				if now.After(entry.expiresAt) {
					delete(c.localCache, key)
				}
			}
			c.mu.Unlock()
		case <-c.stopCleanup:
			return
		}
	}
}

func generateCacheKey(prefix, userID, scope string, category MemoryCategory) string {
	return fmt.Sprintf("%s%s:%s:%s", prefix, userID, scope, string(category))
}

func getCategoryPrefix(category MemoryCategory) string {
	switch category {
	case CategoryRule:
		return rulesCachePrefix
	case CategoryFact:
		return factsCachePrefix
	case CategorySkill:
		return skillsCachePrefix
	case CategoryContext:
		return contextCachePrefix
	default:
		return factsCachePrefix
	}
}

func (c *inMemoryCache) getTTLForCategory(category MemoryCategory) time.Duration {
	switch category {
	case CategoryRule:
		return c.rulesTTL
	case CategoryFact:
		return c.factsTTL
	case CategorySkill:
		return c.skillsTTL
	case CategoryContext:
		return c.contextTTL
	default:
		return c.factsTTL
	}
}

func (c *redisCache) getTTLForCategory(category MemoryCategory) time.Duration {
	switch category {
	case CategoryRule:
		return c.rulesTTL
	case CategoryFact:
		return c.factsTTL
	case CategorySkill:
		return c.skillsTTL
	case CategoryContext:
		return c.contextTTL
	default:
		return c.factsTTL
	}
}

func filterExpiredMemories(memories []Memory) []Memory {
	now := time.Now()
	result := make([]Memory, 0, len(memories))
	for _, mem := range memories {
		if mem.ExpiresAt == nil || now.Before(*mem.ExpiresAt) {
			if mem.Status == StatusActive || mem.Status == "" {
				result = append(result, mem)
			}
		}
	}
	return result
}

// filterMemoriesWithCriteria applies dynamic filter criteria to memories
func filterMemoriesWithCriteria(memories []Memory, filter MemoryFilter) []Memory {
	now := time.Now()
	result := make([]Memory, 0, len(memories))

	for _, mem := range memories {
		// Check expiry if required
		if filter.NotExpired {
			if mem.ExpiresAt != nil && now.After(*mem.ExpiresAt) {
				continue
			}
		}

		// Check status
		if filter.Status != nil && mem.Status != *filter.Status {
			continue
		}

		// Check categories (if specified)
		if len(filter.Categories) > 0 {
			categoryMatch := false
			for _, cat := range filter.Categories {
				if mem.Category == cat {
					categoryMatch = true
					break
				}
			}
			if !categoryMatch {
				continue
			}
		}

		// Check lifespans (if specified)
		if len(filter.Lifespans) > 0 {
			lifespanMatch := false
			for _, ls := range filter.Lifespans {
				if mem.Lifespan == ls {
					lifespanMatch = true
					break
				}
			}
			if !lifespanMatch {
				continue
			}
		}

		// Check minimum importance
		if filter.MinImportance != nil && mem.Importance < *filter.MinImportance {
			continue
		}

		result = append(result, mem)
	}

	return result
}

// parseCategories parses a comma-separated category string into a slice
func parseCategories(categoryStr string) []MemoryCategory {
	if categoryStr == "" {
		return nil
	}
	parts := strings.Split(categoryStr, ",")
	categories := make([]MemoryCategory, 0, len(parts))
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			categories = append(categories, MemoryCategory(trimmed))
		}
	}
	return categories
}

// parseLifespans parses a comma-separated lifespan string into a slice
func parseLifespans(lifespanStr string) []MemoryLifespan {
	if lifespanStr == "" {
		return nil
	}
	parts := strings.Split(lifespanStr, ",")
	lifespans := make([]MemoryLifespan, 0, len(parts))
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			lifespans = append(lifespans, MemoryLifespan(trimmed))
		}
	}
	return lifespans
}

func (c *inMemoryCache) IsEnabled() bool {
	return c.enabled
}

func (c *inMemoryCache) GetBackend() CacheMethod {
	return CacheMethodMemory
}

func (c *inMemoryCache) get(key string) (any, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, exists := c.cache[key]
	if !exists || time.Now().After(entry.expiresAt) {
		return nil, false
	}
	return entry.data, true
}

func (c *inMemoryCache) set(key string, data any, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.cache[key] = &cacheEntry{
		data:      data,
		expiresAt: time.Now().Add(ttl),
	}
}

func (c *inMemoryCache) delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.cache, key)
}

func (c *inMemoryCache) deleteByPrefix(prefix string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	for key := range c.cache {
		if len(key) >= len(prefix) && key[:len(prefix)] == prefix {
			delete(c.cache, key)
		}
	}
}

func (c *inMemoryCache) CacheMemoriesByCategory(ctx context.Context, userID, scope string, category MemoryCategory, memories []Memory) error {
	if !c.enabled {
		return nil
	}

	ttl := c.getTTLForCategory(category)
	prefix := getCategoryPrefix(category)
	key := generateCacheKey(prefix, userID, scope, category)

	cached := CachedMemories{
		Memories:  filterExpiredMemories(memories),
		CachedAt:  time.Now(),
		ExpiresAt: time.Now().Add(ttl),
	}

	c.set(key, cached, ttl)
	c.logger.Debug("karma_memory: cached memories by category",
		zap.String("category", string(category)),
		zap.Int("count", len(cached.Memories)))
	return nil
}

func (c *inMemoryCache) GetCachedMemoriesByCategory(ctx context.Context, userID, scope string, category MemoryCategory) ([]Memory, bool) {
	if !c.enabled {
		return nil, false
	}

	prefix := getCategoryPrefix(category)
	key := generateCacheKey(prefix, userID, scope, category)

	data, found := c.get(key)
	if !found {
		return nil, false
	}

	cached, ok := data.(CachedMemories)
	if !ok {
		return nil, false
	}

	filtered := filterExpiredMemories(cached.Memories)
	c.logger.Debug("karma_memory: cache hit for category",
		zap.String("category", string(category)),
		zap.Int("count", len(filtered)))
	return filtered, true
}

func (c *inMemoryCache) CacheRules(ctx context.Context, userID, scope string, rules []Memory) error {
	return c.CacheMemoriesByCategory(ctx, userID, scope, CategoryRule, rules)
}

func (c *inMemoryCache) GetCachedRules(ctx context.Context, userID, scope string) ([]Memory, bool) {
	return c.GetCachedMemoriesByCategory(ctx, userID, scope, CategoryRule)
}

func (c *inMemoryCache) CacheFacts(ctx context.Context, userID, scope string, facts []Memory) error {
	return c.CacheMemoriesByCategory(ctx, userID, scope, CategoryFact, facts)
}

func (c *inMemoryCache) GetCachedFacts(ctx context.Context, userID, scope string) ([]Memory, bool) {
	return c.GetCachedMemoriesByCategory(ctx, userID, scope, CategoryFact)
}

func (c *inMemoryCache) CacheSkills(ctx context.Context, userID, scope string, skills []Memory) error {
	return c.CacheMemoriesByCategory(ctx, userID, scope, CategorySkill, skills)
}

func (c *inMemoryCache) GetCachedSkills(ctx context.Context, userID, scope string) ([]Memory, bool) {
	return c.GetCachedMemoriesByCategory(ctx, userID, scope, CategorySkill)
}

func (c *inMemoryCache) CacheContext(ctx context.Context, userID, scope string, contextMemories []Memory) error {
	return c.CacheMemoriesByCategory(ctx, userID, scope, CategoryContext, contextMemories)
}

func (c *inMemoryCache) GetCachedContext(ctx context.Context, userID, scope string) ([]Memory, bool) {
	return c.GetCachedMemoriesByCategory(ctx, userID, scope, CategoryContext)
}

// CacheAllMemories caches all memories for a user/scope for dynamic filtering
func (c *inMemoryCache) CacheAllMemories(ctx context.Context, userID, scope string, memories []Memory) error {
	if !c.enabled {
		return nil
	}

	key := fmt.Sprintf("%s%s:%s", allMemoriesCachePrefix, userID, scope)

	cached := CachedMemories{
		Memories:  memories, // Store all, filter on retrieval
		CachedAt:  time.Now(),
		ExpiresAt: time.Now().Add(c.allMemoriesTTL),
	}

	c.set(key, cached, c.allMemoriesTTL)
	c.logger.Debug("karma_memory: cached all memories for dynamic filtering",
		zap.Int("count", len(memories)))
	return nil
}

// GetCachedMemoriesWithFilter retrieves memories from cache and applies dynamic filters
func (c *inMemoryCache) GetCachedMemoriesWithFilter(ctx context.Context, userID, scope string, filter MemoryFilter) ([]Memory, bool) {
	if !c.enabled {
		return nil, false
	}

	key := fmt.Sprintf("%s%s:%s", allMemoriesCachePrefix, userID, scope)

	data, found := c.get(key)
	if !found {
		return nil, false
	}

	cached, ok := data.(CachedMemories)
	if !ok {
		return nil, false
	}

	// Apply dynamic filters
	filtered := filterMemoriesWithCriteria(cached.Memories, filter)

	c.logger.Debug("karma_memory: cache hit with dynamic filter",
		zap.Int("total_cached", len(cached.Memories)),
		zap.Int("after_filter", len(filtered)),
		zap.Int("categories", len(filter.Categories)),
		zap.Int("lifespans", len(filter.Lifespans)))

	return filtered, len(filtered) > 0
}

func (c *inMemoryCache) InvalidateCategoryCache(ctx context.Context, userID, scope string, category MemoryCategory) error {
	if !c.enabled {
		return nil
	}

	prefix := getCategoryPrefix(category)
	key := generateCacheKey(prefix, userID, scope, category)
	c.delete(key)
	c.logger.Debug("karma_memory: invalidated category cache",
		zap.String("category", string(category)))
	return nil
}

func (c *inMemoryCache) InvalidateUserCache(ctx context.Context, userID string) error {
	if !c.enabled {
		return nil
	}

	prefixes := []string{
		fmt.Sprintf("%s%s:", rulesCachePrefix, userID),
		fmt.Sprintf("%s%s:", factsCachePrefix, userID),
		fmt.Sprintf("%s%s:", skillsCachePrefix, userID),
		fmt.Sprintf("%s%s:", contextCachePrefix, userID),
	}

	for _, prefix := range prefixes {
		c.deleteByPrefix(prefix)
	}
	c.logger.Debug("karma_memory: invalidated all cache for user", zap.String("userID", userID))
	return nil
}

func (c *inMemoryCache) GetCacheStats(ctx context.Context, userID string) (map[string]any, error) {
	if !c.enabled {
		return map[string]any{"enabled": false, "backend": "memory"}, nil
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	stats := map[string]any{
		"enabled":       true,
		"backend":       "memory",
		"total_entries": len(c.cache),
	}

	rulesCount, factsCount, skillsCount, contextCount := 0, 0, 0, 0
	userPrefix := userID + ":"

	for key := range c.cache {
		if len(key) > len(rulesCachePrefix) && key[:len(rulesCachePrefix)] == rulesCachePrefix {
			if len(key) > len(rulesCachePrefix)+len(userPrefix) && key[len(rulesCachePrefix):len(rulesCachePrefix)+len(userPrefix)] == userPrefix {
				rulesCount++
			}
		} else if len(key) > len(factsCachePrefix) && key[:len(factsCachePrefix)] == factsCachePrefix {
			if len(key) > len(factsCachePrefix)+len(userPrefix) && key[len(factsCachePrefix):len(factsCachePrefix)+len(userPrefix)] == userPrefix {
				factsCount++
			}
		} else if len(key) > len(skillsCachePrefix) && key[:len(skillsCachePrefix)] == skillsCachePrefix {
			if len(key) > len(skillsCachePrefix)+len(userPrefix) && key[len(skillsCachePrefix):len(skillsCachePrefix)+len(userPrefix)] == userPrefix {
				skillsCount++
			}
		} else if len(key) > len(contextCachePrefix) && key[:len(contextCachePrefix)] == contextCachePrefix {
			if len(key) > len(contextCachePrefix)+len(userPrefix) && key[len(contextCachePrefix):len(contextCachePrefix)+len(userPrefix)] == userPrefix {
				contextCount++
			}
		}
	}

	stats["rules_count"] = rulesCount
	stats["facts_count"] = factsCount
	stats["skills_count"] = skillsCount
	stats["context_count"] = contextCount

	return stats, nil
}

func (c *inMemoryCache) WarmupCache(ctx context.Context, userID, scope string, vectorClient vectorService) error {
	if !c.enabled {
		return nil
	}

	c.logger.Info("karma_memory: starting cache warmup", zap.String("userID", userID), zap.String("scope", scope))

	categories := []MemoryCategory{CategoryRule, CategoryFact, CategorySkill, CategoryContext}

	for _, category := range categories {
		categoryStr := string(category)
		memories, err := vectorClient.queryVectorByMetadata(filters{Category: &categoryStr})
		if err != nil {
			c.logger.Warn("karma_memory: failed to fetch memories for warmup",
				zap.String("category", categoryStr),
				zap.Error(err))
			continue
		}

		if len(memories) > 0 {
			memoryList := make([]Memory, 0, len(memories))
			for _, m := range memories {
				memoryList = append(memoryList, metadataToMemory(m, ""))
			}
			c.CacheMemoriesByCategory(ctx, userID, scope, category, memoryList)
		}
	}

	c.logger.Info("karma_memory: cache warmup completed")
	return nil
}

func (c *inMemoryCache) Close() error {
	close(c.stopCleanup)
	return nil
}

func (c *redisCache) IsEnabled() bool {
	return c.enabled && c.client != nil
}

func (c *redisCache) GetBackend() CacheMethod {
	return CacheMethodRedis
}

func (c *redisCache) getLocal(key string) (any, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, exists := c.localCache[key]
	if !exists || time.Now().After(entry.expiresAt) {
		return nil, false
	}
	return entry.data, true
}

func (c *redisCache) setLocal(key string, data any, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	actualTTL := c.localTTL
	if ttl < actualTTL {
		actualTTL = ttl
	}

	c.localCache[key] = &cacheEntry{
		data:      data,
		expiresAt: time.Now().Add(actualTTL),
	}
}

func (c *redisCache) deleteLocal(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.localCache, key)
}

func (c *redisCache) deleteLocalByPrefix(prefix string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	for key := range c.localCache {
		if len(key) >= len(prefix) && key[:len(prefix)] == prefix {
			delete(c.localCache, key)
		}
	}
}

func (c *redisCache) CacheMemoriesByCategory(ctx context.Context, userID, scope string, category MemoryCategory, memories []Memory) error {
	if !c.IsEnabled() {
		return nil
	}

	ttl := c.getTTLForCategory(category)
	prefix := getCategoryPrefix(category)
	key := generateCacheKey(prefix, userID, scope, category)

	cached := CachedMemories{
		Memories:  filterExpiredMemories(memories),
		CachedAt:  time.Now(),
		ExpiresAt: time.Now().Add(ttl),
	}

	data, err := json.Marshal(cached)
	if err != nil {
		c.logger.Error("karma_memory: failed to marshal memories for cache", zap.Error(err))
		return err
	}

	if err := c.client.Set(ctx, key, data, ttl).Err(); err != nil {
		c.logger.Error("karma_memory: failed to cache memories in Redis", zap.Error(err))
		return err
	}

	c.setLocal(key, cached, ttl)
	c.logger.Debug("karma_memory: cached memories by category",
		zap.String("category", string(category)),
		zap.Int("count", len(cached.Memories)))
	return nil
}

func (c *redisCache) GetCachedMemoriesByCategory(ctx context.Context, userID, scope string, category MemoryCategory) ([]Memory, bool) {
	if !c.IsEnabled() {
		return nil, false
	}

	prefix := getCategoryPrefix(category)
	key := generateCacheKey(prefix, userID, scope, category)

	if data, found := c.getLocal(key); found {
		if cached, ok := data.(CachedMemories); ok {
			filtered := filterExpiredMemories(cached.Memories)
			c.logger.Debug("karma_memory: local cache hit for category",
				zap.String("category", string(category)),
				zap.Int("count", len(filtered)))
			return filtered, true
		}
	}

	data, err := c.client.Get(ctx, key).Bytes()
	if err != nil {
		return nil, false
	}

	var cached CachedMemories
	if err := json.Unmarshal(data, &cached); err != nil {
		c.logger.Warn("karma_memory: failed to unmarshal cached memories", zap.Error(err))
		return nil, false
	}

	if time.Now().After(cached.ExpiresAt) {
		c.client.Del(ctx, key)
		return nil, false
	}

	filtered := filterExpiredMemories(cached.Memories)
	c.setLocal(key, cached, time.Until(cached.ExpiresAt))
	c.logger.Debug("karma_memory: Redis cache hit for category",
		zap.String("category", string(category)),
		zap.Int("count", len(filtered)))
	return filtered, true
}

func (c *redisCache) CacheRules(ctx context.Context, userID, scope string, rules []Memory) error {
	return c.CacheMemoriesByCategory(ctx, userID, scope, CategoryRule, rules)
}

func (c *redisCache) GetCachedRules(ctx context.Context, userID, scope string) ([]Memory, bool) {
	return c.GetCachedMemoriesByCategory(ctx, userID, scope, CategoryRule)
}

func (c *redisCache) CacheFacts(ctx context.Context, userID, scope string, facts []Memory) error {
	return c.CacheMemoriesByCategory(ctx, userID, scope, CategoryFact, facts)
}

func (c *redisCache) GetCachedFacts(ctx context.Context, userID, scope string) ([]Memory, bool) {
	return c.GetCachedMemoriesByCategory(ctx, userID, scope, CategoryFact)
}

func (c *redisCache) CacheSkills(ctx context.Context, userID, scope string, skills []Memory) error {
	return c.CacheMemoriesByCategory(ctx, userID, scope, CategorySkill, skills)
}

func (c *redisCache) GetCachedSkills(ctx context.Context, userID, scope string) ([]Memory, bool) {
	return c.GetCachedMemoriesByCategory(ctx, userID, scope, CategorySkill)
}

func (c *redisCache) CacheContext(ctx context.Context, userID, scope string, contextMemories []Memory) error {
	return c.CacheMemoriesByCategory(ctx, userID, scope, CategoryContext, contextMemories)
}

func (c *redisCache) GetCachedContext(ctx context.Context, userID, scope string) ([]Memory, bool) {
	return c.GetCachedMemoriesByCategory(ctx, userID, scope, CategoryContext)
}

// CacheAllMemories caches all memories for a user/scope for dynamic filtering
func (c *redisCache) CacheAllMemories(ctx context.Context, userID, scope string, memories []Memory) error {
	if !c.IsEnabled() {
		return nil
	}

	key := fmt.Sprintf("%s%s:%s", allMemoriesCachePrefix, userID, scope)

	cached := CachedMemories{
		Memories:  memories,
		CachedAt:  time.Now(),
		ExpiresAt: time.Now().Add(c.allMemoriesTTL),
	}

	data, err := json.Marshal(cached)
	if err != nil {
		c.logger.Warn("karma_memory: failed to marshal all memories for cache", zap.Error(err))
		return err
	}

	if err := c.client.Set(ctx, key, data, c.allMemoriesTTL).Err(); err != nil {
		c.logger.Warn("karma_memory: failed to cache all memories in Redis", zap.Error(err))
		return err
	}

	// Also cache locally for faster subsequent reads
	c.setLocal(key, cached, c.localTTL)

	c.logger.Debug("karma_memory: cached all memories for dynamic filtering",
		zap.Int("count", len(memories)))
	return nil
}

// GetCachedMemoriesWithFilter retrieves memories from cache and applies dynamic filters
func (c *redisCache) GetCachedMemoriesWithFilter(ctx context.Context, userID, scope string, filter MemoryFilter) ([]Memory, bool) {
	if !c.IsEnabled() {
		return nil, false
	}

	key := fmt.Sprintf("%s%s:%s", allMemoriesCachePrefix, userID, scope)

	// Try local cache first
	if data, found := c.getLocal(key); found {
		if cached, ok := data.(CachedMemories); ok {
			filtered := filterMemoriesWithCriteria(cached.Memories, filter)
			c.logger.Debug("karma_memory: local cache hit with dynamic filter",
				zap.Int("total_cached", len(cached.Memories)),
				zap.Int("after_filter", len(filtered)))
			return filtered, len(filtered) > 0
		}
	}

	// Try Redis
	data, err := c.client.Get(ctx, key).Bytes()
	if err != nil {
		if err != redis.Nil {
			c.logger.Warn("karma_memory: failed to get all memories from Redis", zap.Error(err))
		}
		return nil, false
	}

	var cached CachedMemories
	if err := json.Unmarshal(data, &cached); err != nil {
		c.logger.Warn("karma_memory: failed to unmarshal all memories from Redis", zap.Error(err))
		return nil, false
	}

	// Cache locally for faster subsequent reads
	c.setLocal(key, cached, c.localTTL)

	// Apply dynamic filters
	filtered := filterMemoriesWithCriteria(cached.Memories, filter)

	c.logger.Debug("karma_memory: Redis cache hit with dynamic filter",
		zap.Int("total_cached", len(cached.Memories)),
		zap.Int("after_filter", len(filtered)),
		zap.Int("categories", len(filter.Categories)),
		zap.Int("lifespans", len(filter.Lifespans)))

	return filtered, len(filtered) > 0
}

func (c *redisCache) InvalidateCategoryCache(ctx context.Context, userID, scope string, category MemoryCategory) error {
	if !c.IsEnabled() {
		return nil
	}

	prefix := getCategoryPrefix(category)
	key := generateCacheKey(prefix, userID, scope, category)
	c.deleteLocal(key)

	if err := c.client.Del(ctx, key).Err(); err != nil {
		c.logger.Warn("karma_memory: failed to invalidate category cache in Redis",
			zap.String("category", string(category)),
			zap.Error(err))
		return err
	}

	c.logger.Debug("karma_memory: invalidated category cache", zap.String("category", string(category)))
	return nil
}

func (c *redisCache) InvalidateUserCache(ctx context.Context, userID string) error {
	if !c.IsEnabled() {
		return nil
	}

	patterns := []string{
		fmt.Sprintf("%s%s:*", rulesCachePrefix, userID),
		fmt.Sprintf("%s%s:*", factsCachePrefix, userID),
		fmt.Sprintf("%s%s:*", skillsCachePrefix, userID),
		fmt.Sprintf("%s%s:*", contextCachePrefix, userID),
	}

	prefixes := []string{
		fmt.Sprintf("%s%s:", rulesCachePrefix, userID),
		fmt.Sprintf("%s%s:", factsCachePrefix, userID),
		fmt.Sprintf("%s%s:", skillsCachePrefix, userID),
		fmt.Sprintf("%s%s:", contextCachePrefix, userID),
	}

	for _, prefix := range prefixes {
		c.deleteLocalByPrefix(prefix)
	}

	for _, pattern := range patterns {
		if err := c.deleteByPattern(ctx, pattern); err != nil {
			c.logger.Warn("karma_memory: failed to delete cache pattern", zap.String("pattern", pattern), zap.Error(err))
		}
	}

	c.logger.Debug("karma_memory: invalidated all cache for user", zap.String("userID", userID))
	return nil
}

func (c *redisCache) deleteByPattern(ctx context.Context, pattern string) error {
	var cursor uint64
	for {
		keys, nextCursor, err := c.client.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			return err
		}

		if len(keys) > 0 {
			c.client.Del(ctx, keys...)
		}

		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}
	return nil
}

func (c *redisCache) GetCacheStats(ctx context.Context, userID string) (map[string]any, error) {
	if !c.IsEnabled() {
		return map[string]any{"enabled": false, "backend": "redis"}, nil
	}

	stats := map[string]any{
		"enabled": true,
		"backend": "redis",
	}

	patterns := map[string]string{
		"rules":   fmt.Sprintf("%s%s:*", rulesCachePrefix, userID),
		"facts":   fmt.Sprintf("%s%s:*", factsCachePrefix, userID),
		"skills":  fmt.Sprintf("%s%s:*", skillsCachePrefix, userID),
		"context": fmt.Sprintf("%s%s:*", contextCachePrefix, userID),
	}

	for name, pattern := range patterns {
		count := 0
		var cursor uint64
		for {
			keys, nextCursor, err := c.client.Scan(ctx, cursor, pattern, 100).Result()
			if err != nil {
				break
			}
			count += len(keys)
			cursor = nextCursor
			if cursor == 0 {
				break
			}
		}
		stats[name+"_count"] = count
	}

	c.mu.RLock()
	stats["local_cache_entries"] = len(c.localCache)
	c.mu.RUnlock()

	return stats, nil
}

func (c *redisCache) WarmupCache(ctx context.Context, userID, scope string, vectorClient vectorService) error {
	if !c.IsEnabled() {
		return nil
	}

	c.logger.Info("karma_memory: starting cache warmup", zap.String("userID", userID), zap.String("scope", scope))

	categories := []MemoryCategory{CategoryRule, CategoryFact, CategorySkill, CategoryContext}

	for _, category := range categories {
		categoryStr := string(category)
		memories, err := vectorClient.queryVectorByMetadata(filters{Category: &categoryStr})
		if err != nil {
			c.logger.Warn("karma_memory: failed to fetch memories for warmup",
				zap.String("category", categoryStr),
				zap.Error(err))
			continue
		}

		if len(memories) > 0 {
			memoryList := make([]Memory, 0, len(memories))
			for _, m := range memories {
				memoryList = append(memoryList, metadataToMemory(m, ""))
			}
			c.CacheMemoriesByCategory(ctx, userID, scope, category, memoryList)
		}
	}

	c.logger.Info("karma_memory: cache warmup completed")
	return nil
}

func (c *redisCache) Close() error {
	close(c.stopCleanup)
	if c.client != nil {
		return c.client.Close()
	}
	return nil
}
