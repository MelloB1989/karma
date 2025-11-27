package memory

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/MelloB1989/karma/ai"
	"github.com/MelloB1989/karma/config"
	"github.com/MelloB1989/karma/models"
)

func setupTestMemory(t *testing.T) *KarmaMemory {
	t.Helper()

	apiKey := config.GetEnvRaw("OPENAI_KEY")
	if apiKey == "" {
		t.Skip("OPENAI_KEY not set")
	}

	kai := ai.NewKarmaAI(ai.Llama33_70B, ai.Groq)
	mem := NewKarmaMemory(kai, "test_user_123", "test_scope")
	mem.UseMemoryLLM(ai.Llama31_8B, ai.Groq)
	mem.UseService(VectorServicePinecone)

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
