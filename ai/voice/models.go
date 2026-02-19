package voice

import (
	"fmt"
	"sort"
	"strings"
)

// VoiceModel represents a model identifier for STT or TTS.
type VoiceModel string

// ModelKind identifies speech direction.
type ModelKind string

const (
	ModelKindSTT ModelKind = "stt"
	ModelKindTTS ModelKind = "tts"
)

const (
	// OpenAI STT
	OpenAIWhisper1            VoiceModel = "whisper-1"
	OpenAIGPT4oTranscribe     VoiceModel = "gpt-4o-transcribe"
	OpenAIGPT4oMiniTranscribe VoiceModel = "gpt-4o-mini-transcribe"
	OpenAIGPT4oTranscribeDia  VoiceModel = "gpt-4o-transcribe-diarize"

	// OpenAI TTS
	OpenAITTS1         VoiceModel = "tts-1"
	OpenAITTS1HD       VoiceModel = "tts-1-hd"
	OpenAIGPT4oMiniTTS VoiceModel = "gpt-4o-mini-tts"

	// Together STT
	TogetherWhisperLargeV3 VoiceModel = "openai/whisper-large-v3"

	// Together TTS
	TogetherCartesiaSonic       VoiceModel = "cartesia/sonic"
	TogetherHexgradKokoro82M    VoiceModel = "hexgrad/Kokoro-82M"
	TogetherCanopyOrpheus3B01FT VoiceModel = "canopylabs/orpheus-3b-0.1-ft"

	// ElevenLabs STT
	ElevenLabsScribeV1 VoiceModel = "scribe_v1"

	// ElevenLabs TTS
	ElevenLabsFlashV25       VoiceModel = "eleven_flash_v2_5"
	ElevenLabsTurboV25       VoiceModel = "eleven_turbo_v2_5"
	ElevenLabsMultilingualV2 VoiceModel = "eleven_multilingual_v2"
)

// ProviderSTTModelMapping maps provider + model constant to the API model string.
var ProviderSTTModelMapping = map[Provider]map[VoiceModel]string{
	ProviderOpenAI: {
		OpenAIWhisper1:            string(OpenAIWhisper1),
		OpenAIGPT4oTranscribe:     string(OpenAIGPT4oTranscribe),
		OpenAIGPT4oMiniTranscribe: string(OpenAIGPT4oMiniTranscribe),
		OpenAIGPT4oTranscribeDia:  string(OpenAIGPT4oTranscribeDia),
	},
	ProviderTogether: {
		TogetherWhisperLargeV3: string(TogetherWhisperLargeV3),
	},
	ProviderElevenLabs: {
		ElevenLabsScribeV1: string(ElevenLabsScribeV1),
	},
}

// ProviderTTSModelMapping maps provider + model constant to the API model string.
var ProviderTTSModelMapping = map[Provider]map[VoiceModel]string{
	ProviderOpenAI: {
		OpenAITTS1:         string(OpenAITTS1),
		OpenAITTS1HD:       string(OpenAITTS1HD),
		OpenAIGPT4oMiniTTS: string(OpenAIGPT4oMiniTTS),
	},
	ProviderTogether: {
		TogetherCartesiaSonic:       string(TogetherCartesiaSonic),
		TogetherHexgradKokoro82M:    string(TogetherHexgradKokoro82M),
		TogetherCanopyOrpheus3B01FT: string(TogetherCanopyOrpheus3B01FT),
	},
	ProviderElevenLabs: {
		ElevenLabsFlashV25:       string(ElevenLabsFlashV25),
		ElevenLabsTurboV25:       string(ElevenLabsTurboV25),
		ElevenLabsMultilingualV2: string(ElevenLabsMultilingualV2),
	},
}

// GetAvailableSTTModels returns supported STT models for a provider.
func GetAvailableSTTModels(provider Provider) []VoiceModel {
	return getAvailableModels(ProviderSTTModelMapping, provider)
}

// GetAvailableTTSModels returns supported TTS models for a provider.
func GetAvailableTTSModels(provider Provider) []VoiceModel {
	return getAvailableModels(ProviderTTSModelMapping, provider)
}

func getAvailableModels(mapping map[Provider]map[VoiceModel]string, provider Provider) []VoiceModel {
	providerModels := mapping[provider]
	if len(providerModels) == 0 {
		return nil
	}

	models := make([]VoiceModel, 0, len(providerModels))
	for model := range providerModels {
		models = append(models, model)
	}
	sort.Slice(models, func(i, j int) bool {
		return models[i] < models[j]
	})
	return models
}

func resolveProviderModel(provider Provider, kind ModelKind, preferred VoiceModel, fallback VoiceModel) (string, error) {
	selected := preferred
	if selected == "" {
		selected = fallback
	}
	if selected == "" {
		return "", fmt.Errorf("no %s model configured for provider %s", kind, provider)
	}

	mapping := ProviderSTTModelMapping
	if kind == ModelKindTTS {
		mapping = ProviderTTSModelMapping
	}

	providerModels, ok := mapping[provider]
	if !ok || len(providerModels) == 0 {
		return "", fmt.Errorf("provider %s does not support %s models", provider, kind)
	}

	if canonical, exists := providerModels[selected]; exists {
		return canonical, nil
	}

	available := getAvailableModels(mapping, provider)
	availableText := make([]string, 0, len(available))
	for _, model := range available {
		availableText = append(availableText, string(model))
	}

	return "", fmt.Errorf(
		"unsupported %s model %q for provider %s; available models: %s",
		kind,
		selected,
		provider,
		strings.Join(availableText, ", "),
	)
}
