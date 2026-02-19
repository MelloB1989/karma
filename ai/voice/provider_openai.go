package voice

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/openai/openai-go/v3"
	openaioption "github.com/openai/openai-go/v3/option"
)

type openAISpeechProvider struct {
	client openai.Client
	cfg    OpenAIConfig
}

func newOpenAISpeechProvider(cfg OpenAIConfig, httpClient *http.Client) (SpeechProvider, error) {
	if strings.TrimSpace(cfg.APIKey) == "" {
		return nil, errors.New("openai api key not found (set OPENAI_API_KEY or OPENAI_KEY)")
	}

	opts := []openaioption.RequestOption{openaioption.WithAPIKey(cfg.APIKey)}
	if cfg.BaseURL != "" {
		opts = append(opts, openaioption.WithBaseURL(cfg.BaseURL))
	}
	if httpClient != nil {
		opts = append(opts, openaioption.WithHTTPClient(httpClient))
	}

	return &openAISpeechProvider{
		client: openai.NewClient(opts...),
		cfg:    cfg,
	}, nil
}

func (p *openAISpeechProvider) Transcribe(ctx context.Context, req TranscribeRequest) (*TranscribeResponse, error) {
	if len(req.Audio) == 0 {
		return nil, errors.New("audio is required for transcription")
	}

	model, err := resolveProviderModel(ProviderOpenAI, ModelKindSTT, req.Model, p.cfg.STTModel)
	if err != nil {
		return nil, err
	}
	params := openai.AudioTranscriptionNewParams{
		File:  bytes.NewReader(req.Audio),
		Model: openai.AudioModel(model),
	}

	if req.Language != "" {
		params.Language = openai.String(req.Language)
	}
	if req.Prompt != "" {
		params.Prompt = openai.String(req.Prompt)
	}

	resp, err := p.client.Audio.Transcriptions.New(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("openai transcription failed: %w", err)
	}
	if resp == nil {
		return nil, errors.New("openai transcription returned empty response")
	}

	return &TranscribeResponse{
		Text: strings.TrimSpace(resp.Text),
		Raw:  resp,
	}, nil
}

func (p *openAISpeechProvider) Synthesize(ctx context.Context, req SynthesizeRequest) (*SynthesizeResponse, error) {
	text := strings.TrimSpace(req.Text)
	if text == "" {
		return nil, errors.New("text is required for synthesis")
	}

	model, err := resolveProviderModel(ProviderOpenAI, ModelKindTTS, req.Model, p.cfg.TTSModel)
	if err != nil {
		return nil, err
	}
	voice := firstNonEmpty(req.VoiceID, p.cfg.TTSVoice, "alloy")
	format := firstNonEmpty(req.Format, p.cfg.TTSFormat, "mp3")

	params := openai.AudioSpeechNewParams{
		Input:          text,
		Model:          openai.SpeechModel(model),
		Voice:          openai.AudioSpeechNewParamsVoice(voice),
		ResponseFormat: openai.AudioSpeechNewParamsResponseFormat(format),
	}

	if req.Instructions != "" {
		params.Instructions = openai.String(req.Instructions)
	}
	if req.Speed > 0 {
		params.Speed = openai.Float(req.Speed)
	}

	resp, err := p.client.Audio.Speech.New(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("openai synthesis failed: %w", err)
	}
	if resp == nil || resp.Body == nil {
		return nil, errors.New("openai synthesis returned empty response")
	}
	defer resp.Body.Close()

	audioBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed reading openai audio response: %w", err)
	}
	if len(audioBytes) == 0 {
		return nil, errors.New("openai synthesis returned empty audio")
	}

	return &SynthesizeResponse{
		Audio:  audioBytes,
		Format: format,
		Raw:    resp,
	}, nil
}
