package voice

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
)

// Sarvam AI speech provider: saaras for transcription, bulbul for synthesis.
// Strong on Indian languages and code-mixed speech, which is exactly the
// audience the other providers serve least well.

const (
	SarvamSaarasV3 VoiceModel = "saaras:v3"
	SarvamBulbulV2 VoiceModel = "bulbul:v2"
	SarvamBulbulV3 VoiceModel = "bulbul:v3"
)

// bulbul:v3 accepts 2500 characters per request; longer text is chunked at
// sentence boundaries and the audio joined.
const sarvamTTSChunkLimit = 2400

type sarvamProvider struct {
	cfg    SarvamConfig
	client *http.Client
}

func newSarvamProvider(cfg SarvamConfig, client *http.Client) (*sarvamProvider, error) {
	if strings.TrimSpace(cfg.APIKey) == "" {
		return nil, fmt.Errorf("sarvam voice provider requires an API key (SARVAM_API_KEY)")
	}
	if cfg.BaseURL == "" {
		cfg.BaseURL = "https://api.sarvam.ai"
	}
	if cfg.STTModel == "" {
		cfg.STTModel = SarvamSaarasV3
	}
	if cfg.TTSModel == "" {
		cfg.TTSModel = SarvamBulbulV3
	}
	if cfg.TTSLanguage == "" {
		cfg.TTSLanguage = "en-IN"
	}
	if cfg.TTSCodec == "" {
		cfg.TTSCodec = "mp3"
	}
	if cfg.TTSSampleRate == 0 {
		cfg.TTSSampleRate = 24000
	}
	if client == nil {
		client = http.DefaultClient
	}
	return &sarvamProvider{cfg: cfg, client: client}, nil
}

func (p *sarvamProvider) Transcribe(ctx context.Context, req TranscribeRequest) (*TranscribeResponse, error) {
	model := req.Model
	if model == "" {
		model = p.cfg.STTModel
	}
	name := req.FileName
	if name == "" {
		name = "audio.wav"
	}

	var body bytes.Buffer
	form := multipart.NewWriter(&body)
	part, err := form.CreateFormFile("file", name)
	if err != nil {
		return nil, err
	}
	if _, err := part.Write(req.Audio); err != nil {
		return nil, err
	}
	_ = form.WriteField("model", string(model))
	if req.Language != "" {
		_ = form.WriteField("language_code", req.Language)
	}
	if err := form.Close(); err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.cfg.BaseURL+"/speech-to-text", &body)
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", form.FormDataContentType())
	httpReq.Header.Set("api-subscription-key", p.cfg.APIKey)

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("sarvam stt: %d: %s", resp.StatusCode, clip(raw, 300))
	}
	var parsed struct {
		Transcript string `json:"transcript"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, fmt.Errorf("sarvam stt: unparseable response: %w", err)
	}
	return &TranscribeResponse{Text: parsed.Transcript, Raw: json.RawMessage(raw)}, nil
}

func (p *sarvamProvider) Synthesize(ctx context.Context, req SynthesizeRequest) (*SynthesizeResponse, error) {
	model := req.Model
	if model == "" {
		model = p.cfg.TTSModel
	}
	language := req.Language
	if language == "" {
		language = p.cfg.TTSLanguage
	}
	speaker := req.VoiceID
	if speaker == "" {
		speaker = p.cfg.TTSSpeaker
	}
	format := req.Format
	if format == "" {
		format = p.cfg.TTSCodec
	}
	sampleRate := req.SampleRate
	if sampleRate == 0 {
		sampleRate = p.cfg.TTSSampleRate
	}

	var audio []byte
	for i, chunk := range splitForTTS(req.Text, sarvamTTSChunkLimit) {
		payload := map[string]any{
			"text":               chunk,
			"model":              string(model),
			"language_code":      language,
			"output_audio_codec": format,
			"speech_sample_rate": sampleRate,
		}
		if speaker != "" {
			payload["speaker"] = speaker
		}
		if req.Speed > 0 {
			payload["pace"] = req.Speed
		}
		encoded, err := json.Marshal(payload)
		if err != nil {
			return nil, err
		}

		httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.cfg.BaseURL+"/text-to-speech", bytes.NewReader(encoded))
		if err != nil {
			return nil, err
		}
		httpReq.Header.Set("Content-Type", "application/json")
		httpReq.Header.Set("api-subscription-key", p.cfg.APIKey)

		resp, err := p.client.Do(httpReq)
		if err != nil {
			return nil, err
		}
		raw, err := io.ReadAll(io.LimitReader(resp.Body, 32<<20))
		resp.Body.Close()
		if err != nil {
			return nil, err
		}
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("sarvam tts: %d: %s", resp.StatusCode, clip(raw, 300))
		}
		var parsed struct {
			Audios []string `json:"audios"`
		}
		if err := json.Unmarshal(raw, &parsed); err != nil || len(parsed.Audios) == 0 {
			return nil, fmt.Errorf("sarvam tts: unparseable response")
		}
		for _, b64 := range parsed.Audios {
			data, err := base64.StdEncoding.DecodeString(b64)
			if err != nil {
				return nil, fmt.Errorf("sarvam tts: bad audio payload: %w", err)
			}
			// MP3 frames concatenate legally; a repeated ID3 header does not.
			if i > 0 || len(audio) > 0 {
				data = stripID3(data)
			}
			audio = append(audio, data...)
		}
	}

	return &SynthesizeResponse{Audio: audio, Format: format}, nil
}

// splitForTTS cuts text into chunks under limit, preferring sentence ends.
func splitForTTS(text string, limit int) []string {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil
	}
	if len(text) <= limit {
		return []string{text}
	}
	var chunks []string
	for len(text) > limit {
		cut := strings.LastIndexAny(text[:limit], ".!?।\n")
		if cut < limit/2 {
			if sp := strings.LastIndex(text[:limit], " "); sp > limit/2 {
				cut = sp
			} else {
				cut = limit - 1
			}
		}
		chunks = append(chunks, strings.TrimSpace(text[:cut+1]))
		text = strings.TrimSpace(text[cut+1:])
	}
	if text != "" {
		chunks = append(chunks, text)
	}
	return chunks
}

// stripID3 removes a leading ID3v2 tag so concatenated MP3 chunks play clean.
func stripID3(data []byte) []byte {
	if len(data) < 10 || string(data[:3]) != "ID3" {
		return data
	}
	size := int(data[6]&0x7F)<<21 | int(data[7]&0x7F)<<14 | int(data[8]&0x7F)<<7 | int(data[9]&0x7F)
	if 10+size >= len(data) {
		return data
	}
	return data[10+size:]
}

func clip(b []byte, n int) string {
	if len(b) > n {
		return string(b[:n])
	}
	return string(b)
}
