package memory

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/MelloB1989/karma/ai"
	"github.com/MelloB1989/karma/config"
	"github.com/MelloB1989/karma/models"
	"go.uber.org/zap"
)

const (
	service             = VectorServiceUpstash
	memory_llm          = ai.Llama31_8B
	memory_llm_provider = ai.Groq
	memory_max_tokens   = 2048
)

func setupTestMemory(t *testing.T) *KarmaMemory {
	t.Helper()

	apiKey := config.GetEnvRaw("OPENAI_KEY")
	if apiKey == "" {
		t.Skip("OPENAI_KEY not set")
	}

	kai := ai.NewKarmaAI(ai.Llama4_Guard_12B, ai.Groq)
	mem := NewKarmaMemory(kai, "test_user_123", "test_scope")
	mem.UseMemoryLLM(memory_llm, memory_llm_provider, MemoryLlmConfig{MaxTokens: intPtr(memory_max_tokens)})
	mem.UseService(service)

	return mem
}

func setupTestMemoryWithRedisCache(t *testing.T) *KarmaMemory {
	t.Helper()

	apiKey := config.GetEnvRaw("OPENAI_KEY")
	if apiKey == "" {
		t.Skip("OPENAI_KEY not set")
	}

	redisURL := config.DefaultConfig().RedisURL
	if redisURL == "" {
		t.Skip("REDIS_URL not set")
	}

	kai := ai.NewKarmaAI(ai.Llama33_70B, ai.Groq)
	mem := NewKarmaMemory(kai, "test_user_cache", "test_scope")
	mem.EnableRedisCache(CacheConfig{
		Backend:    CacheMethodRedis,
		RulesTTL:   15 * time.Minute,
		FactsTTL:   10 * time.Minute,
		SkillsTTL:  10 * time.Minute,
		ContextTTL: 5 * time.Minute,
		Enabled:    true,
	})
	mem.UseRetrievalMode(RetrievalModeAuto)
	mem.UseMemoryLLM(memory_llm, memory_llm_provider, MemoryLlmConfig{MaxTokens: intPtr(memory_max_tokens)})
	mem.UseService(service)

	return mem
}

func setupTestMemoryWithMemoryCache(t *testing.T) *KarmaMemory {
	t.Helper()

	apiKey := config.GetEnvRaw("OPENAI_KEY")
	if apiKey == "" {
		t.Skip("OPENAI_KEY not set")
	}

	kai := ai.NewKarmaAI(ai.Llama33_70B, ai.Groq)
	mem := NewKarmaMemory(kai, "test_user_memcache", "test_scope")
	mem.EnableMemoryCache(CacheConfig{
		Backend:    CacheMethodMemory,
		RulesTTL:   15 * time.Minute,
		FactsTTL:   10 * time.Minute,
		SkillsTTL:  10 * time.Minute,
		ContextTTL: 5 * time.Minute,
		Enabled:    true,
	})
	mem.UseRetrievalMode(RetrievalModeAuto)
	mem.UseMemoryLLM(memory_llm, memory_llm_provider, MemoryLlmConfig{MaxTokens: intPtr(memory_max_tokens)})
	mem.UseService(service)

	return mem
}

func TestNewKarmaMemory(t *testing.T) {
	mem := setupTestMemory(t)

	if mem.userID != "test_user_123" {
		t.Errorf("Expected userID to be test_user_123, got %s", mem.userID)
	}

	if mem.scope != "test_scope" {
		t.Errorf("Expected scope to be test_scope, got %s", mem.scope)
	}

	if mem.retrievalMode != RetrievalModeAuto {
		t.Errorf("Expected default retrieval mode to be Auto, got %s", mem.retrievalMode)
	}

	if mem.logger == nil {
		t.Error("Logger should be initialized")
	}
}

func TestUpdateMessageHistory(t *testing.T) {
	mem := setupTestMemory(t)

	messages := []models.AIMessage{
		{
			Role:      models.User,
			Message:   "I use PostgreSQL for my databases",
			Timestamp: time.Now(),
			UniqueId:  "msg1",
		},
		{
			Role:      models.Assistant,
			Message:   "That's great! PostgreSQL is a robust choice.",
			Timestamp: time.Now(),
			UniqueId:  "msg2",
		},
	}

	mem.UpdateMessageHistory(messages)

	if len(mem.messagesHistory.Messages) != 2 {
		t.Errorf("Expected 2 messages in history, got %d", len(mem.messagesHistory.Messages))
	}

	time.Sleep(40000 * time.Millisecond)
}

func TestSetRetrievalMode(t *testing.T) {
	mem := setupTestMemory(t)

	mem.UseRetrievalMode(RetrievalModeConscious)
	if mem.retrievalMode != RetrievalModeConscious {
		t.Errorf("Expected retrieval mode to be Conscious, got %s", mem.retrievalMode)
	}

	mem.UseRetrievalMode(RetrievalModeAuto)
	if mem.retrievalMode != RetrievalModeAuto {
		t.Errorf("Expected retrieval mode to be Auto, got %s", mem.retrievalMode)
	}
}

func TestUseScope(t *testing.T) {
	mem := setupTestMemory(t)

	success := mem.UseScope("new_scope")
	if !success {
		t.Error("UseScope should return true")
	}

	if mem.scope != "new_scope" {
		t.Errorf("Expected scope to be new_scope, got %s", mem.scope)
	}
}

// Redis Cache Tests

func TestRedisCacheEnabled(t *testing.T) {
	mem := setupTestMemoryWithRedisCache(t)

	if !mem.IsCacheEnabled() {
		t.Error("Cache should be enabled")
	}

	if mem.GetCacheMethod() != CacheMethodRedis {
		t.Errorf("Expected Redis backend, got %s", mem.GetCacheMethod())
	}

	stats, err := mem.GetCacheStats()
	if err != nil {
		t.Errorf("GetCacheStats failed: %v", err)
	}

	if enabled, ok := stats["enabled"].(bool); !ok || !enabled {
		t.Error("Cache stats should show enabled=true")
	}
}

func TestMemoryCacheEnabled(t *testing.T) {
	mem := setupTestMemoryWithMemoryCache(t)

	if !mem.IsCacheEnabled() {
		t.Error("Cache should be enabled")
	}

	if mem.GetCacheMethod() != CacheMethodMemory {
		t.Errorf("Expected Memory backend, got %s", mem.GetCacheMethod())
	}

	stats, err := mem.GetCacheStats()
	if err != nil {
		t.Errorf("GetCacheStats failed: %v", err)
	}

	if enabled, ok := stats["enabled"].(bool); !ok || !enabled {
		t.Error("Cache stats should show enabled=true")
	}

	if backend, ok := stats["backend"].(string); !ok || backend != "memory" {
		t.Error("Cache stats should show backend=memory")
	}
}

func TestCacheDisable(t *testing.T) {
	mem := setupTestMemoryWithMemoryCache(t)

	if !mem.IsCacheEnabled() {
		t.Skip("Cache not enabled, skipping disable test")
	}

	mem.DisableCache()

	if mem.IsCacheEnabled() {
		t.Error("Cache should be disabled after DisableCache()")
	}
}

func TestRedisCacheInvalidation(t *testing.T) {
	mem := setupTestMemoryWithRedisCache(t)

	if !mem.IsCacheEnabled() {
		t.Skip("Cache not enabled")
	}

	err := mem.InvalidateCache()
	if err != nil {
		t.Errorf("InvalidateCache failed: %v", err)
	}
}

