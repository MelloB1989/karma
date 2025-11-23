package memory

import (
	"fmt"
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

	kai := ai.NewKarmaAI(ai.GPT4oMini, ai.OpenAI)
	mem := NewKarmaMemory(kai, "test_user_123", "test_scope")

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
	time.Sleep(1 * time.Second)

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
