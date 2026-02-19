package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"

	"github.com/MelloB1989/karma/ai"
	"github.com/MelloB1989/karma/ai/voice"
	"github.com/MelloB1989/karma/models"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024 * 64,
	WriteBufferSize: 1024 * 64,
	// Allow all origins for the demo; tighten this in production.
	CheckOrigin: func(r *http.Request) bool { return true },
}

const (
	inputAudioFormatWebM     = "webm"
	inputAudioFormatPCM16kHz = "pcm16_16000"
	defaultVoiceProvider     = voice.ProviderTogether
	voiceProviderEnvName     = "KARMA_VOICE_PROVIDER"
	sttLanguageEnvName       = "KARMA_VOICE_STT_LANGUAGE"
	voiceProviderOpenAI      = voice.ProviderOpenAI
	voiceProviderTogether    = voice.ProviderTogether
	voiceProviderElevenLabs  = voice.ProviderElevenLabs
)

type voiceRuntime struct {
	agent             *voice.Agent
	provider          voice.Provider
	inputAudioFormat  string
	transcribeRequest voice.TranscribeRequest
}

func main() {
	// Serve the static HTML frontend.
	http.Handle("/", http.FileServer(http.Dir("static")))

	// WebSocket endpoint for voice interactions.
	http.HandleFunc("/ws", handleVoiceWS)

	port := "8080"
	if p := os.Getenv("PORT"); p != "" {
		port = p
	}

	log.Printf("🎙️  Voice AI Agent → http://localhost:%s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

// session holds per-connection state.
type session struct {
	mu      sync.Mutex
	history *models.AIChatHistory
}

func (s *session) getHistory() *models.AIChatHistory {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.history
}

func (s *session) setHistory(h *models.AIChatHistory) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.history = h
}

func (s *session) resetHistory() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.history = &models.AIChatHistory{}
}

func handleVoiceWS(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// ── Text AI (brain) ────────────────────────────────────────────────────
	textAI := ai.NewKarmaAI(
		ai.Llama33_70B,
		ai.Groq,
		ai.WithSystemMessage(
			"You are a helpful, concise voice assistant. "+
				"Keep every answer to 1–3 short sentences and use plain, "+
				"conversational language. Never use markdown, bullet points, "+
				"or any special formatting.",
		),
		ai.WithMaxTokens(300),
		ai.WithTemperature(0.8),
	)

	// ── Voice runtime (provider + expected input format + STT metadata) ────
	runtime, err := buildVoiceRuntime(textAI)
	if err != nil {
		log.Printf("Failed to create voice runtime: %v", err)
		conn.Close()
		return
	}
	log.Printf(
		"[session] voice provider=%s input_audio_format=%s stt_language=%q",
		runtime.provider,
		runtime.inputAudioFormat,
		runtime.transcribeRequest.Language,
	)

	sess := &session{history: &models.AIChatHistory{}}

	// We declare `handler` before the closure so the closure can reference it
	// by the time Run() calls it (handler is guaranteed non-nil at that point).
	var handler *voice.WebSocketHandler

	handler, err = voice.NewWebSocketHandler(conn,
		voice.WithWSMessageHandler(func(ctx context.Context, msg voice.WSMessage) error {
			// ── JSON control messages ──────────────────────────────────
			if msg.JSON != nil {
				return handleControlMessage(ctx, handler, sess, msg.JSON)
			}

			// ── Binary audio payload ───────────────────────────────────
			if len(msg.Data) == 0 {
				return nil
			}

			log.Printf("[session] received %d bytes of audio", len(msg.Data))

			// Tell the client we are working on it.
			if sendErr := handler.SendJSON(ctx, map[string]any{
				"event": "processing",
			}); sendErr != nil {
				return sendErr
			}

			resp, err := runtime.agent.Converse(ctx, voice.ConverseRequest{
				History:           sess.getHistory(),
				Audio:             msg.Data,
				TranscribeRequest: runtime.transcribeRequest,
			})
			if err != nil {
				if isNoSpeechError(err) {
					log.Printf("[session] no speech detected: %v", err)
					return handler.SendJSON(ctx, map[string]any{
						"event":   "no_speech",
						"message": "No speech detected. Please try again.",
					})
				}
				log.Printf("[session] Converse error: %v", err)
				return handler.SendJSON(ctx, map[string]any{
					"event":   "error",
					"message": err.Error(),
				})
			}

			// Persist the updated history for the next turn.
			sess.setHistory(resp.History)

			aiText := ""
			if resp.TextResponse != nil {
				aiText = resp.TextResponse.AIResponse
			}

			payload := map[string]any{
				"event":      "response",
				"transcript": resp.Transcript,
				"text":       aiText,
				"audio":      base64.StdEncoding.EncodeToString(resp.Audio),
				"format":     resp.AudioFormat,
			}

			log.Printf("[session] transcript=%q reply=%q audio=%d bytes",
				resp.Transcript, aiText, len(resp.Audio))

			return handler.SendJSON(ctx, payload)
		}),

		voice.WithWSCloseHandler(func(code int, text string) error {
			log.Printf("[session] client disconnected (code=%d text=%q)", code, text)
			cancel()
			return nil
		}),
	)
	if err != nil {
		log.Printf("Failed to create WebSocket handler: %v", err)
		conn.Close()
		return
	}

	// Announce readiness to the client.
	if err := handler.SendJSON(ctx, map[string]any{
		"event":              "ready",
		"provider":           runtime.provider,
		"input_audio_format": runtime.inputAudioFormat,
	}); err != nil {
		log.Printf("Failed to send ready event: %v", err)
		return
	}

	if err := handler.Run(ctx); err != nil && ctx.Err() == nil {
		log.Printf("[session] ended: %v", err)
	}
}