func TestMemoryCacheInvalidation(t *testing.T) {
	mem := setupTestMemoryWithMemoryCache(t)

	if !mem.IsCacheEnabled() {
		t.Skip("Cache not enabled")
	}

	err := mem.InvalidateCache()
	if err != nil {
		t.Errorf("InvalidateCache failed: %v", err)
	}
}

func TestRecallLatencyWithRedisCache(t *testing.T) {
	mem := setupTestMemoryWithRedisCache(t)

	if !mem.IsCacheEnabled() {
		t.Skip("Cache not enabled")
	}

	messages := []models.AIMessage{
		{Role: models.User, Message: "My favorite database is PostgreSQL and I use Redis for caching", Timestamp: time.Now(), UniqueId: "1"},
		{Role: models.Assistant, Message: "Great choices for performance!", Timestamp: time.Now(), UniqueId: "2"},
	}
	mem.UpdateMessageHistory(messages)
	time.Sleep(2 * time.Second)

	start1 := time.Now()
	_, err := mem.GetContext("What database do I use?")
	coldLatency := time.Since(start1)
	if err != nil {
		t.Fatalf("First GetContext failed: %v", err)
	}

	start2 := time.Now()
	_, err = mem.GetContext("What database do I use?")
	warmLatency := time.Since(start2)
	if err != nil {
		t.Fatalf("Second GetContext failed: %v", err)
	}

	t.Logf("Redis cache - Cold latency: %v, Warm latency: %v", coldLatency, warmLatency)
}

func TestRecallLatencyWithMemoryCache(t *testing.T) {
	mem := setupTestMemoryWithMemoryCache(t)

	if !mem.IsCacheEnabled() {
		t.Skip("Cache not enabled")
	}

	messages := []models.AIMessage{
		{Role: models.User, Message: "My favorite language is Go and I use PostgreSQL", Timestamp: time.Now(), UniqueId: "1"},
		{Role: models.Assistant, Message: "Great tech stack!", Timestamp: time.Now(), UniqueId: "2"},
	}
	mem.UpdateMessageHistory(messages)
	time.Sleep(2 * time.Second)

	start1 := time.Now()
	_, err := mem.GetContext("What language do I prefer?")
	coldLatency := time.Since(start1)
	if err != nil {
		t.Fatalf("First GetContext failed: %v", err)
	}

	start2 := time.Now()
	_, err = mem.GetContext("What language do I prefer?")
	warmLatency := time.Since(start2)
	if err != nil {
		t.Fatalf("Second GetContext failed: %v", err)
	}

	t.Logf("Memory cache - Cold latency: %v, Warm latency: %v", coldLatency, warmLatency)
}

func TestCacheWarmup(t *testing.T) {
	mem := setupTestMemoryWithMemoryCache(t)

	if !mem.IsCacheEnabled() {
		t.Skip("Cache not enabled")
	}

	err := mem.WarmupCache()
	if err != nil {
		t.Errorf("WarmupCache failed: %v", err)
	}

	stats, err := mem.GetCacheStats()
	if err != nil {
		t.Errorf("GetCacheStats failed: %v", err)
	}

	t.Logf("Cache stats after warmup: %+v", stats)
}

func TestNewKarmaMemoryWithOptions(t *testing.T) {
	apiKey := config.GetEnvRaw("OPENAI_KEY")
	if apiKey == "" {
		t.Skip("OPENAI_KEY not set")
	}

	kai := ai.NewKarmaAI(ai.Llama33_70B, ai.Groq)
	mem := NewKarmaMemory(kai, "test_user_options", "test_scope")
	mem.EnableMemoryCache(CacheConfig{Enabled: true})
	mem.UseRetrievalMode(RetrievalModeConscious)

	if mem.retrievalMode != RetrievalModeConscious {
		t.Errorf("Expected retrieval mode Conscious, got %s", mem.retrievalMode)
	}

	if mem.userID != "test_user_options" {
		t.Errorf("Expected userID test_user_options, got %s", mem.userID)
	}

	if !mem.IsCacheEnabled() {
		t.Error("Cache should be enabled")
	}

	if mem.GetCacheMethod() != CacheMethodMemory {
		t.Errorf("Expected Memory backend, got %s", mem.GetCacheMethod())
	}
}

// Benchmark comparing cached vs non-cached recall

func BenchmarkGetContextWithoutCache(b *testing.B) {
	apiKey := config.GetEnvRaw("OPENAI_KEY")
	if apiKey == "" {
		b.Skip("OPENAI_KEY not set")
	}

	kai := ai.NewKarmaAI(ai.Llama33_70B, ai.Groq)
	mem := NewKarmaMemory(kai, "bench_user_nocache", "bench_scope")
	mem.UseMemoryLLM(memory_llm, memory_llm_provider, MemoryLlmConfig{MaxTokens: intPtr(memory_max_tokens)})
	mem.UseService(service)

	// Seed memories
	messages := []models.AIMessage{
		{Role: models.User, Message: "I work with Go and PostgreSQL daily", Timestamp: time.Now(), UniqueId: "1"},
		{Role: models.Assistant, Message: "Great tech stack!", Timestamp: time.Now(), UniqueId: "2"},
	}
	mem.UpdateMessageHistory(messages)
	time.Sleep(2 * time.Second)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = mem.GetContext("What technologies do I use?")
	}
}

func BenchmarkGetContextWithRedisCache(b *testing.B) {
	apiKey := config.GetEnvRaw("OPENAI_KEY")
	if apiKey == "" {
		b.Skip("OPENAI_KEY not set")
	}

	redisURL := config.DefaultConfig().RedisURL
	if redisURL == "" {
		b.Skip("REDIS_URL not set")
	}

	kai := ai.NewKarmaAI(ai.Llama33_70B, ai.Groq)
	mem := NewKarmaMemory(kai, "bench_user_cache", "bench_scope")
	mem.EnableRedisCache(CacheConfig{
		Backend:    CacheMethodRedis,
		RulesTTL:   15 * time.Minute,
		FactsTTL:   10 * time.Minute,
		SkillsTTL:  10 * time.Minute,
		ContextTTL: 5 * time.Minute,
		Enabled:    true,
	})
	mem.UseMemoryLLM(memory_llm, memory_llm_provider, MemoryLlmConfig{MaxTokens: intPtr(memory_max_tokens)})
	mem.UseService(service)

	messages := []models.AIMessage{
		{Role: models.User, Message: "I work with Go and PostgreSQL daily", Timestamp: time.Now(), UniqueId: "1"},
		{Role: models.Assistant, Message: "Great tech stack!", Timestamp: time.Now(), UniqueId: "2"},
	}
	mem.UpdateMessageHistory(messages)
	time.Sleep(2 * time.Second)

	_, _ = mem.GetContext("What technologies do I use?")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = mem.GetContext("What technologies do I use?")
	}
}

