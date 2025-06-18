package ai

import (
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
)

type Models string

const (
	// OpenAI Models
	ChatModelO1                              Models = "o1"
	ChatModelO1_2024_12_17                   Models = "o1-2024-12-17"
	ChatModelO1Preview                       Models = "o1-preview"
	ChatModelO1Preview2024_09_12             Models = "o1-preview-2024-09-12"
	ChatModelO1Mini                          Models = "o1-mini"
	ChatModelO1Mini2024_09_12                Models = "o1-mini-2024-09-12"
	ChatModelGPT4o                           Models = "gpt-4o"
	ChatModelGPT4o2024_11_20                 Models = "gpt-4o-2024-11-20"
	ChatModelGPT4o2024_08_06                 Models = "gpt-4o-2024-08-06"
	ChatModelGPT4o2024_05_13                 Models = "gpt-4o-2024-05-13"
	ChatModelGPT4oAudioPreview               Models = "gpt-4o-audio-preview"
	ChatModelGPT4oAudioPreview2024_10_01     Models = "gpt-4o-audio-preview-2024-10-01"
	ChatModelGPT4oAudioPreview2024_12_17     Models = "gpt-4o-audio-preview-2024-12-17"
	ChatModelGPT4oMiniAudioPreview           Models = "gpt-4o-mini-audio-preview"
	ChatModelGPT4oMiniAudioPreview2024_12_17 Models = "gpt-4o-mini-audio-preview-2024-12-17"
	ChatModelChatgpt4oLatest                 Models = "chatgpt-4o-latest"
	ChatModelGPT4oMini                       Models = "gpt-4o-mini"
	ChatModelGPT4oMini2024_07_18             Models = "gpt-4o-mini-2024-07-18"
	ChatModelGPT4Turbo                       Models = "gpt-4-turbo"
	ChatModelGPT4Turbo2024_04_09             Models = "gpt-4-turbo-2024-04-09"
	ChatModelGPT4_0125Preview                Models = "gpt-4-0125-preview"
	ChatModelGPT4TurboPreview                Models = "gpt-4-turbo-preview"
	ChatModelGPT4_1106Preview                Models = "gpt-4-1106-preview"
	ChatModelGPT4VisionPreview               Models = "gpt-4-vision-preview"
	ChatModelGPT4                            Models = "gpt-4"
	ChatModelGPT4_0314                       Models = "gpt-4-0314"
	ChatModelGPT4_0613                       Models = "gpt-4-0613"
	ChatModelGPT4_32k                        Models = "gpt-4-32k"
	ChatModelGPT4_32k0314                    Models = "gpt-4-32k-0314"
	ChatModelGPT4_32k0613                    Models = "gpt-4-32k-0613"
	ChatModelGPT3_5Turbo                     Models = "gpt-3.5-turbo"
	ChatModelGPT3_5Turbo16k                  Models = "gpt-3.5-turbo-16k"
	ChatModelGPT3_5Turbo0301                 Models = "gpt-3.5-turbo-0301"
	ChatModelGPT3_5Turbo0613                 Models = "gpt-3.5-turbo-0613"
	ChatModelGPT3_5Turbo1106                 Models = "gpt-3.5-turbo-1106"
	ChatModelGPT3_5Turbo0125                 Models = "gpt-3.5-turbo-0125"
	ChatModelGPT3_5Turbo16k0613              Models = "gpt-3.5-turbo-16k-0613"

	// Anthropic Models For BEDROCK
	ClaudeInstantV1_2_100K        Models = "anthropic.claude-instant-v1:2:100k"
	ClaudeInstantV1               Models = "anthropic.claude-instant-v1"
	ClaudeV2_0_18K                Models = "anthropic.claude-v2:0:18k"
	ClaudeV2_0_100K               Models = "anthropic.claude-v2:0:100k"
	ClaudeV2_1_18K                Models = "anthropic.claude-v2:1:18k"
	ClaudeV2_1_200K               Models = "anthropic.claude-v2:1:200k"
	ClaudeV2_1                    Models = "anthropic.claude-v2:1"
	ClaudeV2                      Models = "anthropic.claude-v2"
	Claude3Sonnet20240229V1_28K   Models = "anthropic.claude-3-sonnet-20240229-v1:0:28k"
	Claude3Sonnet20240229V1_200K  Models = "anthropic.claude-3-sonnet-20240229-v1:0:200k"
	Claude3Sonnet20240229V1       Models = "anthropic.claude-3-sonnet-20240229-v1:0"
	Claude3Haiku20240307V1_48K    Models = "anthropic.claude-3-haiku-20240307-v1:0:48k"
	Claude3Haiku20240307V1_200K   Models = "anthropic.claude-3-haiku-20240307-v1:0:200k"
	Claude3Haiku20240307V1        Models = "anthropic.claude-3-haiku-20240307-v1:0"
	Claude3Opus20240229V1_12K     Models = "anthropic.claude-3-opus-20240229-v1:0:12k"
	Claude3Opus20240229V1_28K     Models = "anthropic.claude-3-opus-20240229-v1:0:28k"
	Claude3Opus20240229V1_200K    Models = "anthropic.claude-3-opus-20240229-v1:0:200k"
	Claude3Opus20240229V1         Models = "anthropic.claude-3-opus-20240229-v1:0"
	Claude3_5Sonnet20240620V1     Models = "anthropic.claude-3-5-sonnet-20240620-v1:0"
	Claude3_5Sonnet20241022V2     Models = "anthropic.claude-3-5-sonnet-20241022-v2:0"
	Claude3_5Haiku20241022V1      Models = "anthropic.claude-3-5-haiku-20241022-v1:0"
	Claude3_7Sonnet20250219V1     Models = "us.anthropic.claude-3-7-sonnet-20250219-v1:0"
	ApacClaude3_5Sonnet20240620V1 Models = "apac.anthropic.claude-3-5-sonnet-20240620-v1:0"

	// Meta's LLaMA Models
	Llama3_8B    Models = "meta.llama3-8b-instruct-v1:0"
	Llama3_70B   Models = "meta.llama3-70b-instruct-v1:0"
	Llama3_1_8B  Models = "meta.llama3-1-8b-instruct-v1:0"
	Llama3_1_70B Models = "meta.llama3-1-70b-instruct-v1:0"
	Llama3_2_11B Models = "meta.llama3-2-11b-instruct-v1:0"
	Llama3_2_90B Models = "meta.llama3-2-90b-instruct-v1:0"
	Llama3_2_1B  Models = "meta.llama3-2-1b-instruct-v1:0"
	Llama3_2_3B  Models = "meta.llama3-2-3b-instruct-v1:0"
	Llama3_3_70B Models = "meta.llama3-3-70b-instruct-v1:0"

	// AWS US LLaMA Models
	US_Llama3_3_70B Models = "us.meta.llama3-3-70b-instruct-v1:0"

	// Mistral AI Models
	Mistral7BInstructV0   Models = "mistral.mistral-7b-instruct-v0:2"
	Mixtral8x7BInstructV0 Models = "mistral.mixtral-8x7b-instruct-v0:1"
	MistralLarge2402V1    Models = "mistral.mistral-large-2402-v1:0"
	MistralSmall2402V1    Models = "mistral.mistral-small-2402-v1:0"

	// Titan Models
	TitanTG1Large         Models = "amazon.titan-tg1-large"
	TitanImageGeneratorV1 Models = "amazon.titan-image-generator-v1:0"
	TitanImageGeneratorV2 Models = "amazon.titan-image-generator-v2:0"
	TitanTextPremierV1    Models = "amazon.titan-text-premier-v1:0"
	TitanEmbedG1Text02    Models = "amazon.titan-embed-g1-text-02"
	TitanTextLiteV1_4K    Models = "amazon.titan-text-lite-v1:0:4k"
	TitanTextLiteV1       Models = "amazon.titan-text-lite-v1"
	TitanTextExpressV1_8K Models = "amazon.titan-text-express-v1:0:8k"
	TitanTextExpressV1    Models = "amazon.titan-text-express-v1"
	TitanEmbedTextV1_8K   Models = "amazon.titan-embed-text-v1:2:8k"
	TitanEmbedTextV1      Models = "amazon.titan-embed-text-v1"
	TitanEmbedTextV2_8K   Models = "amazon.titan-embed-text-v2:0:8k"
	TitanEmbedTextV2      Models = "amazon.titan-embed-text-v2:0"
	TitanEmbedImageV1     Models = "amazon.titan-embed-image-v1:0"

	// Nova Models
	NovaProV1_300K   Models = "amazon.nova-pro-v1:0:300k"
	NovaProV1        Models = "amazon.nova-pro-v1:0"
	NovaLiteV1_300K  Models = "amazon.nova-lite-v1:0:300k"
	NovaLiteV1       Models = "amazon.nova-lite-v1:0"
	NovaCanvasV1     Models = "amazon.nova-canvas-v1:0"
	NovaReelV1       Models = "amazon.nova-reel-v1:0"
	NovaMicroV1_128K Models = "amazon.nova-micro-v1:0:128k"
	NovaMicroV1      Models = "amazon.nova-micro-v1:0"

	// Stable Diffusion Model
	StableDiffusionXLV1 Models = "stability.stable-diffusion-xl-v1:0"

	// Google Models
	PaLM2                          Models = "palm-2"
	Gemini_2_5_Flash_Preview_04_17 Models = "gemini-2.5-flash-preview-04-17"
	Gemini25ProPreview             Models = "gemini-2.5-pro-preview-05-06"
	Gemini20Flash                  Models = "gemini-2.0-flash"
	Gemini20FlashPreviewImageGen   Models = "gemini-2.0-flash-preview-image-generation"
	Gemini20FlashLite              Models = "gemini-2.0-flash-lite"
	Gemini15Flash                  Models = "gemini-1.5-flash"
	Gemini15Flash8B                Models = "gemini-1.5-flash-8b"
	Gemini15Pro                    Models = "gemini-1.5-pro"
	GeminiEmbedding                Models = "gemini-embedding-exp"
	Gemini20FlashLive              Models = "gemini-2.0-flash-live-001"

	// Anthropic Models without BEDROCK
	ModelClaude3_7SonnetLatest      Models = "claude-3-7-sonnet-latest"
	ModelClaude3_7Sonnet20250219    Models = "claude-3-7-sonnet-20250219"
	ModelClaude3_5HaikuLatest       Models = "claude-3-5-haiku-latest"
	ModelClaude3_5Haiku20241022     Models = "claude-3-5-haiku-20241022"
	ModelClaudeSonnet4_20250514     Models = "claude-sonnet-4-20250514"
	ModelClaudeSonnet4_0            Models = "claude-sonnet-4-0"
	ModelClaude4Sonnet20250514      Models = "claude-4-sonnet-20250514"
	ModelClaude3_5SonnetLatest      Models = "claude-3-5-sonnet-latest"
	ModelClaude3_5Sonnet20241022    Models = "claude-3-5-sonnet-20241022"
	ModelClaude_3_5_Sonnet_20240620 Models = "claude-3-5-sonnet-20240620"
	ModelClaudeOpus4_0              Models = "claude-opus-4-0"
	ModelClaudeOpus4_20250514       Models = "claude-opus-4-20250514"
	ModelClaude4Opus20250514        Models = "claude-4-opus-20250514"
	ModelClaude3OpusLatest          Models = "claude-3-opus-latest"
	ModelClaude_3_Opus_20240229     Models = "claude-3-opus-20240229"
	// Deprecated: Will reach end-of-life on July 21st, 2025. Please migrate to a newer
	// model. Visit https://docs.anthropic.com/en/docs/resources/model-deprecations for
	// more information.
	ModelClaude_3_Sonnet_20240229 Models = "claude-3-sonnet-20240229"
	ModelClaude_3_Haiku_20240307  Models = "claude-3-haiku-20240307"
	// Deprecated: Will reach end-of-life on July 21st, 2025. Please migrate to a newer
	// model. Visit https://docs.anthropic.com/en/docs/resources/model-deprecations for
	// more information.
	ModelClaude_2_1 Models = "claude-2.1"
	// Deprecated: Will reach end-of-life on July 21st, 2025. Please migrate to a newer
	// model. Visit https://docs.anthropic.com/en/docs/resources/model-deprecations for
	// more information.
	ModelClaude_2_0 Models = "claude-2.0"
)

