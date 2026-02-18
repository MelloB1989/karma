package voice

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	together "github.com/togethercomputer/together-go"
	togetheroption "github.com/togethercomputer/together-go/option"
)

type togetherSpeechProvider struct {
	client together.Client
	cfg    TogetherConfig
}

func newTogetherSpeechProvider(cfg TogetherConfig, httpClient *http.Client) (SpeechProvider, error) {
	if strings.TrimSpace(cfg.APIKey) == "" {
		return nil, errors.New("together api key not found (set TOGETHER_API_KEY)")
	}

	opts := []togetheroption.RequestOption{togetheroption.WithAPIKey(cfg.APIKey)}
	if cfg.BaseURL != "" {
		opts = append(opts, togetheroption.WithBaseURL(cfg.BaseURL))
	}
	if httpClient != nil {
		opts = append(opts, togetheroption.WithHTTPClient(httpClient))
	}

	return &togetherSpeechProvider{
		client: together.NewClient(opts...),
		cfg:    cfg,
	}, nil
}

func (p *togetherSpeechProvider) Transcribe(ctx context.Context, req TranscribeRequest) (*TranscribeResponse, error) {
	if len(req.Audio) == 0 {
		return nil, errors.New("audio is required for transcription")
	}

	model := firstNonEmpty(req.Model, p.cfg.STTModel, "openai/whisper-large-v3")
	params := together.AudioTranscriptionNewParams{
		File: together.AudioTranscriptionNewParamsFileUnion{
			OfFile: bytes.NewReader(req.Audio),
		},
		Model: together.AudioTranscriptionNewParamsModel(model),
	}

	if req.Language != "" {
		params.Language = together.String(req.Language)
	}
	if req.Prompt != "" {
		params.Prompt = together.String(req.Prompt)
	}

	resp, err := p.client.Audio.Transcriptions.New(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("together transcription failed: %w", err)
	}
	if resp == nil {
		return nil, errors.New("together transcription returned empty response")
	}

	return &TranscribeResponse{
		Text: strings.TrimSpace(resp.Text),
		Raw:  resp,
	}, nil
}

func (p *togetherSpeechProvider) Synthesize(ctx context.Context, req SynthesizeRequest) (*SynthesizeResponse, error) {
	text := strings.TrimSpace(req.Text)
	if text == "" {
		return nil, errors.New("text is required for synthesis")
	}

	model := firstNonEmpty(req.Model, p.cfg.TTSModel, "hexgrad/Kokoro-82M")
	voice := firstNonEmpty(req.VoiceID, p.cfg.TTSVoice, "af_alloy")
	format := firstNonEmpty(req.Format, p.cfg.TTSFormat, "mp3")

	params := together.AudioSpeechNewParams{
		Input:          text,
		Model:          together.AudioSpeechNewParamsModel(model),
		Voice:          voice,
		ResponseFormat: together.AudioSpeechNewParamsResponseFormat(format),
	}

	if req.SampleRate > 0 {
		params.SampleRate = together.Int(int64(req.SampleRate))
	}
	if req.Language != "" {
		params.Language = together.AudioSpeechNewParamsLanguage(req.Language)
	}

	resp, err := p.client.Audio.Speech.New(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("together synthesis failed: %w", err)
	}
	if resp == nil || resp.Body == nil {
		return nil, errors.New("together synthesis returned empty response")
	}
	defer resp.Body.Close()

	audioBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed reading together audio response: %w", err)
	}
	if len(audioBytes) == 0 {
		return nil, errors.New("together synthesis returned empty audio")
	}

	return &SynthesizeResponse{
		Audio:  audioBytes,
		Format: format,
		Raw:    resp,
	}, nil
}
