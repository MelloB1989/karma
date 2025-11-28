# Karma Memory

Karma Memory is a sophisticated, long-term memory system for AI agents. It bridges the gap between stateless LLM calls and persistent, context-aware interactions by storing, retrieving, and managing memories using vector databases and intelligent caching.

## Features

- **Long-term Persistence**: Stores conversations, facts, rules, and skills in vector databases (Pinecone, Upstash).
- **Intelligent Retrieval**:
  - **Auto Mode**: Fast, category-based retrieval for general context.
  - **Conscious Mode**: LLM-driven dynamic queries that filter by category, lifespan, importance, and status.
- **High-Performance Caching**:
  - **Multi-Level Caching**: In-memory (local) and Redis (shared) support.
  - **Smart Invalidation**: Caches rules, facts, skills, and context separately with configurable TTLs.
  - **Parallel Fetching**: Optimizes latency by fetching from cache and vector stores concurrently.
- **Automatic Ingestion**: Asynchronously processes conversation history to extract and store new memories.

## Installation

```bash
go get github.com/MelloB1989/karma/ai/memory
```

## Configuration

Ensure the following environment variables are set:

- `OPENAI_KEY`: Required for embeddings and memory processing.
- `PINECONE_API_KEY` / `UPSTASH_VECTOR_REST_URL`: Depending on your chosen vector service.
- `REDIS_URL`: (Optional) For shared caching.

## Initialization

Initialize the memory system with a KarmaAI client, a user ID, and an optional scope (e.g., project ID, session ID).

```go
import (
    "github.com/MelloB1989/karma/ai"
    "github.com/MelloB1989/karma/ai/memory"
)

func main() {
    // 1. Initialize KarmaAI
    kai := ai.NewKarmaAI(ai.GPT4o, ai.OpenAI)

    // 2. Initialize Memory
    // NewKarmaMemory(client, userID, scope)
    mem := memory.NewKarmaMemory(kai, "user_123", "project_alpha")
    
    // 3. Configure (Optional)
    mem.UseRetrievalMode(memory.RetrievalModeConscious) // Use smarter, dynamic retrieval
    mem.EnableMemoryCache(memory.CacheConfig{Enabled: true}) // Enable local caching
}
```

## Usage

### 1. Simple Usage (Managed Flow)

The easiest way to use Karma Memory is via the built-in `ChatCompletion` methods. These handle context retrieval, prompt augmentation, and history updates automatically.

```go
response, err := mem.ChatCompletion("My name is John and I love Go.")
if err != nil {
    log.Fatal(err)
}

fmt.Println("AI:", response.AIResponse)
// The system has now learned that the user's name is John and they love Go.

// Next query will automatically retrieve this context
response2, err := mem.ChatCompletion("What language do I prefer?")
fmt.Println("AI:", response2.AIResponse) 
// Output: "You prefer Go."
```

**Streaming Support:**

```go
_, err := mem.ChatCompletionStream("Tell me a story about my projects", func(chunk models.StreamedResponse) error {
    fmt.Print(chunk.Content)
    return nil
})
```

### 2. Advanced Usage (Manual Control)

For integration with existing chat loops or custom LLM calls, you can manually retrieve context and update history.

**Step 1: Retrieve Context**

Before sending a prompt to your LLM, fetch relevant context.

```go
userPrompt := "How do I fix this bug in my React app?"

// GetContext returns a formatted string of relevant memories (facts, rules, previous messages)
contextStr, err := mem.GetContext(userPrompt)
if err != nil {
    log.Println("Error retrieving context:", err)
}

// Combine context with your prompt
fullPrompt := fmt.Sprintf("Context:\n%s\n\nUser: %s", contextStr, userPrompt)
```

**Step 2: Update History**

After generating a response, feed the interaction back into the memory system. This triggers the asynchronous ingestion process where the AI analyzes the conversation to store new facts or update existing ones.

```go
// Create message objects
messages := []models.AIMessage{
    {
        Role:      models.User,
        Message:   userPrompt,
        Timestamp: time.Now(),
        UniqueId:  "msg_1",
    },
    {
        Role:      models.Assistant,
        Message:   aiResponseString,
        Timestamp: time.Now(),
        UniqueId:  "msg_2",
    },
}

// Update history (triggers ingestion in background)
mem.UpdateMessageHistory(messages)
```

## Caching

Caching significantly reduces latency and vector database costs.

### In-Memory Cache
Best for single-instance deployments.

```go
mem.EnableMemoryCache(memory.CacheConfig{
    Enabled:    true,
    RulesTTL:   30 * time.Minute,
    FactsTTL:   20 * time.Minute,
    ContextTTL: 10 * time.Minute,
})
```

### Redis Cache
Best for distributed deployments where multiple instances share the same memory state.

```go
// Requires REDIS_URL env var or config
mem.EnableRedisCache(memory.CacheConfig{
    Enabled: true,
    // ... custom TTLs
})
```

## Retrieval Modes

You can switch between retrieval modes based on your application's needs:

- **`RetrievalModeAuto` (Default)**: 
  - Fast and cost-effective.
  - Always retrieves active rules, facts, skills, and context.
  - Uses the raw user prompt for vector search.
  
- **`RetrievalModeConscious`**:
  - Smarter but slightly higher latency.
  - Uses an LLM to analyze the user's prompt and generate a dynamic search query.
  - Filters memories by specific categories, lifespans, or importance levels relevant to the current query.

```go
mem.UseRetrievalMode(memory.RetrievalModeConscious)
```