type ModelProviders string

const (
	OpenAI     ModelProviders = "OpenAI"
	Bedrock    ModelProviders = "Bedrock" // Provider for Amazon, Meta, Mistral, Stability, AI21, Cohere
	Anthropic  ModelProviders = "Anthropic"
	Meta       ModelProviders = "Meta"
	MistralAI  ModelProviders = "Mistral AI"
	Google     ModelProviders = "Google"
	Cohere     ModelProviders = "Cohere"
	OpenSource ModelProviders = "Open Source"
	Undefined  ModelProviders = "Undefined"
)

// IsOpenAIModel checks if the model is from OpenAI
func (m Models) IsOpenAIModel() bool {
	// Check for GPT prefixes and O1 prefixes
	return strings.HasPrefix(string(m), "gpt-") ||
		strings.HasPrefix(string(m), "o1") ||
		strings.HasPrefix(string(m), "chatgpt-")
}

// IsGeminiModel checks if the model is from Google
func (m Models) IsGeminiModel() bool {
	return strings.HasPrefix(string(m), "gemini-")
}

// IsMetaModel checks if the model is from Meta
func (m Models) IsMetaModel() bool {
	return strings.HasPrefix(string(m), "llama-") || strings.HasPrefix(string(m), "meta.")
}

func (m Models) IsAnthropicModel() bool {
	return strings.HasPrefix(string(m), "claude-")
}