func BenchmarkGetContextWithMemoryCache(b *testing.B) {
	apiKey := config.GetEnvRaw("OPENAI_KEY")
	if apiKey == "" {
		b.Skip("OPENAI_KEY not set")
	}

	kai := ai.NewKarmaAI(ai.Llama33_70B, ai.Groq)
	mem := NewKarmaMemory(kai, "bench_user_memcache", "bench_scope")
	mem.EnableMemoryCache(CacheConfig{
		Backend:    CacheMethodMemory,
		RulesTTL:   15 * time.Minute,
		FactsTTL:   10 * time.Minute,
		SkillsTTL:  10 * time.Minute,
		ContextTTL: 5 * time.Minute,
		Enabled:    true,
	})
	mem.UseMemoryLLM(memory_llm, memory_llm_provider, MemoryLlmConfig{MaxTokens: intPtr(memory_max_tokens)})
	mem.UseService(service)

	messages := []models.AIMessage{
		{Role: models.User, Message: "I work with Rust and MongoDB daily", Timestamp: time.Now(), UniqueId: "1"},
		{Role: models.Assistant, Message: "Interesting choices!", Timestamp: time.Now(), UniqueId: "2"},
	}
	mem.UpdateMessageHistory(messages)
	time.Sleep(2 * time.Second)

	_, _ = mem.GetContext("What technologies do I use?")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = mem.GetContext("What technologies do I use?")
	}
}

func BenchmarkMemoryCacheOperations(b *testing.B) {
	logger, _ := zap.NewProduction()
	cache := NewMemoryCache(logger, CacheConfig{
		Backend:    CacheMethodMemory,
		RulesTTL:   15 * time.Minute,
		FactsTTL:   10 * time.Minute,
		SkillsTTL:  10 * time.Minute,
		ContextTTL: 5 * time.Minute,
		Enabled:    true,
	})

	ctx := context.Background()

	b.Run("CacheRules", func(b *testing.B) {
		rules := []Memory{
			{Id: "rule1", Summary: "Always be helpful", Category: CategoryRule, Status: StatusActive},
			{Id: "rule2", Summary: "Never share personal info", Category: CategoryRule, Status: StatusActive},
		}
		for i := 0; i < b.N; i++ {
			cache.CacheRules(ctx, "bench_user", "bench_scope", rules)
		}
	})

	b.Run("GetCachedRules", func(b *testing.B) {
		rules := []Memory{
			{Id: "rule1", Summary: "Always be helpful", Category: CategoryRule, Status: StatusActive},
		}
		cache.CacheRules(ctx, "bench_user", "bench_scope", rules)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			cache.GetCachedRules(ctx, "bench_user", "bench_scope")
		}
	})

	b.Run("CacheFacts", func(b *testing.B) {
		facts := []Memory{
			{Id: "fact1", Summary: "User likes Go", Category: CategoryFact, Status: StatusActive},
			{Id: "fact2", Summary: "User uses PostgreSQL", Category: CategoryFact, Status: StatusActive},
		}
		for i := 0; i < b.N; i++ {
			cache.CacheFacts(ctx, "bench_user", "bench_scope", facts)
		}
	})

	b.Run("GetCachedFacts", func(b *testing.B) {
		facts := []Memory{
			{Id: "fact1", Summary: "User likes Go", Category: CategoryFact, Status: StatusActive},
		}
		cache.CacheFacts(ctx, "bench_user", "bench_scope", facts)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			cache.GetCachedFacts(ctx, "bench_user", "bench_scope")
		}
	})
}

func BenchmarkRedisCacheOperations(b *testing.B) {
	redisURL := config.DefaultConfig().RedisURL
	if redisURL == "" {
		b.Skip("REDIS_URL not set")
	}

	logger, _ := zap.NewProduction()
	cache := NewRedisCache(logger, CacheConfig{
		Backend:    CacheMethodRedis,
		RulesTTL:   15 * time.Minute,
		FactsTTL:   10 * time.Minute,
		SkillsTTL:  10 * time.Minute,
		ContextTTL: 5 * time.Minute,
		Enabled:    true,
	})

	ctx := context.Background()

	b.Run("CacheRules", func(b *testing.B) {
		rules := []Memory{
			{Id: "rule1", Summary: "Always be helpful", Category: CategoryRule, Status: StatusActive},
			{Id: "rule2", Summary: "Never share personal info", Category: CategoryRule, Status: StatusActive},
		}
		for i := 0; i < b.N; i++ {
			cache.CacheRules(ctx, "bench_user", "bench_scope", rules)
		}
	})

	b.Run("GetCachedRules", func(b *testing.B) {
		rules := []Memory{
			{Id: "rule1", Summary: "Always be helpful", Category: CategoryRule, Status: StatusActive},
		}
		cache.CacheRules(ctx, "bench_user", "bench_scope", rules)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			cache.GetCachedRules(ctx, "bench_user", "bench_scope")
		}
	})

	b.Run("CacheFacts", func(b *testing.B) {
		facts := []Memory{
			{Id: "fact1", Summary: "User likes Go", Category: CategoryFact, Status: StatusActive},
			{Id: "fact2", Summary: "User uses PostgreSQL", Category: CategoryFact, Status: StatusActive},
		}
		for i := 0; i < b.N; i++ {
			cache.CacheFacts(ctx, "bench_user", "bench_scope", facts)
		}
	})

	b.Run("GetCachedFacts", func(b *testing.B) {
		facts := []Memory{
			{Id: "fact1", Summary: "User likes Go", Category: CategoryFact, Status: StatusActive},
		}
		cache.CacheFacts(ctx, "bench_user", "bench_scope", facts)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			cache.GetCachedFacts(ctx, "bench_user", "bench_scope")
		}
	})
}

func TestCacheLatencyComparison(t *testing.T) {
	memWithRedis := setupTestMemoryWithRedisCache(t)
	memWithMemory := setupTestMemoryWithMemoryCache(t)
	memWithoutCache := setupTestMemory(t)

	query := "What programming language do I prefer?"

	var noCacheLatencies []time.Duration
	for i := 0; i < 3; i++ {
		start := time.Now()
		_, _ = memWithoutCache.GetContext(query)
		noCacheLatencies = append(noCacheLatencies, time.Since(start))
	}

	var redisLatencies []time.Duration
	if memWithRedis.IsCacheEnabled() {
		_, _ = memWithRedis.GetContext(query)
		for i := 0; i < 3; i++ {
			start := time.Now()
			_, _ = memWithRedis.GetContext(query)
			redisLatencies = append(redisLatencies, time.Since(start))
		}
	}

	var memoryLatencies []time.Duration
	if memWithMemory.IsCacheEnabled() {
		_, _ = memWithMemory.GetContext(query)
		for i := 0; i < 3; i++ {
			start := time.Now()
			_, _ = memWithMemory.GetContext(query)
			memoryLatencies = append(memoryLatencies, time.Since(start))
		}
	}

	var noCacheAvg time.Duration
	for _, l := range noCacheLatencies {
		noCacheAvg += l
	}
	noCacheAvg /= time.Duration(len(noCacheLatencies))

	t.Logf("Average latency without cache: %v", noCacheAvg)

	if len(redisLatencies) > 0 {
		var redisAvg time.Duration
		for _, l := range redisLatencies {
			redisAvg += l
		}
		redisAvg /= time.Duration(len(redisLatencies))
		t.Logf("Average latency with Redis cache: %v (speedup: %.2fx)", redisAvg, float64(noCacheAvg)/float64(redisAvg))
	}

	if len(memoryLatencies) > 0 {
		var memoryAvg time.Duration
		for _, l := range memoryLatencies {
			memoryAvg += l
		}
		memoryAvg /= time.Duration(len(memoryLatencies))
		t.Logf("Average latency with Memory cache: %v (speedup: %.2fx)", memoryAvg, float64(noCacheAvg)/float64(memoryAvg))
	}
}