func buildVoiceRuntime(textAI voice.TextAI) (*voiceRuntime, error) {
	provider := defaultVoiceProvider
	if raw := strings.TrimSpace(strings.ToLower(os.Getenv(voiceProviderEnvName))); raw != "" {
		provider = voice.Provider(raw)
	}
	sttLanguage := strings.TrimSpace(os.Getenv(sttLanguageEnvName))

	switch provider {
	case voiceProviderOpenAI:
		agent, err := voice.NewOpenAIAgent(textAI)
		if err != nil {
			return nil, err
		}
		return &voiceRuntime{
			agent:            agent,
			provider:         provider,
			inputAudioFormat: inputAudioFormatWebM,
			transcribeRequest: voice.TranscribeRequest{
				FileName: "recording.webm",
				MIMEType: "audio/webm",
				Language: sttLanguage,
			},
		}, nil
	case voiceProviderTogether:
		agent, err := voice.NewTogetherAgent(textAI)
		if err != nil {
			return nil, err
		}
		return &voiceRuntime{
			agent:            agent,
			provider:         provider,
			inputAudioFormat: inputAudioFormatWebM,
			transcribeRequest: voice.TranscribeRequest{
				FileName: "recording.webm",
				MIMEType: "audio/webm",
				Language: sttLanguage,
			},
		}, nil
	case voiceProviderElevenLabs:
		agent, err := voice.NewElevenLabsAgent(textAI)
		if err != nil {
			return nil, err
		}
		if sttLanguage == "" {
			sttLanguage = "en"
		}
		return &voiceRuntime{
			agent:            agent,
			provider:         provider,
			inputAudioFormat: inputAudioFormatPCM16kHz,
			transcribeRequest: voice.TranscribeRequest{
				FileName:    "recording.pcm",
				MIMEType:    "audio/pcm",
				AudioFormat: "pcm_16000",
				SampleRate:  16000,
				Language:    sttLanguage,
			},
		}, nil
	default:
		return nil, fmt.Errorf(
			"unsupported %s=%q (supported: %s, %s, %s)",
			voiceProviderEnvName,
			provider,
			voiceProviderOpenAI,
			voiceProviderTogether,
			voiceProviderElevenLabs,
		)
	}
}

// handleControlMessage processes JSON control frames sent by the client.
func handleControlMessage(
	ctx context.Context,
	handler *voice.WebSocketHandler,
	sess *session,
	payload map[string]any,
) error {
	event, _ := payload["event"].(string)
	switch event {
	case "ping":
		return handler.SendJSON(ctx, map[string]any{"event": "pong"})

	case "reset":
		sess.resetHistory()
		log.Printf("[session] conversation history reset")
		return handler.SendJSON(ctx, map[string]any{"event": "reset_ok"})

	default:
		log.Printf("[session] unknown control event: %q", event)
	}
	return nil
}

func isNoSpeechError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "no speech detected") ||
		strings.Contains(msg, "insufficient audio activity") ||
		strings.Contains(msg, "empty committed transcript") ||
		strings.Contains(msg, "commit_throttled") ||
		strings.Contains(msg, "uncommitted audio")
}