func (m Models) IsAmazonModel() bool {
	return strings.HasPrefix(string(m), "amazon.")
}

func (m Models) IsCohereModel() bool {
	return strings.HasPrefix(string(m), "cohere.")
}

func (m Models) IsMistralModel() bool {
	return strings.HasPrefix(string(m), "mistral.")
}

func (m Models) IsStabilityModel() bool {
	return strings.HasPrefix(string(m), "stability.")
}

func (m Models) IsAI21Model() bool {
	return strings.HasPrefix(string(m), "ai21.")
}

func (m Models) IsGoogleModel() bool {
	return strings.HasPrefix(string(m), "palm-") || strings.HasPrefix(string(m), "gemini-")
}

func (m Models) ToClaudeModel() anthropic.Model {
	return anthropic.Model(string(m))
}

func (m Models) IsBedrockModel() bool {
	bedrockPrefixes := []string{
		"meta.", "mistral.", "amazon.", "stability.", "ai21.", "anthropic.", "cohere.", "apac.", "us.anthropic", "us.meta",
	}

	for _, prefix := range bedrockPrefixes {
		if strings.HasPrefix(string(m), prefix) {
			return true
		}
	}
	return false
}

// GetModelProvider returns the provider of the model
func (m Models) GetModelProvider() ModelProviders {
	switch {
	case m.IsOpenAIModel():
		return OpenAI
	case m.IsAnthropicModel():
		return Anthropic
	case m.IsMetaModel():
		return Meta
	default:
		return Undefined
	}
}