func TestUseUser(t *testing.T) {
	mem := setupTestMemory(t)

	success := mem.UseUser("new_user_456")
	if !success {
		t.Error("UseUser should return true")
	}

	if mem.userID != "new_user_456" {
		t.Errorf("Expected userID to be new_user_456, got %s", mem.userID)
	}
}

func TestHistoryFunctions(t *testing.T) {
	mem := setupTestMemory(t)

	messages := []models.AIMessage{
		{Role: models.User, Message: "Hello", UniqueId: "1"},
		{Role: models.Assistant, Message: "Hi there", UniqueId: "2"},
	}

	mem.UpdateMessageHistory(messages)

	count := mem.NumberOfMessages()
	if count != 2 {
		t.Errorf("Expected 2 messages, got %d", count)
	}

	history := mem.GetHistory()
	if len(history.Messages) != 2 {
		t.Errorf("Expected 2 messages in history, got %d", len(history.Messages))
	}

	mem.ClearHistory()

	count = mem.NumberOfMessages()
	if count != 0 {
		t.Errorf("Expected 0 messages after clear, got %d", count)
	}
}

func TestChatCompletion(t *testing.T) {
	mem := setupTestMemory(t)

	messages := []models.AIMessage{
		{
			Role:      models.User,
			Message:   "My name is John and I love coding in Go",
			Timestamp: time.Now(),
			UniqueId:  "msg1",
		},
		{
			Role:      models.Assistant,
			Message:   "Nice to meet you John! Go is a great language.",
			Timestamp: time.Now(),
			UniqueId:  "msg2",
		},
	}
	mem.UpdateMessageHistory(messages)

	time.Sleep(500 * time.Millisecond)

	response, err := mem.ChatCompletion("What's my name?")
	if err != nil {
		t.Fatalf("ChatCompletion failed: %v", err)
	}

	if response == nil {
		t.Fatal("Expected non-nil response")
	}

	if response.AIResponse == "" {
		t.Error("Expected non-empty AI response")
	}

	if mem.NumberOfMessages() < 4 {
		t.Errorf("Expected at least 4 messages after completion, got %d", mem.NumberOfMessages())
	}
}

func TestChatCompletionWithContext(t *testing.T) {
	mem := setupTestMemory(t)
	mem.UseRetrievalMode(RetrievalModeConscious)

	messages := []models.AIMessage{
		{
			Role:      models.User,
			Message:   "I prefer using PostgreSQL for databases",
			Timestamp: time.Now(),
			UniqueId:  "msg1",
		},
		{
			Role:      models.Assistant,
			Message:   "PostgreSQL is an excellent choice for relational databases.",
			Timestamp: time.Now(),
			UniqueId:  "msg2",
		},
	}
	mem.UpdateMessageHistory(messages)

	time.Sleep(1 * time.Second)

	response, err := mem.ChatCompletion("What database do I prefer?")
	if err != nil {
		t.Fatalf("ChatCompletion failed: %v", err)
	}

	if response == nil {
		t.Fatal("Expected non-nil response")
	}

	t.Logf("Response: %s", response.AIResponse)
}

func TestGetContext(t *testing.T) {
	mem := setupTestMemory(t)

	messages := []models.AIMessage{
		{
			Role:      models.User,
			Message:   "I use React and TypeScript for frontend development",
			Timestamp: time.Now(),
			UniqueId:  "msg1",
		},
		{
			Role:      models.Assistant,
			Message:   "That's a powerful combination for building modern web applications.",
			Timestamp: time.Now(),
			UniqueId:  "msg2",
		},
	}
	mem.UpdateMessageHistory(messages)

	time.Sleep(2 * time.Second)

	context, err := mem.GetContext("What frameworks do I use?")
	if err != nil {
		t.Logf("GetContext returned error (expected if no memories yet): %v", err)
	}

	t.Logf("Retrieved context: %s", context)
}

func TestRetrievalModes(t *testing.T) {
	mem := setupTestMemory(t)

	messages := []models.AIMessage{
		{
			Role:      models.User,
			Message:   "I love Python and Go for backend development. I also like clean code practices.",
			Timestamp: time.Now(),
			UniqueId:  "msg1",
		},
		{
			Role:      models.Assistant,
			Message:   "Both are excellent choices for backend systems.",
			Timestamp: time.Now(),
			UniqueId:  "msg2",
		},
	}
	mem.UpdateMessageHistory(messages)

	time.Sleep(2 * time.Second)

	mem.UseRetrievalMode(RetrievalModeConscious)
	contextConscious, err := mem.GetContext("What languages do I prefer?")
	if err != nil {
		t.Logf("Conscious mode context error: %v", err)
	}
	t.Logf("Conscious mode context length: %d", len(contextConscious))

	mem.UseRetrievalMode(RetrievalModeAuto)
	contextAuto, err := mem.GetContext("What languages do I prefer?")
	if err != nil {
		t.Logf("Auto mode context error: %v", err)
	}
	t.Logf("Auto mode context length: %d", len(contextAuto))
}

func TestMultipleConversations(t *testing.T) {
	mem := setupTestMemory(t)

	conversations := []struct {
		user      string
		assistant string
	}{
		{"My favorite color is blue", "Blue is a calming color!"},
		{"I work at TechCorp as a senior engineer", "That sounds like an exciting position!"},
		{"Never reply in French, I don't understand it", "Understood, I'll stick to English."},
	}

	allMessages := []models.AIMessage{}
	for i, conv := range conversations {
		allMessages = append(allMessages,
			models.AIMessage{Role: models.User, Message: conv.user, Timestamp: time.Now(), UniqueId: fmt.Sprintf("u%d", i)},
			models.AIMessage{Role: models.Assistant, Message: conv.assistant, Timestamp: time.Now(), UniqueId: fmt.Sprintf("a%d", i)},
		)
		mem.UpdateMessageHistory(allMessages)
		time.Sleep(500 * time.Millisecond)
	}

	if mem.NumberOfMessages() != len(conversations)*2 {
		t.Errorf("Expected %d messages, got %d", len(conversations)*2, mem.NumberOfMessages())
	}
}

func TestMemoryPersistence(t *testing.T) {
	mem1 := setupTestMemory(t)

	messages := []models.AIMessage{
		{Role: models.User, Message: "I'm allergic to peanuts", Timestamp: time.Now(), UniqueId: "1"},
		{Role: models.Assistant, Message: "I'll remember your peanut allergy.", Timestamp: time.Now(), UniqueId: "2"},
	}
	mem1.UpdateMessageHistory(messages)
	time.Sleep(6 * time.Second)

	mem2 := setupTestMemory(t)
	context, err := mem2.GetContext("What are my allergies?")
	if err != nil {
		t.Logf("Context retrieval for persistence test: %v", err)
	}

	t.Logf("Persistence test context: %s", context)
}

func TestUseMemoryLLM(t *testing.T) {
	mem := setupTestMemory(t)

	mem.UseMemoryLLM(ai.GPT4o, ai.OpenAI)

	if mem.memoryAI == nil {
		t.Error("Memory AI should not be nil after UseMemoryLLM")
	}
}

