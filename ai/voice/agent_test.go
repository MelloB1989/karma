package voice

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/MelloB1989/karma/models"
)

type fakeTextAI struct {
	response *models.AIChatResponse
	err      error
	history  []models.AIChatHistory
}

func (f *fakeTextAI) ChatCompletionManaged(messages *models.AIChatHistory) (*models.AIChatResponse, error) {
	f.history = append(f.history, *messages)
	if f.err != nil {
		return nil, f.err
	}
	if f.response == nil {
		return &models.AIChatResponse{AIResponse: "ok"}, nil
	}
	return f.response, nil
}

type fakeSpeechProvider struct {
	transcribeResponse *TranscribeResponse
	synthesizeResponse *SynthesizeResponse
	transcribeErr      error
	synthesizeErr      error
	transcribeCalls    int
	synthesizeCalls    int
	lastTranscribeReq  TranscribeRequest
	lastSynthesizeReq  SynthesizeRequest
}

func (f *fakeSpeechProvider) Transcribe(_ context.Context, req TranscribeRequest) (*TranscribeResponse, error) {
	f.transcribeCalls++
	f.lastTranscribeReq = req
	if f.transcribeErr != nil {
		return nil, f.transcribeErr
	}
	if f.transcribeResponse == nil {
		return &TranscribeResponse{Text: "default transcript"}, nil
	}
	return f.transcribeResponse, nil
}

func (f *fakeSpeechProvider) Synthesize(_ context.Context, req SynthesizeRequest) (*SynthesizeResponse, error) {
	f.synthesizeCalls++
	f.lastSynthesizeReq = req
	if f.synthesizeErr != nil {
		return nil, f.synthesizeErr
	}
	if f.synthesizeResponse == nil {
		return &SynthesizeResponse{Audio: []byte("audio"), Format: "mp3"}, nil
	}
	return f.synthesizeResponse, nil
}

func TestNewAgentWithProvider_ValidatesInput(t *testing.T) {
	_, err := NewAgentWithProvider(nil, ProviderOpenAI, &fakeSpeechProvider{})
	if err == nil {
		t.Fatal("expected error for nil textAI")
	}

	_, err = NewAgentWithProvider(&fakeTextAI{}, ProviderOpenAI, nil)
	if err == nil {
		t.Fatal("expected error for nil speechProvider")
	}
}

