package voice

import (
	"net/http"
	"os"
	"time"
)

func defaultConfig() Config {
	return Config{
		OpenAI: OpenAIConfig{
			APIKey:    firstNonEmpty(os.Getenv("OPENAI_API_KEY"), os.Getenv("OPENAI_KEY")),
			BaseURL:   os.Getenv("OPENAI_BASE_URL"),
			STTModel:  firstNonEmpty(os.Getenv("KARMA_VOICE_OPENAI_STT_MODEL"), "whisper-1"),
			TTSModel:  firstNonEmpty(os.Getenv("KARMA_VOICE_OPENAI_TTS_MODEL"), "gpt-4o-mini-tts"),
			TTSVoice:  firstNonEmpty(os.Getenv("KARMA_VOICE_OPENAI_TTS_VOICE"), "alloy"),
			TTSFormat: firstNonEmpty(os.Getenv("KARMA_VOICE_OPENAI_TTS_FORMAT"), "mp3"),
		},
		Together: TogetherConfig{
			APIKey:    os.Getenv("TOGETHER_API_KEY"),
			BaseURL:   os.Getenv("TOGETHER_BASE_URL"),
			STTModel:  firstNonEmpty(os.Getenv("KARMA_VOICE_TOGETHER_STT_MODEL"), "openai/whisper-large-v3"),
			TTSModel:  firstNonEmpty(os.Getenv("KARMA_VOICE_TOGETHER_TTS_MODEL"), "hexgrad/Kokoro-82M"),
			TTSVoice:  firstNonEmpty(os.Getenv("KARMA_VOICE_TOGETHER_TTS_VOICE"), "af_alloy"),
			TTSFormat: firstNonEmpty(os.Getenv("KARMA_VOICE_TOGETHER_TTS_FORMAT"), "mp3"),
		},
		ElevenLabs: ElevenLabsConfig{
			APIKey:                     os.Getenv("ELEVENLABS_API_KEY"),
			Token:                      os.Getenv("ELEVENLABS_TOKEN"),
			BaseWSURL:                  firstNonEmpty(os.Getenv("ELEVENLABS_WS_BASE_URL"), "wss://api.elevenlabs.io"),
			TTSModel:                   firstNonEmpty(os.Getenv("KARMA_VOICE_ELEVENLABS_TTS_MODEL"), "eleven_flash_v2_5"),
			TTSVoiceID:                 os.Getenv("ELEVENLABS_VOICE_ID"),
			TTSOutputFormat:            firstNonEmpty(os.Getenv("KARMA_VOICE_ELEVENLABS_TTS_OUTPUT_FORMAT"), "mp3_44100_128"),
			TTSLanguageCode:            os.Getenv("KARMA_VOICE_ELEVENLABS_TTS_LANGUAGE"),
			TTSInactivityTimeoutSecond: 20,
			STTModel:                   os.Getenv("KARMA_VOICE_ELEVENLABS_STT_MODEL"),
			STTAudioFormat:             firstNonEmpty(os.Getenv("KARMA_VOICE_ELEVENLABS_STT_AUDIO_FORMAT"), "pcm_16000"),
			STTCommitStrategy:          firstNonEmpty(os.Getenv("KARMA_VOICE_ELEVENLABS_STT_COMMIT_STRATEGY"), "manual"),
			STTLanguageCode:            os.Getenv("KARMA_VOICE_ELEVENLABS_STT_LANGUAGE"),
			ReadTimeout:                60 * time.Second,
		},
	}
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

// WithHTTPClient overrides the HTTP client used by provider SDKs.
func WithHTTPClient(client *http.Client) Option {
	return func(c *Config) {
		c.HTTPClient = client
	}
}

// WithOpenAIAPIKey overrides OpenAI API key.
func WithOpenAIAPIKey(apiKey string) Option {
	return func(c *Config) {
		c.OpenAI.APIKey = apiKey
	}
}

// WithTogetherAPIKey overrides Together API key.
func WithTogetherAPIKey(apiKey string) Option {
	return func(c *Config) {
		c.Together.APIKey = apiKey
	}
}

// WithElevenLabsAPIKey overrides ElevenLabs API key.
func WithElevenLabsAPIKey(apiKey string) Option {
	return func(c *Config) {
		c.ElevenLabs.APIKey = apiKey
	}
}

// WithOpenAIConfig merges OpenAI provider config.
func WithOpenAIConfig(cfg OpenAIConfig) Option {
	return func(c *Config) {
		if cfg.APIKey != "" {
			c.OpenAI.APIKey = cfg.APIKey
		}
		if cfg.BaseURL != "" {
			c.OpenAI.BaseURL = cfg.BaseURL
		}
		if cfg.STTModel != "" {
			c.OpenAI.STTModel = cfg.STTModel
		}
		if cfg.TTSModel != "" {
			c.OpenAI.TTSModel = cfg.TTSModel
		}
		if cfg.TTSVoice != "" {
			c.OpenAI.TTSVoice = cfg.TTSVoice
		}
		if cfg.TTSFormat != "" {
			c.OpenAI.TTSFormat = cfg.TTSFormat
		}
	}
}

// WithTogetherConfig merges Together provider config.
func WithTogetherConfig(cfg TogetherConfig) Option {
	return func(c *Config) {
		if cfg.APIKey != "" {
			c.Together.APIKey = cfg.APIKey
		}
		if cfg.BaseURL != "" {
			c.Together.BaseURL = cfg.BaseURL
		}
		if cfg.STTModel != "" {
			c.Together.STTModel = cfg.STTModel
		}
		if cfg.TTSModel != "" {
			c.Together.TTSModel = cfg.TTSModel
		}
		if cfg.TTSVoice != "" {
			c.Together.TTSVoice = cfg.TTSVoice
		}
		if cfg.TTSFormat != "" {
			c.Together.TTSFormat = cfg.TTSFormat
		}
	}
}

// WithElevenLabsConfig merges ElevenLabs provider config.
func WithElevenLabsConfig(cfg ElevenLabsConfig) Option {
	return func(c *Config) {
		if cfg.APIKey != "" {
			c.ElevenLabs.APIKey = cfg.APIKey
		}
		if cfg.Token != "" {
			c.ElevenLabs.Token = cfg.Token
		}
		if cfg.BaseWSURL != "" {
			c.ElevenLabs.BaseWSURL = cfg.BaseWSURL
		}
		if cfg.TTSModel != "" {
			c.ElevenLabs.TTSModel = cfg.TTSModel
		}
		if cfg.TTSVoiceID != "" {
			c.ElevenLabs.TTSVoiceID = cfg.TTSVoiceID
		}
		if cfg.TTSOutputFormat != "" {
			c.ElevenLabs.TTSOutputFormat = cfg.TTSOutputFormat
		}
		if cfg.TTSLanguageCode != "" {
			c.ElevenLabs.TTSLanguageCode = cfg.TTSLanguageCode
		}
		if cfg.TTSInactivityTimeoutSecond > 0 {
			c.ElevenLabs.TTSInactivityTimeoutSecond = cfg.TTSInactivityTimeoutSecond
		}
		if cfg.STTModel != "" {
			c.ElevenLabs.STTModel = cfg.STTModel
		}
		if cfg.STTAudioFormat != "" {
			c.ElevenLabs.STTAudioFormat = cfg.STTAudioFormat
		}
		if cfg.STTCommitStrategy != "" {
			c.ElevenLabs.STTCommitStrategy = cfg.STTCommitStrategy
		}
		if cfg.STTLanguageCode != "" {
			c.ElevenLabs.STTLanguageCode = cfg.STTLanguageCode
		}
		if cfg.IncludeTimestamps {
			c.ElevenLabs.IncludeTimestamps = true
		}
		if cfg.IncludeLanguageDetection {
			c.ElevenLabs.IncludeLanguageDetection = true
		}
		if cfg.ReadTimeout > 0 {
			c.ElevenLabs.ReadTimeout = cfg.ReadTimeout
		}
	}
}
