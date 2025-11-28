package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/MelloB1989/karma/ai"
	"github.com/MelloB1989/karma/ai/memory"
	"github.com/MelloB1989/karma/models"
)

// Session management
type Session struct {
	Memory     *memory.KarmaMemory
	LastAccess time.Time
}

var (
	sessions = make(map[string]*Session)
	sessMu   sync.RWMutex
)

// Rate limiting
type visitor struct {
	lastSeen time.Time
	tokens   int
}

var (
	visitors = make(map[string]*visitor)
	mtx      sync.Mutex
)

// allowRequest implements a simple token bucket rate limiter
// Allows 20 requests per minute per IP
func allowRequest(ip string) bool {
	mtx.Lock()
	defer mtx.Unlock()

	v, exists := visitors[ip]
	if !exists {
		visitors[ip] = &visitor{
			lastSeen: time.Now(),
			tokens:   20,
		}
		return true
	}

	now := time.Now()
	// Refill tokens if a minute has passed since last seen
	if now.Sub(v.lastSeen) > time.Minute {
		v.tokens = 20
	}
	v.lastSeen = now

	if v.tokens > 0 {
		v.tokens--
		return true
	}

	return false
}

// API Types
type ChatRequest struct {
	SessionID string `json:"session_id"`
	Message   string `json:"message"`
}

type ConfigRequest struct {
	SessionID     string `json:"session_id"`
	RetrievalMode string `json:"retrieval_mode"` // "auto", "conscious"
	CacheMode     string `json:"cache_mode"`     // "none", "memory", "redis"
}

type StreamEvent struct {
	Type          string `json:"type"` // "recall", "token", "done", "error"
	Content       string `json:"content,omitempty"`
	RecallLatency int64  `json:"recall_latency_ms,omitempty"`
	TotalLatency  int64  `json:"total_latency_ms,omitempty"`
	Message       string `json:"message,omitempty"`
}

func getSession(sessionID string) *Session {
	sessMu.Lock()
	defer sessMu.Unlock()

	if sessionID == "" {
		return nil
	}

	sess, exists := sessions[sessionID]
	if !exists {
		// Using Llama31_8B via Groq for fast inference
		kai := ai.NewKarmaAI(ai.Llama31_8B, ai.Groq)

		// Create memory instance
		// Using a unique user ID per session to isolate memories for the demo
		mem := memory.NewKarmaMemory(kai, "demo_user_"+sessionID, "demo_scope")

		// Default config
		mem.UseRetrievalMode(memory.RetrievalModeAuto)
		// Use GPT-4o Mini for memory operations (summarization, extraction)
		mem.UseMemoryLLM(ai.GPT4oMini, ai.OpenAI)
		mem.EnableMemoryCache(memory.CacheConfig{Enabled: true})

		sess = &Session{
			Memory:     mem,
			LastAccess: time.Now(),
		}
		sessions[sessionID] = sess
		log.Printf("Created new session: %s", sessionID)
	}
	sess.LastAccess = time.Now()
	return sess
}

func corsMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next(w, r)
	}
}

func rateLimitMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ip, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			ip = r.RemoteAddr
		}

		if !allowRequest(ip) {
			http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
			return
		}

		next(w, r)
	}
}

