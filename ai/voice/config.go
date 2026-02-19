package voice

import (
	"net/http"
	"time"

	"github.com/MelloB1989/karma/config"
)

func defaultConfig() Config {
	return Config{
		OpenAI: OpenAIConfig{
			APIKey: firstNonEmpty(config.GetEnvRaw("OPENAI_API_KEY"), config.GetEnvRaw("OPENAI_KEY")),
			// BaseURL:   config.GetEnvRaw("OPENAI_BASE_URL"),
			STTModel:  OpenAIWhisper1,
			TTSModel:  OpenAIGPT4oMiniTTS,
			TTSVoice:  firstNonEmpty(config.GetEnvRaw("KARMA_VOICE_OPENAI_TTS_VOICE"), "alloy"),
			TTSFormat: firstNonEmpty(config.GetEnvRaw("KARMA_VOICE_OPENAI_TTS_FORMAT"), "mp3"),
		},
		Together: TogetherConfig{
			APIKey: config.GetEnvRaw("TOGETHER_API_KEY"),
			// BaseURL:   config.GetEnvRaw("TOGETHER_BASE_URL"),
			STTModel:  TogetherWhisperLargeV3,
			TTSModel:  TogetherHexgradKokoro82M,
			TTSVoice:  firstNonEmpty(config.GetEnvRaw("KARMA_VOICE_TOGETHER_TTS_VOICE"), "af_alloy"),
			TTSFormat: firstNonEmpty(config.GetEnvRaw("KARMA_VOICE_TOGETHER_TTS_FORMAT"), "mp3"),
		},
		ElevenLabs: ElevenLabsConfig{
			APIKey:                     config.GetEnvRaw("ELEVENLABS_API_KEY"),
			Token:                      config.GetEnvRaw("ELEVENLABS_TOKEN"),
			BaseWSURL:                  firstNonEmpty(config.GetEnvRaw("ELEVENLABS_WS_BASE_URL"), "wss://api.elevenlabs.io"),
			TTSModel:                   ElevenLabsFlashV25,
			TTSVoiceID:                 config.GetEnvRaw("ELEVENLABS_VOICE_ID"),
			TTSOutputFormat:            firstNonEmpty(config.GetEnvRaw("KARMA_VOICE_ELEVENLABS_TTS_OUTPUT_FORMAT"), "mp3_44100_128"),
			TTSLanguageCode:            config.GetEnvRaw("KARMA_VOICE_ELEVENLABS_TTS_LANGUAGE"),
			TTSInactivityTimeoutSecond: 20,
			STTModel:                   ElevenLabsScribeV2,
			STTAudioFormat:             firstNonEmpty(config.GetEnvRaw("KARMA_VOICE_ELEVENLABS_STT_AUDIO_FORMAT"), "pcm_16000"),
			STTCommitStrategy:          firstNonEmpty(config.GetEnvRaw("KARMA_VOICE_ELEVENLABS_STT_COMMIT_STRATEGY"), "manual"),
			STTLanguageCode:            config.GetEnvRaw("KARMA_VOICE_ELEVENLABS_STT_LANGUAGE"),
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

// WithOpenAIModels overrides OpenAI STT and TTS models.
func WithOpenAIModels(sttModel VoiceModel, ttsModel VoiceModel) Option {
	return func(c *Config) {
		if sttModel != "" {
			c.OpenAI.STTModel = sttModel
		}
		if ttsModel != "" {
			c.OpenAI.TTSModel = ttsModel
		}
	}
}

// WithTogetherModels overrides Together STT and TTS models.
func WithTogetherModels(sttModel VoiceModel, ttsModel VoiceModel) Option {
	return func(c *Config) {
		if sttModel != "" {
			c.Together.STTModel = sttModel
		}
		if ttsModel != "" {
			c.Together.TTSModel = ttsModel
		}
	}
}

// WithElevenLabsModels overrides ElevenLabs STT and TTS models.
func WithElevenLabsModels(sttModel VoiceModel, ttsModel VoiceModel) Option {
	return func(c *Config) {
		if sttModel != "" {
			c.ElevenLabs.STTModel = sttModel
		}
		if ttsModel != "" {
			c.ElevenLabs.TTSModel = ttsModel
		}
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
