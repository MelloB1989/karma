package voice

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

type elevenLabsProvider struct {
	cfg    ElevenLabsConfig
	dialer *websocket.Dialer
}

func newElevenLabsProvider(cfg ElevenLabsConfig, _ *http.Client) (SpeechProvider, error) {
	if strings.TrimSpace(cfg.APIKey) == "" {
		return nil, errors.New("elevenlabs api key not found (set ELEVENLABS_API_KEY)")
	}

	if cfg.BaseWSURL == "" {
		cfg.BaseWSURL = "wss://api.elevenlabs.io"
	}
	if cfg.ReadTimeout <= 0 {
		cfg.ReadTimeout = 60 * time.Second
	}
	if cfg.TTSInactivityTimeoutSecond <= 0 {
		cfg.TTSInactivityTimeoutSecond = 20
	}

	d := *websocket.DefaultDialer

	return &elevenLabsProvider{
		cfg:    cfg,
		dialer: &d,
	}, nil
}

func (p *elevenLabsProvider) Transcribe(ctx context.Context, req TranscribeRequest) (*TranscribeResponse, error) {
	if len(req.Audio) == 0 {
		return nil, errors.New("audio is required for transcription")
	}

	model, err := resolveProviderModel(ProviderElevenLabs, ModelKindSTT, req.Model, p.cfg.STTModel)
	if err != nil {
		return nil, err
	}

	query := url.Values{}
	if model != "" {
		query.Set("model_id", model)
	}
	if language := firstNonEmpty(req.Language, p.cfg.STTLanguageCode); language != "" {
		query.Set("language_code", language)
	}
	if format := firstNonEmpty(req.AudioFormat, p.cfg.STTAudioFormat); format != "" {
		query.Set("audio_format", format)
	}
	if strategy := firstNonEmpty(p.cfg.STTCommitStrategy, "manual"); strategy != "" {
		query.Set("commit_strategy", strategy)
	}
	if p.cfg.IncludeTimestamps {
		query.Set("include_timestamps", "true")
	}
	if p.cfg.IncludeLanguageDetection {
		query.Set("include_language_detection", "true")
	}
	if p.cfg.Token != "" {
		query.Set("token", p.cfg.Token)
	}

	wsURL, err := p.buildWSURL("/v1/speech-to-text/realtime", query)
	if err != nil {
		return nil, err
	}

	headers := make(http.Header)
	headers.Set("xi-api-key", p.cfg.APIKey)

	conn, _, err := p.dialer.DialContext(ctx, wsURL, headers)
	if err != nil {
		return nil, fmt.Errorf("elevenlabs stt websocket dial failed: %w", err)
	}
	defer conn.Close()

	sampleRate := req.SampleRate
	if sampleRate <= 0 {
		sampleRate = 16000
	}

	message := map[string]any{
		"message_type":  "input_audio_chunk",
		"audio_base_64": base64.StdEncoding.EncodeToString(req.Audio),
		"commit":        true,
		"sample_rate":   sampleRate,
	}
	if req.PreviousText != "" {
		message["previous_text"] = req.PreviousText
	}

	if err := conn.WriteJSON(message); err != nil {
		return nil, fmt.Errorf("elevenlabs stt websocket write failed: %w", err)
	}

	latestPartial := ""
	for {
		if err := p.setReadDeadline(ctx, conn); err != nil {
			return nil, err
		}

		_, data, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
				break
			}
			if latestPartial != "" {
				return &TranscribeResponse{Text: latestPartial}, nil
			}
			return nil, fmt.Errorf("elevenlabs stt websocket read failed: %w", err)
		}

		payload := map[string]any{}
		if err := json.Unmarshal(data, &payload); err != nil {
			continue
		}

		messageType := asString(payload["message_type"])
		switch messageType {
		case "partial_transcript":
			if text := strings.TrimSpace(asString(payload["text"])); text != "" {
				latestPartial = text
			}
		case "committed_transcript", "committed_transcript_with_timestamps":
			text := strings.TrimSpace(asString(payload["text"]))
			if text == "" {
				text = latestPartial
			}
			if text == "" {
				return nil, errors.New("elevenlabs transcription committed with empty text")
			}
			return &TranscribeResponse{Text: text, Raw: json.RawMessage(data)}, nil
		case "error", "auth_error", "quota_exceeded", "commit_throttled", "unaccepted_terms", "rate_limited", "queue_overflow", "resource_exhausted", "session_time_limit_exceeded", "input_error", "chunk_size_exceeded", "insufficient_audio_activity", "transcriber_error":
			return nil, elevenLabsPayloadError(messageType, payload)
		}
	}

	if latestPartial != "" {
		return &TranscribeResponse{Text: latestPartial}, nil
	}

	return nil, errors.New("no transcription received from elevenlabs")
}

