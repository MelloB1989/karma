package memory

import (
	"github.com/MelloB1989/karma/ai"
	"github.com/MelloB1989/karma/models"
	"go.uber.org/zap"
)

type RetrievalMode string

const (
	RetrievalModeConscious RetrievalMode = "conscious"
	RetrievalModeAuto      RetrievalMode = "auto"
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
	messagesHistory      models.AIChatHistory
	kai                  *ai.KarmaAI
	memoryAI             *ai.KarmaAI
	embeddingAI          *ai.KarmaAI
	retrievalAI          *ai.KarmaAI
	Caching              Caching
	memorydb             *vectorClient
	userID               string // User ID
	scope                string // Application name, service name, etc.
	logger               *zap.Logger
	retrievalMode        RetrievalMode
	currentMemoryContext string
}

func NewKarmaMemory(kai *ai.KarmaAI, userId string, sc ...string) *KarmaMemory {
	// We use "default" scope by default, you can change this by using the useScope function
	scope := "default"
	if len(sc) > 0 {
		scope = sc[0]
	}
	logger, _ := zap.NewProduction()
	memorydb := newVectorClient(userId, scope, logger)

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

func (k *KarmaMemory) UseService(service VectorServices) error {
	return k.memorydb.switchService(k.userID, k.scope, service)
}

func (k *KarmaMemory) UseLogger(logger *zap.Logger) {
	k.logger = logger
}

func (k *KarmaMemory) UseRetrievalMode(mode RetrievalMode) {
	k.retrievalMode = mode
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
				k.logger.Error("karmaMemory: memory ingestion failed",
					zap.String("userID", k.userID),
					zap.String("scope", k.scope),
					zap.Error(err))
			} else {
				k.logger.Info("karmaMemory: memory ingested",
					zap.String("userID", k.userID),
					zap.String("scope", k.scope))
			}
		}()
	}
}

func (k *KarmaMemory) GetContext(userPrompt string) (string, error) {
	mode := k.retrievalMode

	var maxTokens int
	var topK int

	switch mode {
	case RetrievalModeConscious:
		maxTokens = 400
		topK = 3
	case RetrievalModeAuto:
		maxTokens = 800
		topK = 5
	default:
		maxTokens = 300
		topK = 5
	}

	searchQuery, err := k.generateSearchQuery(userPrompt)
	sq := searchQuery.SearchQuery
	if err != nil {
		k.logger.Warn("karmaMemory: search query generation failed, using original prompt",
			zap.Error(err))
		sq = userPrompt
	}

	embeddings, err := k.getEmbeddings(sq)
	if err != nil {
		k.logger.Error("karmaMemory: failed to generate embeddings",
			zap.Error(err))
		return "", err
	}

	vectorResults, err := k.memorydb.client.queryVector(embeddings, topK, searchQuery)
	if err != nil {
		k.logger.Warn("karmaMemory: vector search failed",
			zap.Error(err))
	}

	rules, err := k.memorydb.client.queryVectorByMetadata(filters{
		Category: ptrStr("rule"),
	})
	if err != nil {
		k.logger.Warn("karmaMemory: rules query failed",
			zap.Error(err))
		rules = []map[string]any{}
	}

	relevantMemories := k.selectRelevantMemories(vectorResults, rules, topK)

	context := k.formatContext(relevantMemories, maxTokens)
	k.currentMemoryContext = k.formatContextForIngest(relevantMemories)

	k.logger.Info("karmaMemory: context retrieved",
		zap.String("mode", string(mode)),
		zap.Int("memories_count", len(relevantMemories)),
		zap.Int("context_length", len(context)))

	return context, nil
}
