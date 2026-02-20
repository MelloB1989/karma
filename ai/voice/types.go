package voice

import (
	"context"
	"net/http"
	"time"

	"github.com/MelloB1989/karma/models"
)

// Provider represents a voice backend for STT and TTS.
type Provider string

const (
	ProviderOpenAI     Provider = "openai"
	ProviderTogether   Provider = "together"
	ProviderElevenLabs Provider = "elevenlabs"
)

// TextAI is the text reasoning interface used by the voice agent.
//
// Use *ai.KarmaAI to preserve MCP/Go-function tool behavior.
type TextAI interface {
	ChatCompletionManaged(messages *models.AIChatHistory) (*models.AIChatResponse, error)
}

// SpeechProvider defines STT and TTS operations.
//
// Custom providers can implement this interface and be wired with NewAgentWithProvider.
type SpeechProvider interface {
	Transcribe(ctx context.Context, req TranscribeRequest) (*TranscribeResponse, error)
	Synthesize(ctx context.Context, req SynthesizeRequest) (*SynthesizeResponse, error)
}

// Agent orchestrates speech I/O with a text AI backend.
type Agent struct {
	provider Provider
	textAI   TextAI
	speech   SpeechProvider
	now      func() time.Time

	stripThinkingTokens   bool
	synthesizeThinkingRaw bool
}

// Config configures all provider clients.
type Config struct {
	HTTPClient *http.Client
	OpenAI     OpenAIConfig
	Together   TogetherConfig
	ElevenLabs ElevenLabsConfig

	// StripThinkingTokens removes <think>...</think> blocks from AI text before
	// history/update output by default.
	StripThinkingTokens bool
	// SynthesizeThinkingRaw controls whether TTS should use raw model output
	// (including <think>...</think>) when available.
	SynthesizeThinkingRaw bool
}

// OpenAIConfig configures the OpenAI speech provider.
type OpenAIConfig struct {
	APIKey    string
	BaseURL   string
	STTModel  VoiceModel
	TTSModel  VoiceModel
	TTSVoice  string
	TTSFormat string
}

// TogetherConfig configures the Together speech provider.
type TogetherConfig struct {
	APIKey    string
	BaseURL   string
	STTModel  VoiceModel
	TTSModel  VoiceModel
	TTSVoice  string
	TTSFormat string
}

// ElevenLabsConfig configures the ElevenLabs websocket provider.
type ElevenLabsConfig struct {
	APIKey                     string
	Token                      string
	BaseWSURL                  string
	TTSModel                   VoiceModel
	TTSVoiceID                 string
	TTSOutputFormat            string
	TTSLanguageCode            string
	TTSInactivityTimeoutSecond int
	STTModel                   VoiceModel
	STTAudioFormat             string
	STTCommitStrategy          string
	STTLanguageCode            string
	IncludeTimestamps          bool
	IncludeLanguageDetection   bool
	ReadTimeout                time.Duration
}

// Option mutates agent config.
type Option func(*Config)

// TranscribeRequest holds STT input.
type TranscribeRequest struct {
	Audio        []byte
	FileName     string
	MIMEType     string
	AudioFormat  string
	Model        VoiceModel
	Language     string
	SampleRate   int
	Prompt       string
	PreviousText string
}

// TranscribeResponse holds STT output.
type TranscribeResponse struct {
	Text string
	Raw  any
}

// SynthesizeRequest holds TTS input.
type SynthesizeRequest struct {
	Text         string
	Model        VoiceModel
	VoiceID      string
	Language     string
	Format       string
	Speed        float64
	SampleRate   int
	Instructions string
}

// SynthesizeResponse holds TTS output.
type SynthesizeResponse struct {
	Audio  []byte
	Format string
	Raw    any
}

// ConverseRequest executes a full voice turn.
type ConverseRequest struct {
	History           *models.AIChatHistory
	Audio             []byte
	UserText          string
	TranscribeRequest TranscribeRequest
	SynthesizeRequest SynthesizeRequest

	// DisableTranscription skips STT and requires UserText.
	DisableTranscription bool
	// DisableSynthesis skips TTS and returns text-only output.
	DisableSynthesis bool
	// SkipSynthesis is a backward-compatible alias for DisableSynthesis.
	SkipSynthesis bool
}

// ConverseResponse returns transcript, text, and optional synthesized audio.
type ConverseResponse struct {
	Transcript   string
	TextResponse *models.AIChatResponse
	Audio        []byte
	AudioFormat  string
	History      *models.AIChatHistory
}