// KarmaAI is a struct that holds the model and configurations for the AI
type KarmaAI struct {
	Model         Models
	SystemMessage string
	Context       string
	UserPrePrompt string // User pre-prompt is the message that is added before the user's message
	Temperature   float64
	TopP          float64
	TopK          float64
	MaxTokens     int64
	ResponseType  string // `text/plain`, `application/json`, `application/xml`, `application/yaml` and `text/x.enum`
}

// Option is a function type that modifies KarmaAI
type Option func(*KarmaAI)

// WithSystemMessage sets the system message
func WithSystemMessage(message string) Option {
	return func(k *KarmaAI) {
		k.SystemMessage = message
	}
}

// WithContext sets the context
func WithContext(context string) Option {
	return func(k *KarmaAI) {
		k.Context = context
	}
}

// WithUserPrePrompt sets the user pre-prompt
func WithUserPrePrompt(prePrompt string) Option {
	return func(k *KarmaAI) {
		k.UserPrePrompt = prePrompt
	}
}

// WithTemperature sets the temperature
func WithTemperature(temperature float64) Option {
	return func(k *KarmaAI) {
		k.Temperature = temperature
	}
}

// WithMaxTokens sets the max tokens
func WithMaxTokens(maxTokens int64) Option {
	return func(k *KarmaAI) {
		k.MaxTokens = maxTokens
	}
}

// WithTopP sets the top p
func WithTopP(topP float64) Option {
	return func(k *KarmaAI) {
		k.TopP = topP
	}
}

// WithTopK sets the top k
func WithTopK(topK float64) Option {
	return func(k *KarmaAI) {
		k.TopK = topK
	}
}

func WithResponseType(responseType string) Option {
	return func(k *KarmaAI) {
		k.ResponseType = responseType
	}
}

// NewKarmaAI creates a new KarmaAI instance with required parameters and optional configurations
func NewKarmaAI(model any, opts ...Option) *KarmaAI {
	modelVal, ok := model.(Models)
	if !ok {
		panic("model must be of type Models")
	}
	karma := &KarmaAI{
		Model: modelVal,
	}
	if karma.MaxTokens == 0 {
		karma.MaxTokens = 500
	}

	// Apply all provided options
	for _, opt := range opts {
		opt(karma)
	}

	return karma
}