func TestUseEmbeddingLLM(t *testing.T) {
	mem := setupTestMemory(t)

	mem.UseEmbeddingLLM(ai.TextEmbedding3Large, ai.OpenAI)

	if mem.embeddingAI == nil {
		t.Error("Embedding AI should not be nil after UseEmbeddingLLM")
	}
}

func TestEmptyMessageHistory(t *testing.T) {
	mem := setupTestMemory(t)

	mem.UpdateMessageHistory([]models.AIMessage{})

	if mem.NumberOfMessages() != 0 {
		t.Errorf("Expected 0 messages for empty update, got %d", mem.NumberOfMessages())
	}
}

func TestSingleMessage(t *testing.T) {
	mem := setupTestMemory(t)

	messages := []models.AIMessage{
		{Role: models.User, Message: "Hello", Timestamp: time.Now(), UniqueId: "1"},
	}

	mem.UpdateMessageHistory(messages)
	time.Sleep(100 * time.Millisecond)

	if mem.NumberOfMessages() != 1 {
		t.Errorf("Expected 1 message, got %d", mem.NumberOfMessages())
	}
}

// ==================== Benchmark Tests ====================

func BenchmarkGetContext(b *testing.B) {
	apiKey := config.GetEnvRaw("OPENAI_KEY")
	if apiKey == "" {
		b.Skip("OPENAI_KEY not set")
	}

	kai := ai.NewKarmaAI(ai.GPT4oMini, ai.OpenAI)
	mem := NewKarmaMemory(kai, "bench_user", "bench_scope")

	// Seed some memory
	messages := []models.AIMessage{
		{Role: models.User, Message: "I use PostgreSQL and Redis", Timestamp: time.Now(), UniqueId: "1"},
		{Role: models.Assistant, Message: "Great database choices!", Timestamp: time.Now(), UniqueId: "2"},
	}
	mem.UpdateMessageHistory(messages)
	time.Sleep(2 * time.Second)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = mem.GetContext("What databases do I use?")
	}
}

func BenchmarkUpdateMessageHistory(b *testing.B) {
	apiKey := config.GetEnvRaw("OPENAI_KEY")
	if apiKey == "" {
		b.Skip("OPENAI_KEY not set")
	}

	kai := ai.NewKarmaAI(ai.GPT4oMini, ai.OpenAI)
	mem := NewKarmaMemory(kai, "bench_user", "bench_scope")

	messages := []models.AIMessage{
		{Role: models.User, Message: "Test message", Timestamp: time.Now(), UniqueId: "1"},
		{Role: models.Assistant, Message: "Test response", Timestamp: time.Now(), UniqueId: "2"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mem.UpdateMessageHistory(messages)
	}
}

func BenchmarkChatCompletion(b *testing.B) {
	apiKey := config.GetEnvRaw("OPENAI_KEY")
	if apiKey == "" {
		b.Skip("OPENAI_KEY not set")
	}

	kai := ai.NewKarmaAI(ai.GPT4oMini, ai.OpenAI)
	mem := NewKarmaMemory(kai, "bench_user", "bench_scope")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = mem.ChatCompletion("Hello")
		mem.ClearHistory()
	}
}

func BenchmarkHistoryOperations(b *testing.B) {
	apiKey := config.GetEnvRaw("OPENAI_KEY")
	if apiKey == "" {
		b.Skip("OPENAI_KEY not set")
	}

	kai := ai.NewKarmaAI(ai.GPT4oMini, ai.OpenAI)
	mem := NewKarmaMemory(kai, "bench_user", "bench_scope")

	messages := make([]models.AIMessage, 100)
	for i := 0; i < 100; i++ {
		role := models.User
		if i%2 == 1 {
			role = models.Assistant
		}
		messages[i] = models.AIMessage{
			Role:      role,
			Message:   fmt.Sprintf("Message %d", i),
			Timestamp: time.Now(),
			UniqueId:  fmt.Sprintf("msg_%d", i),
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mem.UpdateMessageHistory(messages)
		_ = mem.GetHistory()
		_ = mem.NumberOfMessages()
		mem.ClearHistory()
	}
}

// ==================== Recall Latency Tests ====================

func TestRecallLatency(t *testing.T) {
	mem := setupTestMemory(t)

	// Seed memories
	messages := []models.AIMessage{
		{Role: models.User, Message: "My favorite programming language is Rust", Timestamp: time.Now(), UniqueId: "1"},
		{Role: models.Assistant, Message: "Rust is known for memory safety!", Timestamp: time.Now(), UniqueId: "2"},
	}
	mem.UpdateMessageHistory(messages)
	time.Sleep(2 * time.Second)

	// Measure recall latency
	iterations := 5
	var totalLatency time.Duration

	for i := 0; i < iterations; i++ {
		start := time.Now()
		_, err := mem.GetContext("What is my favorite language?")
		elapsed := time.Since(start)

		if err != nil {
			t.Logf("Iteration %d error: %v", i, err)
		}

		totalLatency += elapsed
		t.Logf("Iteration %d latency: %v", i, elapsed)
	}

	avgLatency := totalLatency / time.Duration(iterations)
	t.Logf("Average recall latency: %v", avgLatency)

	// Warn if latency is too high (>2 seconds)
	if avgLatency > 2*time.Second {
		t.Logf("WARNING: Average recall latency exceeds 2 seconds: %v", avgLatency)
	}
}

func TestRecallLatencyByMode(t *testing.T) {
	mem := setupTestMemory(t)

	// Seed memories
	messages := []models.AIMessage{
		{Role: models.User, Message: "I work as a software architect at Google", Timestamp: time.Now(), UniqueId: "1"},
		{Role: models.Assistant, Message: "That's an impressive role!", Timestamp: time.Now(), UniqueId: "2"},
	}
	mem.UpdateMessageHistory(messages)
	time.Sleep(2 * time.Second)

	modes := []RetrievalMode{RetrievalModeAuto, RetrievalModeConscious}

	for _, mode := range modes {
		mem.UseRetrievalMode(mode)

		start := time.Now()
		_, err := mem.GetContext("Where do I work?")
		elapsed := time.Since(start)

		if err != nil {
			t.Logf("Mode %s error: %v", mode, err)
		}

		t.Logf("Mode %s latency: %v", mode, elapsed)
	}
}

func TestRecallLatencyUnderLoad(t *testing.T) {
	mem := setupTestMemory(t)

	// Seed memories
	messages := []models.AIMessage{
		{Role: models.User, Message: "I prefer dark mode in all applications", Timestamp: time.Now(), UniqueId: "1"},
		{Role: models.Assistant, Message: "Dark mode is easier on the eyes!", Timestamp: time.Now(), UniqueId: "2"},
	}
	mem.UpdateMessageHistory(messages)
	time.Sleep(2 * time.Second)

	concurrentRequests := 5
	var wg sync.WaitGroup
	latencies := make(chan time.Duration, concurrentRequests)

	for i := 0; i < concurrentRequests; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			start := time.Now()
			_, _ = mem.GetContext("What theme do I prefer?")
			latencies <- time.Since(start)
		}(i)
	}

	wg.Wait()
	close(latencies)

	var total time.Duration
	var max time.Duration
	var min = time.Hour
	count := 0

	for lat := range latencies {
		total += lat
		count++
		if lat > max {
			max = lat
		}
		if lat < min {
			min = lat
		}
	}

	avg := total / time.Duration(count)
	t.Logf("Concurrent recall latency stats:")
	t.Logf("  Min: %v", min)
	t.Logf("  Max: %v", max)
	t.Logf("  Avg: %v", avg)
}

