package voice

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/MelloB1989/karma/models"
)

// NewAgent creates a voice agent using one of the built-in speech providers.
//
// textAI is required and is used for text reasoning so existing AI package features
// (MCP, Go function tools, analytics, prompt controls) continue to work unchanged.
func NewAgent(textAI TextAI, provider Provider, options ...Option) (*Agent, error) {
	if textAI == nil {
		return nil, errors.New("textAI is required")
	}

	cfg := defaultConfig()
	for _, option := range options {
		if option != nil {
			option(&cfg)
		}
	}

	speech, err := newBuiltInSpeechProvider(provider, cfg)
	if err != nil {
		return nil, err
	}

	return &Agent{
		provider:              provider,
		textAI:                textAI,
		speech:                speech,
		now:                   time.Now,
		stripThinkingTokens:   cfg.StripThinkingTokens,
		synthesizeThinkingRaw: cfg.SynthesizeThinkingRaw,
	}, nil
}

// NewOpenAIAgent creates a voice agent using OpenAI for speech I/O.
func NewOpenAIAgent(textAI TextAI, options ...Option) (*Agent, error) {
	return NewAgent(textAI, ProviderOpenAI, options...)
}

// NewTogetherAgent creates a voice agent using Together for speech I/O.
func NewTogetherAgent(textAI TextAI, options ...Option) (*Agent, error) {
	return NewAgent(textAI, ProviderTogether, options...)
}

// NewElevenLabsAgent creates a voice agent using ElevenLabs for speech I/O.
func NewElevenLabsAgent(textAI TextAI, options ...Option) (*Agent, error) {
	return NewAgent(textAI, ProviderElevenLabs, options...)
}

// NewAgentWithProvider creates a voice agent with a custom speech provider.
func NewAgentWithProvider(textAI TextAI, providerName Provider, speechProvider SpeechProvider, options ...Option) (*Agent, error) {
	if textAI == nil {
		return nil, errors.New("textAI is required")
	}
	if speechProvider == nil {
		return nil, errors.New("speechProvider is required")
	}

	cfg := defaultConfig()
	for _, option := range options {
		if option != nil {
			option(&cfg)
		}
	}

	return &Agent{
		provider:              providerName,
		textAI:                textAI,
		speech:                speechProvider,
		now:                   time.Now,
		stripThinkingTokens:   cfg.StripThinkingTokens,
		synthesizeThinkingRaw: cfg.SynthesizeThinkingRaw,
	}, nil
}

func newBuiltInSpeechProvider(provider Provider, cfg Config) (SpeechProvider, error) {
	switch provider {
	case ProviderOpenAI:
		return newOpenAISpeechProvider(cfg.OpenAI, cfg.HTTPClient)
	case ProviderTogether:
		return newTogetherSpeechProvider(cfg.Together, cfg.HTTPClient)
	case ProviderElevenLabs:
		return newElevenLabsProvider(cfg.ElevenLabs, cfg.HTTPClient)
	case ProviderVapi:
		return nil, errors.New("vapi is a call agent provider; use NewVapiAgent instead of NewAgent")
	default:
		return nil, fmt.Errorf("unsupported voice provider: %s", provider)
	}
}

// Provider returns the active speech provider name.
func (a *Agent) Provider() Provider {
	return a.provider
}

// Transcribe converts audio to text using the configured speech provider.
func (a *Agent) Transcribe(ctx context.Context, req TranscribeRequest) (*TranscribeResponse, error) {
	if a == nil || a.speech == nil {
		return nil, errors.New("voice agent is not initialized")
	}
	return a.speech.Transcribe(ctx, req)
}

// Synthesize converts text to audio using the configured speech provider.
func (a *Agent) Synthesize(ctx context.Context, req SynthesizeRequest) (*SynthesizeResponse, error) {
	if a == nil || a.speech == nil {
		return nil, errors.New("voice agent is not initialized")
	}
	return a.speech.Synthesize(ctx, req)
}

// Converse runs one voice turn:
//  1. STT (unless UserText is provided or DisableTranscription is true)
//  2. Text completion through textAI
//  3. TTS (unless DisableSynthesis/SkipSynthesis is true)
func (a *Agent) Converse(ctx context.Context, req ConverseRequest) (*ConverseResponse, error) {
	if a == nil {
		return nil, errors.New("voice agent is nil")
	}

	history := req.History
	if history == nil {
		history = &models.AIChatHistory{}
	}

	transcript := strings.TrimSpace(req.UserText)
	if req.DisableTranscription {
		if transcript == "" {
			return nil, errors.New("user text is required when transcription is disabled")
		}
	} else if transcript == "" {
		trReq := req.TranscribeRequest
		if len(trReq.Audio) == 0 && len(req.Audio) > 0 {
			trReq.Audio = req.Audio
		}
		if len(trReq.Audio) == 0 {
			return nil, errors.New("audio is required when user text is empty")
		}

		sttResponse, err := a.Transcribe(ctx, trReq)
		if err != nil {
			return nil, err
		}
		transcript = strings.TrimSpace(sttResponse.Text)
	}

	if transcript == "" {
		return nil, errors.New("empty transcript after transcription")
	}

	history.Messages = append(history.Messages, a.newMessage(models.User, transcript))

	textResponse, err := a.textAI.ChatCompletionManaged(history)
	if err != nil {
		return &ConverseResponse{
			Transcript: transcript,
			History:    history,
		}, err
	}

	rawAssistantText := ""
	assistantText := ""
	resultTextResponse := textResponse
	if textResponse != nil {
		rawAssistantText = strings.TrimSpace(textResponse.AIResponse)
		assistantText = rawAssistantText
		if a.stripThinkingTokens {
			assistantText = stripThinkingTokens(rawAssistantText)
		}

		if assistantText != textResponse.AIResponse {
			cloned := *textResponse
			cloned.AIResponse = assistantText
			resultTextResponse = &cloned
		}
	}

	if assistantText != "" {
		history.Messages = append(history.Messages, a.newMessage(models.Assistant, assistantText))
	}

	result := &ConverseResponse{
		Transcript:   transcript,
		TextResponse: resultTextResponse,
		History:      history,
	}

	if req.SkipSynthesis || req.DisableSynthesis || resultTextResponse == nil {
		return result, nil
	}

	ttsReq := req.SynthesizeRequest
	if strings.TrimSpace(ttsReq.Text) == "" {
		if a.synthesizeThinkingRaw && rawAssistantText != "" {
			ttsReq.Text = rawAssistantText
		} else {
			ttsReq.Text = assistantText
		}
	}
	if strings.TrimSpace(ttsReq.Text) == "" {
		return result, nil
	}

	ttsResponse, err := a.Synthesize(ctx, ttsReq)
	if err != nil {
		return result, err
	}

	result.Audio = ttsResponse.Audio
	result.AudioFormat = ttsResponse.Format

	return result, nil
}

func (a *Agent) newMessage(role models.AIRoles, message string) models.AIMessage {
	now := a.now()
	id := fmt.Sprintf("voice-%d", now.UnixNano())
	return models.AIMessage{
		Message:   message,
		Role:      role,
		Timestamp: now,
		UniqueId:  id,
	}
}