func handleChat(w http.ResponseWriter, r *http.Request) {
	var req ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if req.SessionID == "" {
		http.Error(w, "Session ID required", http.StatusBadRequest)
		return
	}

	sess := getSession(req.SessionID)

	// Set headers for SSE
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming not supported", http.StatusInternalServerError)
		return
	}

	// Helper to send SSE events
	sendEvent := func(event StreamEvent) {
		data, err := json.Marshal(event)
		if err != nil {
			log.Printf("Error marshaling event: %v", err)
			return
		}
		fmt.Fprintf(w, "data: %s\n\n", data)
		flusher.Flush()
	}

	// 1. Measure Recall Latency
	// We call GetContext explicitly to measure the latency.
	// Note: ChatCompletionStream will call it again internally, but if caching is enabled,
	// the second call will be extremely fast (in-memory hit).
	startRecall := time.Now()
	_, err := sess.Memory.GetContext(req.Message)
	recallLatency := time.Since(startRecall)

	if err != nil {
		log.Printf("Recall error: %v", err)
		// We don't fail here, we let ChatCompletionStream handle it or proceed without context
	}

	sendEvent(StreamEvent{
		Type:          "recall",
		RecallLatency: recallLatency.Milliseconds(),
	})

	// 2. Stream Chat Completion
	startTotal := time.Now()

	_, err = sess.Memory.ChatCompletionStream(req.Message, func(chunk models.StreamedResponse) error {
		sendEvent(StreamEvent{
			Type:    "token",
			Content: chunk.AIResponse,
		})
		return nil
	})

	if err != nil {
		log.Printf("Chat error for session %s: %v", req.SessionID, err)
		sendEvent(StreamEvent{
			Type:    "error",
			Message: err.Error(),
		})
		return
	}

	totalLatency := time.Since(startTotal)

	// 3. Send Done Event
	sendEvent(StreamEvent{
		Type:         "done",
		TotalLatency: totalLatency.Milliseconds(),
	})
}

func handleConfig(w http.ResponseWriter, r *http.Request) {
	var req ConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if req.SessionID == "" {
		http.Error(w, "Session ID required", http.StatusBadRequest)
		return
	}

	sess := getSession(req.SessionID)
	mem := sess.Memory

	// Set Retrieval Mode
	switch req.RetrievalMode {
	case "conscious":
		mem.UseRetrievalMode(memory.RetrievalModeConscious)
		log.Printf("Session %s: Switched to Conscious Mode", req.SessionID)
	case "auto":
		mem.UseRetrievalMode(memory.RetrievalModeAuto)
		log.Printf("Session %s: Switched to Auto Mode", req.SessionID)
	}

	// Set Cache Mode
	switch req.CacheMode {
	case "none":
		mem.DisableCache()
		log.Printf("Session %s: Cache Disabled", req.SessionID)
	case "memory":
		mem.EnableMemoryCache(memory.CacheConfig{Enabled: true})
		log.Printf("Session %s: Memory Cache Enabled", req.SessionID)
	case "redis":
		redisURL := os.Getenv("REDIS_URL")
		if redisURL == "" {
			log.Println("REDIS_URL not set, cannot enable Redis cache")
			http.Error(w, "REDIS_URL not configured on server", http.StatusBadRequest)
			return
		}
		mem.EnableRedisCache(memory.CacheConfig{Enabled: true})
		log.Printf("Session %s: Redis Cache Enabled", req.SessionID)
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "updated"})
}

func handleReset(w http.ResponseWriter, r *http.Request) {
	var req ChatRequest // Reusing struct for SessionID
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if req.SessionID == "" {
		http.Error(w, "Session ID required", http.StatusBadRequest)
		return
	}

	sess := getSession(req.SessionID)
	sess.Memory.ClearHistory()

	log.Printf("Session %s: History cleared", req.SessionID)

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "history_cleared"})
}

func main() {
	// Cleanup routine for sessions
	go func() {
		for {
			time.Sleep(10 * time.Minute)
			sessMu.Lock()
			for id, sess := range sessions {
				if time.Since(sess.LastAccess) > 1*time.Hour {
					delete(sessions, id)
					log.Printf("Cleaned up session: %s", id)
				}
			}
			sessMu.Unlock()
		}
	}()

	http.HandleFunc("/api/chat", corsMiddleware(rateLimitMiddleware(handleChat)))
	http.HandleFunc("/api/config", corsMiddleware(rateLimitMiddleware(handleConfig)))
	http.HandleFunc("/api/reset", corsMiddleware(rateLimitMiddleware(handleReset)))

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Karma UI Backend starting on port %s...", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