func TestChatCompletionLatency(t *testing.T) {
	mem := setupTestMemory(t)

	// Seed context
	messages := []models.AIMessage{
		{Role: models.User, Message: "I am building a microservices architecture", Timestamp: time.Now(), UniqueId: "1"},
		{Role: models.Assistant, Message: "Microservices provide great scalability!", Timestamp: time.Now(), UniqueId: "2"},
	}
	mem.UpdateMessageHistory(messages)
	time.Sleep(1 * time.Second)

	iterations := 3
	var totalLatency time.Duration

	for i := 0; i < iterations; i++ {
		start := time.Now()
		_, err := mem.ChatCompletion("What architecture am I building?")
		elapsed := time.Since(start)

		if err != nil {
			t.Logf("Iteration %d error: %v", i, err)
			continue
		}

		totalLatency += elapsed
		t.Logf("ChatCompletion iteration %d latency: %v", i, elapsed)
	}

	avgLatency := totalLatency / time.Duration(iterations)
	t.Logf("Average ChatCompletion latency: %v", avgLatency)
}

func TestMemoryIngestionLatency(t *testing.T) {
	mem := setupTestMemory(t)

	messages := []models.AIMessage{
		{Role: models.User, Message: "I completed a marathon last week", Timestamp: time.Now(), UniqueId: "1"},
		{Role: models.Assistant, Message: "Congratulations on your achievement!", Timestamp: time.Now(), UniqueId: "2"},
	}

	start := time.Now()
	mem.UpdateMessageHistory(messages)
	updateLatency := time.Since(start)

	t.Logf("UpdateMessageHistory latency (sync part): %v", updateLatency)

	// Wait for async ingestion to complete
	time.Sleep(5 * time.Second)

	// Verify memory was ingested by trying to recall it
	recallStart := time.Now()
	context, err := mem.GetContext("What did I complete recently?")
	recallLatency := time.Since(recallStart)

	if err != nil {
		t.Logf("Recall error: %v", err)
	}

	t.Logf("Post-ingestion recall latency: %v", recallLatency)
	t.Logf("Retrieved context: %s", context)
}

func TestP50P99Latency(t *testing.T) {
	mem := setupTestMemory(t)

	// Seed memories
	messages := []models.AIMessage{
		{Role: models.User, Message: "My email is test@example.com", Timestamp: time.Now(), UniqueId: "1"},
		{Role: models.Assistant, Message: "Got it, I'll remember your email.", Timestamp: time.Now(), UniqueId: "2"},
	}
	mem.UpdateMessageHistory(messages)
	time.Sleep(2 * time.Second)

	iterations := 10
	latencies := make([]time.Duration, iterations)

	for i := 0; i < iterations; i++ {
		start := time.Now()
		_, _ = mem.GetContext("What is my email?")
		latencies[i] = time.Since(start)
	}

	// Sort latencies
	for i := 0; i < len(latencies); i++ {
		for j := i + 1; j < len(latencies); j++ {
			if latencies[j] < latencies[i] {
				latencies[i], latencies[j] = latencies[j], latencies[i]
			}
		}
	}

	p50Index := len(latencies) / 2
	p99Index := int(float64(len(latencies)) * 0.99)
	if p99Index >= len(latencies) {
		p99Index = len(latencies) - 1
	}

	t.Logf("Latency percentiles over %d iterations:", iterations)
	t.Logf("  P50: %v", latencies[p50Index])
	t.Logf("  P99: %v", latencies[p99Index])
	t.Logf("  Min: %v", latencies[0])
	t.Logf("  Max: %v", latencies[len(latencies)-1])
}

