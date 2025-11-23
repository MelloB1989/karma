package memory

import (
	"github.com/MelloB1989/karma/ai"
	"github.com/MelloB1989/karma/models"
)

type CachingModes string

const (
	CachingModeMemory  CachingModes = "memory"
	CachingModeHistory CachingModes = "redis"
)

type Caching struct {
	Mode    CachingModes
	Enabled bool
}

type KarmaMemory struct {
	messagesHistory models.AIChatHistory
	kai             *ai.KarmaAI
	memoryAI        *ai.KarmaAI
	embeddingAI     *ai.KarmaAI
	Caching         Caching
	memorydb        *dbClient
	userID          string // User ID
	scope           string // Application name, service name, etc.
}

func NewKarmaMemory(kai *ai.KarmaAI, userId string, sc ...string) *KarmaMemory {
	// We use "default" scope by default, you can change this by using the useScope function
	scope := "default"
	if len(sc) > 0 {
		scope = sc[0]
	}
	memorydb := newDBClient(userId, scope)
	memorydb.runMigrations()
	return &KarmaMemory{
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
		memorydb:    memorydb,
		userID:      userId,
		scope:       scope,
	}
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

func (k *KarmaMemory) UseMemoryLLM(llm ai.BaseModel, provider ai.Provider) {
	k.memoryAI = ai.NewKarmaAI(llm, provider,
		ai.WithSystemMessage(memoryLLMSystemPrompt),
		ai.WithMaxTokens(memoryLLMMaxTokens),
		ai.WithTemperature(1))
}

func (k *KarmaMemory) UseEmbeddingLLM(llm ai.BaseModel, provider ai.Provider) {
	k.embeddingAI = ai.NewKarmaAI(llm, provider)
}

// History functions
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
}