func (p *elevenLabsProvider) Synthesize(ctx context.Context, req SynthesizeRequest) (*SynthesizeResponse, error) {
	text := strings.TrimSpace(req.Text)
	if text == "" {
		return nil, errors.New("text is required for synthesis")
	}

	voiceID := firstNonEmpty(req.VoiceID, p.cfg.TTSVoiceID)
	if voiceID == "" {
		return nil, errors.New("voice_id is required for elevenlabs tts (set SynthesizeRequest.VoiceID or ELEVENLABS_VOICE_ID)")
	}

	model, err := resolveProviderModel(ProviderElevenLabs, ModelKindTTS, req.Model, p.cfg.TTSModel)
	if err != nil {
		return nil, err
	}

	format := firstNonEmpty(req.Format, p.cfg.TTSOutputFormat, "mp3_44100_128")
	query := url.Values{}
	if model != "" {
		query.Set("model_id", model)
	}
	if language := firstNonEmpty(req.Language, p.cfg.TTSLanguageCode); language != "" {
		query.Set("language_code", language)
	}
	if format != "" {
		query.Set("output_format", format)
	}
	if p.cfg.TTSInactivityTimeoutSecond > 0 {
		query.Set("inactivity_timeout", strconv.Itoa(p.cfg.TTSInactivityTimeoutSecond))
	}

	path := fmt.Sprintf("/v1/text-to-speech/%s/stream-input", url.PathEscape(voiceID))
	wsURL, err := p.buildWSURL(path, query)
	if err != nil {
		return nil, err
	}

	headers := make(http.Header)
	headers.Set("xi-api-key", p.cfg.APIKey)

	conn, _, err := p.dialer.DialContext(ctx, wsURL, headers)
	if err != nil {
		return nil, fmt.Errorf("elevenlabs tts websocket dial failed: %w", err)
	}
	defer conn.Close()

	// ElevenLabs websocket requires a blank initial text message.
	if err := conn.WriteJSON(map[string]any{"text": " "}); err != nil {
		return nil, fmt.Errorf("elevenlabs tts init failed: %w", err)
	}

	streamText := ensureTrailingSpace(text)
	if err := conn.WriteJSON(map[string]any{"text": streamText, "flush": true}); err != nil {
		return nil, fmt.Errorf("elevenlabs tts text send failed: %w", err)
	}

	if err := conn.WriteJSON(map[string]any{"text": ""}); err != nil {
		return nil, fmt.Errorf("elevenlabs tts close send failed: %w", err)
	}

	var audio bytes.Buffer
	for {
		if err := p.setReadDeadline(ctx, conn); err != nil {
			return nil, err
		}

		_, data, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
				break
			}
			return nil, fmt.Errorf("elevenlabs tts websocket read failed: %w", err)
		}

		payload := map[string]json.RawMessage{}
		if err := json.Unmarshal(data, &payload); err != nil {
			continue
		}

		if rawErr, ok := payload["error"]; ok {
			var errText string
			_ = json.Unmarshal(rawErr, &errText)
			if strings.TrimSpace(errText) == "" {
				errText = "elevenlabs tts error"
			}
			return nil, errors.New(errText)
		}

		if rawAudio, ok := payload["audio"]; ok {
			var encoded string
			if err := json.Unmarshal(rawAudio, &encoded); err == nil && encoded != "" {
				chunk, decodeErr := base64.StdEncoding.DecodeString(encoded)
				if decodeErr != nil {
					return nil, fmt.Errorf("elevenlabs audio chunk decode failed: %w", decodeErr)
				}
				audio.Write(chunk)
			}
		}

		if rawFinal, ok := payload["isFinal"]; ok {
			var final bool
			if err := json.Unmarshal(rawFinal, &final); err == nil {
				if final {
					break
				}
			} else {
				var finalText string
				if err := json.Unmarshal(rawFinal, &finalText); err == nil && strings.EqualFold(finalText, "true") {
					break
				}
			}
		}
	}

	if audio.Len() == 0 {
		return nil, errors.New("elevenlabs synthesis returned empty audio")
	}

	return &SynthesizeResponse{
		Audio:  audio.Bytes(),
		Format: format,
	}, nil
}

func (p *elevenLabsProvider) buildWSURL(path string, query url.Values) (string, error) {
	base := strings.TrimRight(p.cfg.BaseWSURL, "/")
	if !strings.HasPrefix(base, "ws://") && !strings.HasPrefix(base, "wss://") {
		return "", fmt.Errorf("invalid elevenlabs websocket base url: %s", p.cfg.BaseWSURL)
	}

	u, err := url.Parse(base)
	if err != nil {
		return "", fmt.Errorf("invalid elevenlabs websocket base url: %w", err)
	}

	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	u.Path = strings.TrimRight(u.Path, "/") + path
	u.RawQuery = query.Encode()

	return u.String(), nil
}

func (p *elevenLabsProvider) setReadDeadline(ctx context.Context, conn *websocket.Conn) error {
	if conn == nil {
		return errors.New("nil websocket connection")
	}

	timeout := p.cfg.ReadTimeout
	if timeout <= 0 {
		timeout = 60 * time.Second
	}
	deadline := time.Now().Add(timeout)

	if ctxDeadline, ok := ctx.Deadline(); ok && ctxDeadline.Before(deadline) {
		deadline = ctxDeadline
	}

	if err := conn.SetReadDeadline(deadline); err != nil {
		return fmt.Errorf("unable to set websocket read deadline: %w", err)
	}

	return nil
}

func elevenLabsPayloadError(messageType string, payload map[string]any) error {
	msg := strings.TrimSpace(asString(payload["error"]))
	if msg == "" {
		msg = "unknown elevenlabs error"
	}
	return fmt.Errorf("elevenlabs %s: %s", messageType, msg)
}

func asString(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case fmt.Stringer:
		return t.String()
	default:
		return ""
	}
}

func ensureTrailingSpace(s string) string {
	if strings.HasSuffix(s, " ") {
		return s
	}
	return s + " "
}