func TestZZZBenchmarkSummary(t *testing.T) {
	fmt.Println("\n" + strings.Repeat("=", 90))
	fmt.Println("                       KARMA MEMORY BENCHMARK SUMMARY")
	fmt.Println("                     (Auto Mode vs Conscious Mode Comparison)")
	fmt.Println(strings.Repeat("=", 90))

	apiKey := config.GetEnvRaw("OPENAI_KEY")
	if apiKey == "" {
		t.Skip("OPENAI_KEY not set")
	}

	redisAvailable := config.DefaultConfig().RedisURL != ""

	type latencyStats struct {
		min  time.Duration
		max  time.Duration
		avg  time.Duration
		best time.Duration // best = min (lowest latency is best case)
		all  []time.Duration
	}

	// Results structure: mode -> cache_type -> stats
	// Modes: auto, conscious
	// Cache types: no_cache, memory_cache, redis_cache
	results := make(map[string]map[string]*latencyStats)
	results["auto"] = make(map[string]*latencyStats)
	results["conscious"] = make(map[string]*latencyStats)
	for _, mode := range []string{"auto", "conscious"} {
		results[mode]["no_cache"] = &latencyStats{all: make([]time.Duration, 0)}
		results[mode]["memory_cache"] = &latencyStats{all: make([]time.Duration, 0)}
		results[mode]["redis_cache"] = &latencyStats{all: make([]time.Duration, 0)}
	}

	iterations := 5 // Iterations for measurement
	warmupRuns := 2 // Warmup runs before measuring

	kai := ai.NewKarmaAI(ai.Llama33_70B, ai.Groq)

	// Create memory instances for Auto mode
	autoNoCache := NewKarmaMemory(kai, "bench_auto_nocache", "bench_scope")
	autoNoCache.UseMemoryLLM(memory_llm, memory_llm_provider)
	autoNoCache.UseService(service)
	autoNoCache.DisableCache()
	autoNoCache.UseRetrievalMode(RetrievalModeAuto)

	autoMemCache := NewKarmaMemory(kai, "bench_auto_memcache", "bench_scope")
	autoMemCache.UseMemoryLLM(memory_llm, memory_llm_provider)
	autoMemCache.UseService(service)
	autoMemCache.EnableMemoryCache(CacheConfig{Enabled: true})
	autoMemCache.UseRetrievalMode(RetrievalModeAuto)

	var autoRedisCache *KarmaMemory
	if redisAvailable {
		autoRedisCache = NewKarmaMemory(kai, "bench_auto_rediscache", "bench_scope")
		autoRedisCache.UseMemoryLLM(memory_llm, memory_llm_provider)
		autoRedisCache.UseService(service)
		autoRedisCache.EnableRedisCache(CacheConfig{Enabled: true})
		autoRedisCache.UseRetrievalMode(RetrievalModeAuto)
	}

	// Create memory instances for Conscious mode
	consciousNoCache := NewKarmaMemory(kai, "bench_conscious_nocache", "bench_scope")
	consciousNoCache.UseMemoryLLM(memory_llm, memory_llm_provider)
	consciousNoCache.UseService(service)
	consciousNoCache.DisableCache()
	consciousNoCache.UseRetrievalMode(RetrievalModeConscious)

	consciousMemCache := NewKarmaMemory(kai, "bench_conscious_memcache", "bench_scope")
	consciousMemCache.UseMemoryLLM(memory_llm, memory_llm_provider)
	consciousMemCache.UseService(service)
	consciousMemCache.EnableMemoryCache(CacheConfig{Enabled: true})
	consciousMemCache.UseRetrievalMode(RetrievalModeConscious)

	var consciousRedisCache *KarmaMemory
	if redisAvailable {
		consciousRedisCache = NewKarmaMemory(kai, "bench_conscious_rediscache", "bench_scope")
		consciousRedisCache.UseMemoryLLM(memory_llm, memory_llm_provider)
		consciousRedisCache.UseService(service)
		consciousRedisCache.EnableRedisCache(CacheConfig{Enabled: true})
		consciousRedisCache.UseRetrievalMode(RetrievalModeConscious)
	}

	// Seed memories
	messages := []models.AIMessage{
		{Role: models.User, Message: "I use Go for backend and React for frontend development. My favorite database is PostgreSQL.", Timestamp: time.Now(), UniqueId: "1"},
		{Role: models.Assistant, Message: "Great full-stack combination! Go and React work well together.", Timestamp: time.Now(), UniqueId: "2"},
		{Role: models.User, Message: "Always respond in English and be concise.", Timestamp: time.Now(), UniqueId: "3"},
		{Role: models.Assistant, Message: "Understood, I'll keep responses in English and concise.", Timestamp: time.Now(), UniqueId: "4"},
	}

	// Update all instances with messages
	allInstances := []*KarmaMemory{autoNoCache, autoMemCache, consciousNoCache, consciousMemCache}
	if autoRedisCache != nil {
		allInstances = append(allInstances, autoRedisCache)
	}
	if consciousRedisCache != nil {
		allInstances = append(allInstances, consciousRedisCache)
	}
	for _, mem := range allInstances {
		mem.UpdateMessageHistory(messages)
	}
	fmt.Println("\n  Waiting for memory ingestion...")
	time.Sleep(4 * time.Second)

	query := "What technologies do I use?"

	// Helper to calculate stats
	calcStats := func(stats *latencyStats) {
		if len(stats.all) == 0 {
			return
		}
		stats.min = stats.all[0]
		stats.max = stats.all[0]
		var total time.Duration
		for _, d := range stats.all {
			total += d
			if d < stats.min {
				stats.min = d
			}
			if d > stats.max {
				stats.max = d
			}
		}
		stats.avg = total / time.Duration(len(stats.all))
		stats.best = stats.min
	}

	// Helper to run benchmark
	runBenchmark := func(name string, mem *KarmaMemory, stats *latencyStats) {
		// Warmup
		for i := 0; i < warmupRuns; i++ {
			_, _ = mem.GetContext(query)
		}
		// Measure
		for i := 0; i < iterations; i++ {
			start := time.Now()
			_, _ = mem.GetContext(query)
			stats.all = append(stats.all, time.Since(start))
		}
		calcStats(stats)
	}

	// ==================== AUTO MODE BENCHMARKS ====================
	fmt.Println("\n" + strings.Repeat("-", 90))
	fmt.Println("                              AUTO MODE BENCHMARKS")
	fmt.Println(strings.Repeat("-", 90))

	fmt.Println("  [1/3] Auto Mode - No Cache...")
	runBenchmark("auto_no_cache", autoNoCache, results["auto"]["no_cache"])

	fmt.Println("  [2/3] Auto Mode - Memory Cache...")
	runBenchmark("auto_memory_cache", autoMemCache, results["auto"]["memory_cache"])

	if autoRedisCache != nil {
		fmt.Println("  [3/3] Auto Mode - Redis Cache...")
		runBenchmark("auto_redis_cache", autoRedisCache, results["auto"]["redis_cache"])
	}

	// ==================== CONSCIOUS MODE BENCHMARKS ====================
	fmt.Println("\n" + strings.Repeat("-", 90))
	fmt.Println("                           CONSCIOUS MODE BENCHMARKS")
	fmt.Println(strings.Repeat("-", 90))

	fmt.Println("  [1/3] Conscious Mode - No Cache...")
	runBenchmark("conscious_no_cache", consciousNoCache, results["conscious"]["no_cache"])

	fmt.Println("  [2/3] Conscious Mode - Memory Cache...")
	runBenchmark("conscious_memory_cache", consciousMemCache, results["conscious"]["memory_cache"])

	if consciousRedisCache != nil {
		fmt.Println("  [3/3] Conscious Mode - Redis Cache...")
		runBenchmark("conscious_redis_cache", consciousRedisCache, results["conscious"]["redis_cache"])
	}

	// ==================== RESULTS TABLE ====================
	fmt.Println("\n" + strings.Repeat("=", 90))
	fmt.Println("                              LATENCY RESULTS")
	fmt.Println(strings.Repeat("=", 90))

	printModeResults := func(modeName string, modeResults map[string]*latencyStats) {
		fmt.Printf("\n  %s MODE:\n", modeName)
		fmt.Println("  " + strings.Repeat("-", 86))
		fmt.Printf("  %-22s %12s %12s %12s %12s %12s\n", "Cache Type", "Best", "Avg", "Max", "Spread", "Speedup")
		fmt.Println("  " + strings.Repeat("-", 86))

		baseline := modeResults["no_cache"].best
		if baseline == 0 {
			baseline = 1 // Prevent division by zero
		}

		// No cache (baseline)
		spread := modeResults["no_cache"].max - modeResults["no_cache"].min
		fmt.Printf("  %-22s %12v %12v %12v %12v %12s\n",
			"No Cache (baseline)",
			modeResults["no_cache"].best,
			modeResults["no_cache"].avg,
			modeResults["no_cache"].max,
			spread,
			"1.00x")

		// Memory cache
		if len(modeResults["memory_cache"].all) > 0 {
			memSpeedup := float64(baseline) / float64(modeResults["memory_cache"].best)
			memSpread := modeResults["memory_cache"].max - modeResults["memory_cache"].min
			fmt.Printf("  %-22s %12v %12v %12v %12v %11.2fx\n",
				"Memory Cache",
				modeResults["memory_cache"].best,
				modeResults["memory_cache"].avg,
				modeResults["memory_cache"].max,
				memSpread,
				memSpeedup)
		}

		// Redis cache
		if len(modeResults["redis_cache"].all) > 0 {
			redisSpeedup := float64(baseline) / float64(modeResults["redis_cache"].best)
			redisSpread := modeResults["redis_cache"].max - modeResults["redis_cache"].min
			fmt.Printf("  %-22s %12v %12v %12v %12v %11.2fx\n",
				"Redis Cache",
				modeResults["redis_cache"].best,
				modeResults["redis_cache"].avg,
				modeResults["redis_cache"].max,
				redisSpread,
				redisSpeedup)
		} else {
			fmt.Printf("  %-22s %12s %12s %12s %12s %12s\n", "Redis Cache", "N/A", "N/A", "N/A", "N/A", "N/A")
		}
		fmt.Println("  " + strings.Repeat("-", 86))
	}

	printModeResults("AUTO", results["auto"])
	printModeResults("CONSCIOUS", results["conscious"])

	// ==================== BEST CASE COMPARISON ====================
	fmt.Println("\n" + strings.Repeat("=", 90))
	fmt.Println("                        BEST CASE PERFORMANCE SUMMARY")
	fmt.Println(strings.Repeat("=", 90))

	fmt.Println("\n  AUTO MODE (Cache hits skip vector for rules/facts only):")
	autoBaseline := results["auto"]["no_cache"].best
	if len(results["auto"]["memory_cache"].all) > 0 && autoBaseline > 0 {
		autoMemSpeedup := float64(autoBaseline) / float64(results["auto"]["memory_cache"].best)
		autoMemSaved := autoBaseline - results["auto"]["memory_cache"].best
		fmt.Printf("    Memory Cache: %v best (%.2fx faster, %v saved/request)\n",
			results["auto"]["memory_cache"].best, autoMemSpeedup, autoMemSaved)
	}
	if len(results["auto"]["redis_cache"].all) > 0 && autoBaseline > 0 {
		autoRedisSpeedup := float64(autoBaseline) / float64(results["auto"]["redis_cache"].best)
		autoRedisSaved := autoBaseline - results["auto"]["redis_cache"].best
		fmt.Printf("    Redis Cache:  %v best (%.2fx faster, %v saved/request)\n",
			results["auto"]["redis_cache"].best, autoRedisSpeedup, autoRedisSaved)
	}

	fmt.Println("\n  CONSCIOUS MODE (Cache hits can skip vector service entirely):")
	consciousBaseline := results["conscious"]["no_cache"].best
	if len(results["conscious"]["memory_cache"].all) > 0 && consciousBaseline > 0 {
		consciousMemSpeedup := float64(consciousBaseline) / float64(results["conscious"]["memory_cache"].best)
		consciousMemSaved := consciousBaseline - results["conscious"]["memory_cache"].best
		fmt.Printf("    Memory Cache: %v best (%.2fx faster, %v saved/request)\n",
			results["conscious"]["memory_cache"].best, consciousMemSpeedup, consciousMemSaved)
	}
	if len(results["conscious"]["redis_cache"].all) > 0 && consciousBaseline > 0 {
		consciousRedisSpeedup := float64(consciousBaseline) / float64(results["conscious"]["redis_cache"].best)
		consciousRedisSaved := consciousBaseline - results["conscious"]["redis_cache"].best
		fmt.Printf("    Redis Cache:  %v best (%.2fx faster, %v saved/request)\n",
			results["conscious"]["redis_cache"].best, consciousRedisSpeedup, consciousRedisSaved)
	}

	// ==================== MODE COMPARISON ====================
	fmt.Println("\n" + strings.Repeat("-", 90))
	fmt.Println("                           MODE COMPARISON (Best Case)")
	fmt.Println(strings.Repeat("-", 90))

	fmt.Printf("\n  %-30s %15s %15s\n", "Configuration", "Auto Mode", "Conscious Mode")
	fmt.Println("  " + strings.Repeat("-", 60))

	fmt.Printf("  %-30s %15v %15v\n", "No Cache",
		results["auto"]["no_cache"].best,
		results["conscious"]["no_cache"].best)

	fmt.Printf("  %-30s %15v %15v\n", "Memory Cache",
		results["auto"]["memory_cache"].best,
		results["conscious"]["memory_cache"].best)

	if redisAvailable {
		fmt.Printf("  %-30s %15v %15v\n", "Redis Cache",
			results["auto"]["redis_cache"].best,
			results["conscious"]["redis_cache"].best)
	}

	// ==================== CACHE STATISTICS ====================
	fmt.Println("\n" + strings.Repeat("-", 90))
	fmt.Println("                              CACHE STATISTICS")
	fmt.Println(strings.Repeat("-", 90))

	if autoMemCache.IsCacheEnabled() {
		stats, _ := autoMemCache.GetCacheStats()
		fmt.Printf("\n  Auto Mode - Memory Cache:\n")
		fmt.Printf("    Backend: %v, Total Entries: %v\n", stats["backend"], stats["total_entries"])
	}

	if consciousMemCache.IsCacheEnabled() {
		stats, _ := consciousMemCache.GetCacheStats()
		fmt.Printf("\n  Conscious Mode - Memory Cache:\n")
		fmt.Printf("    Backend: %v, Total Entries: %v\n", stats["backend"], stats["total_entries"])
	}

	if redisAvailable && autoRedisCache != nil {
		stats, _ := autoRedisCache.GetCacheStats()
		fmt.Printf("\n  Auto Mode - Redis Cache:\n")
		fmt.Printf("    Backend: %v, Local Entries: %v\n", stats["backend"], stats["local_cache_entries"])
	}

	if redisAvailable && consciousRedisCache != nil {
		stats, _ := consciousRedisCache.GetCacheStats()
		fmt.Printf("\n  Conscious Mode - Redis Cache:\n")
		fmt.Printf("    Backend: %v, Local Entries: %v\n", stats["backend"], stats["local_cache_entries"])
	}

	// ==================== FINAL SUMMARY ====================
	fmt.Println("\n" + strings.Repeat("=", 90))
	fmt.Println("                                 SUMMARY")
	fmt.Println(strings.Repeat("=", 90))

	fmt.Printf("\n  Test Config: %d iterations after %d warmup runs per configuration\n", iterations, warmupRuns)

	// Determine best configurations
	fmt.Println("\n  RECOMMENDATIONS:")

	// Auto mode analysis
	autoMemSpeedup := float64(1)
	if len(results["auto"]["memory_cache"].all) > 0 && results["auto"]["memory_cache"].best > 0 {
		autoMemSpeedup = float64(results["auto"]["no_cache"].best) / float64(results["auto"]["memory_cache"].best)
	}
	if autoMemSpeedup > 1.5 {
		fmt.Printf("    ✓ Auto + Memory Cache: %.2fx speedup - RECOMMENDED for low-latency\n", autoMemSpeedup)
	} else if autoMemSpeedup > 1.0 {
		fmt.Printf("    ~ Auto + Memory Cache: %.2fx speedup - modest improvement\n", autoMemSpeedup)
	}

	// Conscious mode analysis
	consciousMemSpeedup := float64(1)
	if len(results["conscious"]["memory_cache"].all) > 0 && results["conscious"]["memory_cache"].best > 0 {
		consciousMemSpeedup = float64(results["conscious"]["no_cache"].best) / float64(results["conscious"]["memory_cache"].best)
	}
	if consciousMemSpeedup > 1.5 {
		fmt.Printf("    ✓ Conscious + Memory Cache: %.2fx speedup - RECOMMENDED for smart retrieval\n", consciousMemSpeedup)
	} else if consciousMemSpeedup > 1.0 {
		fmt.Printf("    ~ Conscious + Memory Cache: %.2fx speedup - modest improvement\n", consciousMemSpeedup)
	}

	// Compare modes
	if len(results["auto"]["memory_cache"].all) > 0 && len(results["conscious"]["memory_cache"].all) > 0 {
		autoBest := results["auto"]["memory_cache"].best
		consciousBest := results["conscious"]["memory_cache"].best
		if autoBest < consciousBest {
			diff := consciousBest - autoBest
			fmt.Printf("\n    → Auto mode is %v faster than Conscious mode (with cache)\n", diff)
			fmt.Printf("      Use Auto for speed, Conscious for intelligent filtering\n")
		} else if consciousBest < autoBest {
			diff := autoBest - consciousBest
			fmt.Printf("\n    → Conscious mode is %v faster than Auto mode (with cache)\n", diff)
			fmt.Printf("      Conscious mode benefits more from cache (skips vector service)\n")
		} else {
			fmt.Printf("\n    → Both modes perform similarly with cache\n")
		}
	}

	fmt.Println("\n" + strings.Repeat("=", 90) + "\n")
}