func TestConverse_TranscribeTextAndSynthesize(t *testing.T) {
	textAI := &fakeTextAI{
		response: &models.AIChatResponse{AIResponse: "assistant reply"},
	}
	speech := &fakeSpeechProvider{
		transcribeResponse: &TranscribeResponse{Text: "hello from audio"},
		synthesizeResponse: &SynthesizeResponse{Audio: []byte("audio-bytes"), Format: "mp3"},
	}

	agent, err := NewAgentWithProvider(textAI, ProviderElevenLabs, speech)
	if err != nil {
		t.Fatalf("failed to create agent: %v", err)
	}

	now := time.Date(2026, 2, 18, 10, 0, 0, 0, time.UTC)
	agent.now = func() time.Time { return now }

	history := &models.AIChatHistory{}
	resp, err := agent.Converse(context.Background(), ConverseRequest{
		History: history,
		Audio:   []byte{0x1, 0x2},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if speech.transcribeCalls != 1 {
		t.Fatalf("expected 1 transcribe call, got %d", speech.transcribeCalls)
	}
	if speech.synthesizeCalls != 1 {
		t.Fatalf("expected 1 synthesize call, got %d", speech.synthesizeCalls)
	}
	if resp.Transcript != "hello from audio" {
		t.Fatalf("unexpected transcript: %s", resp.Transcript)
	}
	if string(resp.Audio) != "audio-bytes" {
		t.Fatalf("unexpected audio payload: %s", string(resp.Audio))
	}
	if resp.AudioFormat != "mp3" {
		t.Fatalf("unexpected audio format: %s", resp.AudioFormat)
	}
	if len(history.Messages) != 2 {
		t.Fatalf("expected 2 history messages, got %d", len(history.Messages))
	}
	if history.Messages[0].Role != models.User || history.Messages[0].Message != "hello from audio" {
		t.Fatalf("unexpected user message: %+v", history.Messages[0])
	}
	if history.Messages[1].Role != models.Assistant || history.Messages[1].Message != "assistant reply" {
		t.Fatalf("unexpected assistant message: %+v", history.Messages[1])
	}
	if len(textAI.history) != 1 {
		t.Fatalf("expected textAI to be called once, got %d", len(textAI.history))
	}
	if len(textAI.history[0].Messages) != 1 {
		t.Fatalf("expected textAI history to include the user message only")
	}
}

func TestConverse_UserTextSkipsTranscription(t *testing.T) {
	textAI := &fakeTextAI{
		response: &models.AIChatResponse{AIResponse: "assistant reply"},
	}
	speech := &fakeSpeechProvider{}
	agent, err := NewAgentWithProvider(textAI, ProviderOpenAI, speech)
	if err != nil {
		t.Fatalf("failed to create agent: %v", err)
	}

	_, err = agent.Converse(context.Background(), ConverseRequest{
		UserText: "direct text",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if speech.transcribeCalls != 0 {
		t.Fatalf("transcribe should not be called when user text is provided")
	}
	if speech.synthesizeCalls != 1 {
		t.Fatalf("expected one synthesize call")
	}
}

func TestConverse_SkipSynthesis(t *testing.T) {
	textAI := &fakeTextAI{
		response: &models.AIChatResponse{AIResponse: "assistant reply"},
	}
	speech := &fakeSpeechProvider{}
	agent, err := NewAgentWithProvider(textAI, ProviderOpenAI, speech)
	if err != nil {
		t.Fatalf("failed to create agent: %v", err)
	}

	resp, err := agent.Converse(context.Background(), ConverseRequest{
		UserText:      "hello",
		SkipSynthesis: true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.TextResponse == nil {
		t.Fatal("expected text response")
	}
	if speech.synthesizeCalls != 0 {
		t.Fatalf("synthesize should not be called when SkipSynthesis is true")
	}
}

func TestConverse_DisableTranscriptionRequiresText(t *testing.T) {
	textAI := &fakeTextAI{}
	speech := &fakeSpeechProvider{}
	agent, err := NewAgentWithProvider(textAI, ProviderOpenAI, speech)
	if err != nil {
		t.Fatalf("failed to create agent: %v", err)
	}

	_, err = agent.Converse(context.Background(), ConverseRequest{
		DisableTranscription: true,
	})
	if err == nil {
		t.Fatal("expected error when transcription is disabled without user text")
	}
}

func TestConverse_DisableTranscriptionUsesUserText(t *testing.T) {
	textAI := &fakeTextAI{
		response: &models.AIChatResponse{AIResponse: "assistant reply"},
	}
	speech := &fakeSpeechProvider{}
	agent, err := NewAgentWithProvider(textAI, ProviderOpenAI, speech)
	if err != nil {
		t.Fatalf("failed to create agent: %v", err)
	}

	resp, err := agent.Converse(context.Background(), ConverseRequest{
		UserText:             "typed input",
		DisableTranscription: true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Transcript != "typed input" {
		t.Fatalf("unexpected transcript: %q", resp.Transcript)
	}
	if speech.transcribeCalls != 0 {
		t.Fatalf("expected transcribe to be skipped")
	}
}

func TestConverse_DisableSynthesis(t *testing.T) {
	textAI := &fakeTextAI{
		response: &models.AIChatResponse{AIResponse: "assistant reply"},
	}
	speech := &fakeSpeechProvider{}
	agent, err := NewAgentWithProvider(textAI, ProviderOpenAI, speech)
	if err != nil {
		t.Fatalf("failed to create agent: %v", err)
	}

	resp, err := agent.Converse(context.Background(), ConverseRequest{
		UserText:         "hello",
		DisableSynthesis: true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.TextResponse == nil {
		t.Fatal("expected text response")
	}
	if speech.synthesizeCalls != 0 {
		t.Fatalf("synthesize should not be called when DisableSynthesis is true")
	}
}

func TestConverse_StripsThinkingTokensByDefault(t *testing.T) {
	textAI := &fakeTextAI{
		response: &models.AIChatResponse{
			AIResponse: "<think>internal chain</think>final answer",
		},
	}
	speech := &fakeSpeechProvider{}
	agent, err := NewAgentWithProvider(textAI, ProviderOpenAI, speech)
	if err != nil {
		t.Fatalf("failed to create agent: %v", err)
	}

	resp, err := agent.Converse(context.Background(), ConverseRequest{
		UserText: "hello",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.TextResponse == nil {
		t.Fatal("expected text response")
	}
	if resp.TextResponse.AIResponse != "final answer" {
		t.Fatalf("unexpected cleaned response: %q", resp.TextResponse.AIResponse)
	}
	if speech.lastSynthesizeReq.Text != "final answer" {
		t.Fatalf("expected cleaned text to be synthesized, got %q", speech.lastSynthesizeReq.Text)
	}
}

func TestConverse_CanSynthesizeThinkingTokens(t *testing.T) {
	textAI := &fakeTextAI{
		response: &models.AIChatResponse{
			AIResponse: "<think>internal chain</think>final answer",
		},
	}
	speech := &fakeSpeechProvider{}
	agent, err := NewAgentWithProvider(
		textAI,
		ProviderOpenAI,
		speech,
		WithSynthesizeThinkingTokens(true),
	)
	if err != nil {
		t.Fatalf("failed to create agent: %v", err)
	}

	resp, err := agent.Converse(context.Background(), ConverseRequest{
		UserText: "hello",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.TextResponse == nil {
		t.Fatal("expected text response")
	}
	if resp.TextResponse.AIResponse != "final answer" {
		t.Fatalf("unexpected cleaned response: %q", resp.TextResponse.AIResponse)
	}
	if speech.lastSynthesizeReq.Text != "<think>internal chain</think>final answer" {
		t.Fatalf("expected raw text to be synthesized, got %q", speech.lastSynthesizeReq.Text)
	}
}

func TestConverse_CanKeepThinkingTokensInText(t *testing.T) {
	textAI := &fakeTextAI{
		response: &models.AIChatResponse{
			AIResponse: "<think>internal chain</think>final answer",
		},
	}
	speech := &fakeSpeechProvider{}
	agent, err := NewAgentWithProvider(
		textAI,
		ProviderOpenAI,
		speech,
		WithStripThinkingTokens(false),
	)
	if err != nil {
		t.Fatalf("failed to create agent: %v", err)
	}

	resp, err := agent.Converse(context.Background(), ConverseRequest{
		UserText: "hello",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.TextResponse == nil {
		t.Fatal("expected text response")
	}
	if resp.TextResponse.AIResponse != "<think>internal chain</think>final answer" {
		t.Fatalf("expected raw response, got %q", resp.TextResponse.AIResponse)
	}
}

func TestConverse_ReturnsPartialOnTextAIError(t *testing.T) {
	textAI := &fakeTextAI{err: errors.New("text failure")}
	speech := &fakeSpeechProvider{transcribeResponse: &TranscribeResponse{Text: "hello"}}
	agent, err := NewAgentWithProvider(textAI, ProviderOpenAI, speech)
	if err != nil {
		t.Fatalf("failed to create agent: %v", err)
	}

	resp, err := agent.Converse(context.Background(), ConverseRequest{Audio: []byte{1}})
	if err == nil {
		t.Fatal("expected error")
	}
	if resp == nil {
		t.Fatal("expected partial response")
	}
	if resp.Transcript != "hello" {
		t.Fatalf("unexpected transcript: %s", resp.Transcript)
	}
}

func TestDefaultConfig_OpenAIKeyFallback(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "")
	t.Setenv("OPENAI_KEY", "legacy-openai-key")

	cfg := defaultConfig()
	if cfg.OpenAI.APIKey != "legacy-openai-key" {
		t.Fatalf("expected OPENAI_KEY fallback, got %q", cfg.OpenAI.APIKey)
	}
}
