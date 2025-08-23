package variants

import "github.com/MelloB1989/karma/ai"

type ModelVariant string

// --- OpenAI ---
const (
	// GPT-4
	GPT4          ModelVariant = "gpt-4"
	GPT4_0314     ModelVariant = "gpt-4-0314"
	GPT4_0613     ModelVariant = "gpt-4-0613"
	GPT4_32K      ModelVariant = "gpt-4-32k"
	GPT4_32K_0314 ModelVariant = "gpt-4-32k-0314"
	GPT4_32K_0613 ModelVariant = "gpt-4-32k-0613"

	// GPT-4o
	GPT4o                     ModelVariant = "gpt-4o"
	GPT4o_20241120            ModelVariant = "gpt-4o-2024-11-20"
	GPT4o_20240806            ModelVariant = "gpt-4o-2024-08-06"
	GPT4o_20240513            ModelVariant = "gpt-4o-2024-05-13"
	GPT4o_AudioPreview        ModelVariant = "gpt-4o-audio-preview"
	GPT4o_AudioPreview_202410 ModelVariant = "gpt-4o-audio-preview-2024-10-01"
	GPT4o_AudioPreview_202412 ModelVariant = "gpt-4o-audio-preview-2024-12-17"
	ChatGPT4oLatest           ModelVariant = "chatgpt-4o-latest"

	// GPT-4o Mini
	GPT4oMini                     ModelVariant = "gpt-4o-mini"
	GPT4oMini_20240718            ModelVariant = "gpt-4o-mini-2024-07-18"
	GPT4oMini_AudioPreview        ModelVariant = "gpt-4o-mini-audio-preview"
	GPT4oMini_AudioPreview_202412 ModelVariant = "gpt-4o-mini-audio-preview-2024-12-17"

	// GPT-4 Turbo
	GPT4Turbo          ModelVariant = "gpt-4-turbo"
	GPT4Turbo_20240409 ModelVariant = "gpt-4-turbo-2024-04-09"
	GPT4_0125Preview   ModelVariant = "gpt-4-0125-preview"
	GPT4TurboPreview   ModelVariant = "gpt-4-turbo-preview"
	GPT4_1106Preview   ModelVariant = "gpt-4-1106-preview"
	GPT4VisionPreview  ModelVariant = "gpt-4-vision-preview"

	// GPT-3.5 Turbo
	GPT35Turbo         ModelVariant = "gpt-3.5-turbo"
	GPT35Turbo16K      ModelVariant = "gpt-3.5-turbo-16k"
	GPT35Turbo0301     ModelVariant = "gpt-3.5-turbo-0301"
	GPT35Turbo0613     ModelVariant = "gpt-3.5-turbo-0613"
	GPT35Turbo1106     ModelVariant = "gpt-3.5-turbo-1106"
	GPT35Turbo0125     ModelVariant = "gpt-3.5-turbo-0125"
	GPT35Turbo16K_0613 ModelVariant = "gpt-3.5-turbo-16k-0613"

	// GPT-5
	GPT5     ModelVariant = "gpt-5"
	GPT5Nano ModelVariant = "gpt-5-nano"
	GPT5Mini ModelVariant = "gpt-5-mini"

	// O1
	O1                 ModelVariant = "o1"
	O1_20241217        ModelVariant = "o1-2024-12-17"
	O1Preview          ModelVariant = "o1-preview"
	O1Preview_20240912 ModelVariant = "o1-preview-2024-09-12"
	O1Mini             ModelVariant = "o1-mini"
	O1Mini_20240912    ModelVariant = "o1-mini-2024-09-12"
)

// --- Anthropic ---
const (
	// Claude Instant
	ClaudeInstantV12      ModelVariant = "claude-instant-v1.2"
	ClaudeInstantV12_100k ModelVariant = "claude-instant-v1.2-100k"
	ClaudeInstantV1       ModelVariant = "claude-instant-v1"

	// Claude V2
	ClaudeV20      ModelVariant = "claude-v2.0"
	ClaudeV20_18k  ModelVariant = "claude-v2.0-18k"
	ClaudeV20_100k ModelVariant = "claude-v2.0-100k"
	ClaudeV21      ModelVariant = "claude-v2.1"
	ClaudeV21_18k  ModelVariant = "claude-v2.1-18k"
	ClaudeV21_200k ModelVariant = "claude-v2.1-200k"
	ClaudeV2       ModelVariant = "claude-v2"

	// Claude 3 Sonnet
	Claude3Sonnet_20240229       ModelVariant = "claude-3-sonnet-20240229"
	Claude3Sonnet_20240229_v28k  ModelVariant = "claude-3-sonnet-20240229-v1:28k"
	Claude3Sonnet_20240229_v200k ModelVariant = "claude-3-sonnet-20240229-v1:200k"
	Claude3Sonnet_20240229_v1    ModelVariant = "claude-3-sonnet-20240229-v1"

	// Claude 3 Haiku
	Claude3Haiku_20240307       ModelVariant = "claude-3-haiku-20240307"
	Claude3Haiku_20240307_v48k  ModelVariant = "claude-3-haiku-20240307-v1:48k"
	Claude3Haiku_20240307_v200k ModelVariant = "claude-3-haiku-20240307-v1:200k"
	Claude3Haiku_20240307_v1    ModelVariant = "claude-3-haiku-20240307-v1"

	// Claude 3 Opus
	Claude3Opus_20240229       ModelVariant = "claude-3-opus-20240229"
	Claude3Opus_20240229_v12k  ModelVariant = "claude-3-opus-20240229-v1:12k"
	Claude3Opus_20240229_v28k  ModelVariant = "claude-3-opus-20240229-v1:28k"
	Claude3Opus_20240229_v200k ModelVariant = "claude-3-opus-20240229-v1:200k"
	Claude3Opus_20240229_v1    ModelVariant = "claude-3-opus-20240229-v1"
	Claude3OpusLatest          ModelVariant = "claude-3-opus-latest"

	// Claude 3.5 Sonnet
	Claude35Sonnet_20240620    ModelVariant = "claude-3.5-sonnet-20240620"
	Claude35Sonnet_20240620_v1 ModelVariant = "claude-3.5-sonnet-20240620-v1"
	Claude35Sonnet_20241022    ModelVariant = "claude-3.5-sonnet-20241022"
	Claude35Sonnet_20241022_v2 ModelVariant = "claude-3.5-sonnet-20241022-v2"
	Claude35SonnetLatest       ModelVariant = "claude-3.5-sonnet-latest"

	// Claude 3.5 Haiku
	Claude35Haiku_20241022    ModelVariant = "claude-3.5-haiku-20241022"
	Claude35Haiku_20241022_v1 ModelVariant = "claude-3.5-haiku-20241022-v1"
	Claude35HaikuLatest       ModelVariant = "claude-3.5-haiku-latest"

	// Claude 3.7 Sonnet
	Claude37Sonnet_20250219    ModelVariant = "claude-3.7-sonnet-20250219"
	Claude37Sonnet_20250219_v1 ModelVariant = "claude-3.7-sonnet-20250219-v1"
	Claude37SonnetLatest       ModelVariant = "claude-3.7-sonnet-latest"

	// Claude 4
	Claude4Sonnet_20250514 ModelVariant = "claude-4-sonnet-20250514"
	Claude4SonnetAlias     ModelVariant = "claude-sonnet-4-20250514"
	Claude4Sonnet40        ModelVariant = "claude-sonnet-4.0"
	Claude4Opus_20250514   ModelVariant = "claude-4-opus-20250514"
	Claude4OpusAlias       ModelVariant = "claude-opus-4-20250514"
	Claude4Opus40          ModelVariant = "claude-opus-4.0"
)

// --- Bedrock ---
const (
	// Anthropic via Bedrock
	BedrockClaudeInstantV1            ModelVariant = "anthropic.claude-instant-v1"
	BedrockClaudeInstantV12_100k      ModelVariant = "anthropic.claude-instant-v1:2:100k"
	BedrockClaudeV2                   ModelVariant = "anthropic.claude-v2"
	BedrockClaudeV20_18k              ModelVariant = "anthropic.claude-v2:0:18k"
	BedrockClaudeV20_100k             ModelVariant = "anthropic.claude-v2:0:100k"
	BedrockClaudeV21                  ModelVariant = "anthropic.claude-v2:1"
	BedrockClaudeV21_18k              ModelVariant = "anthropic.claude-v2:1:18k"
	BedrockClaudeV21_200k             ModelVariant = "anthropic.claude-v2:1:200k"
	BedrockClaude3Sonnet20240229      ModelVariant = "anthropic.claude-3-sonnet-20240229-v1:0"
	BedrockClaude3Sonnet20240229_28k  ModelVariant = "anthropic.claude-3-sonnet-20240229-v1:0:28k"
	BedrockClaude3Sonnet20240229_200k ModelVariant = "anthropic.claude-3-sonnet-20240229-v1:0:200k"
	BedrockClaude3Haiku20240307       ModelVariant = "anthropic.claude-3-haiku-20240307-v1:0"
	BedrockClaude3Haiku20240307_48k   ModelVariant = "anthropic.claude-3-haiku-20240307-v1:0:48k"
	BedrockClaude3Haiku20240307_200k  ModelVariant = "anthropic.claude-3-haiku-20240307-v1:0:200k"
	BedrockClaude3Opus20240229        ModelVariant = "anthropic.claude-3-opus-20240229-v1:0"
	BedrockClaude3Opus20240229_12k    ModelVariant = "anthropic.claude-3-opus-20240229-v1:0:12k"
	BedrockClaude3Opus20240229_28k    ModelVariant = "anthropic.claude-3-opus-20240229-v1:0:28k"
	BedrockClaude3Opus20240229_200k   ModelVariant = "anthropic.claude-3-opus-20240229-v1:0:200k"
	BedrockClaude35Sonnet20240620     ModelVariant = "anthropic.claude-3-5-sonnet-20240620-v1:0"
	BedrockClaude35Sonnet20241022     ModelVariant = "anthropic.claude-3-5-sonnet-20241022-v2:0"
	BedrockClaude35Haiku20241022      ModelVariant = "anthropic.claude-3-5-haiku-20241022-v1:0"
	BedrockClaude37Sonnet20250219     ModelVariant = "us.anthropic.claude-3-7-sonnet-20250219-v1:0"
	BedrockClaude35SonnetAPAC         ModelVariant = "apac.anthropic.claude-3-5-sonnet-20240620-v1:0"

	// Llama
	Llama3_8B      ModelVariant = "meta.llama3-8b-instruct-v1:0"
	Llama3_70B     ModelVariant = "meta.llama3-70b-instruct-v1:0"
	Llama31_8B     ModelVariant = "meta.llama3-1-8b-instruct-v1:0"
	Llama31_70B    ModelVariant = "meta.llama3-1-70b-instruct-v1:0"
	Llama32_1B     ModelVariant = "meta.llama3-2-1b-instruct-v1:0"
	Llama32_3B     ModelVariant = "meta.llama3-2-3b-instruct-v1:0"
	Llama32_11B    ModelVariant = "meta.llama3-2-11b-instruct-v1:0"
	Llama32_90B    ModelVariant = "meta.llama3-2-90b-instruct-v1:0"
	Llama33_70B    ModelVariant = "meta.llama3-3-70b-instruct-v1:0"
	Llama33_70B_US ModelVariant = "us.meta.llama3-3-70b-instruct-v1:0"

	// Mistral
	Mistral7B    ModelVariant = "mistral.mistral-7b-instruct-v0:2"
	Mixtral8x7B  ModelVariant = "mistral.mixtral-8x7b-instruct-v0:1"
	MistralLarge ModelVariant = "mistral.mistral-large-2402-v1:0"
	MistralSmall ModelVariant = "mistral.mistral-small-2402-v1:0"

	// Titan
	TitanTG1Large       ModelVariant = "amazon.titan-tg1-large"
	TitanTextPremier    ModelVariant = "amazon.titan-text-premier-v1:0"
	TitanTextLite       ModelVariant = "amazon.titan-text-lite-v1"
	TitanTextLite_0     ModelVariant = "amazon.titan-text-lite-v1:0"
	TitanTextLite_4k    ModelVariant = "amazon.titan-text-lite-v1:0:4k"
	TitanTextExpress    ModelVariant = "amazon.titan-text-express-v1"
	TitanTextExpress_0  ModelVariant = "amazon.titan-text-express-v1:0"
	TitanTextExpress_8k ModelVariant = "amazon.titan-text-express-v1:0:8k"
	TitanEmbedText      ModelVariant = "amazon.titan-embed-text-v1"
	TitanEmbedText_2    ModelVariant = "amazon.titan-embed-text-v1:2"
	TitanEmbedText_8k   ModelVariant = "amazon.titan-embed-text-v1:2:8k"
	TitanEmbedTextV2    ModelVariant = "amazon.titan-embed-text-v2:0"
	TitanEmbedTextV2_8k ModelVariant = "amazon.titan-embed-text-v2:0:8k"
	TitanEmbedG1_02     ModelVariant = "amazon.titan-embed-g1-text-02"
	TitanEmbedImage     ModelVariant = "amazon.titan-embed-image-v1:0"

	// Nova
	NovaPro       ModelVariant = "amazon.nova-pro-v1:0"
	NovaPro300k   ModelVariant = "amazon.nova-pro-v1:0:300k"
	NovaLite      ModelVariant = "amazon.nova-lite-v1:0"
	NovaLite300k  ModelVariant = "amazon.nova-lite-v1:0:300k"
	NovaCanvas    ModelVariant = "amazon.nova-canvas-v1:0"
	NovaReel      ModelVariant = "amazon.nova-reel-v1:0"
	NovaMicro     ModelVariant = "amazon.nova-micro-v1:0"
	NovaMicro128k ModelVariant = "amazon.nova-micro-v1:0:128k"
)

// --- Google ---
const (
	// Gemini
	Gemini25Flash         ModelVariant = "gemini-2.5-flash"
	Gemini25FlashPreview  ModelVariant = "gemini-2.5-flash-preview-04-17"
	Gemini25Pro           ModelVariant = "gemini-2.5-pro"
	Gemini25ProPreview    ModelVariant = "gemini-2.5-pro-preview"
	Gemini25ProPreview506 ModelVariant = "gemini-2.5-pro-preview-05-06"
	Gemini20Flash         ModelVariant = "gemini-2.0-flash"
	Gemini20FlashLite     ModelVariant = "gemini-2.0-flash-lite"
	Gemini20FlashLive     ModelVariant = "gemini-2.0-flash-live"
	Gemini20FlashLive001  ModelVariant = "gemini-2.0-flash-live-001"
	Gemini15Flash         ModelVariant = "gemini-1.5-flash"
	Gemini15Flash8B       ModelVariant = "gemini-1.5-flash-8b"
	Gemini15Pro           ModelVariant = "gemini-1.5-pro"

	// Embeddings
	TextEmbedding004 ModelVariant = "text-embedding-004"
	GeminiEmbedExp   ModelVariant = "gemini-embedding-exp"

	// Legacy
	PaLM2 ModelVariant = "palm-2"
)

// --- XAI ---
const (
	Grok4      ModelVariant = "grok-4"
	Grok4_0709 ModelVariant = "grok-4-0709"
	Grok3      ModelVariant = "grok-3"
	Grok3Mini  ModelVariant = "grok-3-mini"
)

// ---------------------------
// 🔹 Provider Model Mapping
// ---------------------------

var ProviderModelMapping = map[ai.Provider]map[ModelVariant]ai.BaseModel{
	// ---------------- OpenAI ----------------
	ai.OpenAI: {
		// GPT-4
		GPT4:          ai.GPT4,
		GPT4_0314:     ai.GPT4,
		GPT4_0613:     ai.GPT4,
		GPT4_32K:      ai.GPT4,
		GPT4_32K_0314: ai.GPT4,
		GPT4_32K_0613: ai.GPT4,

		// GPT-4o
		GPT4o:                     ai.GPT4o,
		GPT4o_20241120:            ai.GPT4o,
		GPT4o_20240806:            ai.GPT4o,
		GPT4o_20240513:            ai.GPT4o,
		GPT4o_AudioPreview:        ai.GPT4o,
		GPT4o_AudioPreview_202410: ai.GPT4o,
		GPT4o_AudioPreview_202412: ai.GPT4o,
		ChatGPT4oLatest:           ai.GPT4o,

		// GPT-4o Mini
		GPT4oMini:                     ai.GPT4oMini,
		GPT4oMini_20240718:            ai.GPT4oMini,
		GPT4oMini_AudioPreview:        ai.GPT4oMini,
		GPT4oMini_AudioPreview_202412: ai.GPT4oMini,

		// GPT-4 Turbo
		GPT4Turbo:          ai.GPT4Turbo,
		GPT4Turbo_20240409: ai.GPT4Turbo,
		GPT4_0125Preview:   ai.GPT4Turbo,
		GPT4TurboPreview:   ai.GPT4Turbo,
		GPT4_1106Preview:   ai.GPT4Turbo,
		GPT4VisionPreview:  ai.GPT4Turbo,

		// GPT-3.5 Turbo
		GPT35Turbo:         ai.GPT35Turbo,
		GPT35Turbo16K:      ai.GPT35Turbo,
		GPT35Turbo0301:     ai.GPT35Turbo,
		GPT35Turbo0613:     ai.GPT35Turbo,
		GPT35Turbo1106:     ai.GPT35Turbo,
		GPT35Turbo0125:     ai.GPT35Turbo,
		GPT35Turbo16K_0613: ai.GPT35Turbo,

		// GPT-5
		GPT5:     ai.GPT5,
		GPT5Nano: ai.GPT5,
		GPT5Mini: ai.GPT5,

		// O1
		O1:                 ai.O1,
		O1_20241217:        ai.O1,
		O1Preview:          ai.O1,
		O1Preview_20240912: ai.O1,
		O1Mini:             ai.O1,
		O1Mini_20240912:    ai.O1,
	},

	// ---------------- Anthropic ----------------
	ai.Anthropic: {
		ClaudeInstantV12:      ai.ClaudeInstant,
		ClaudeInstantV12_100k: ai.ClaudeInstant,
		ClaudeInstantV1:       ai.ClaudeInstant,

		ClaudeV20:      ai.ClaudeV2,
		ClaudeV20_18k:  ai.ClaudeV2,
		ClaudeV20_100k: ai.ClaudeV2,
		ClaudeV21:      ai.ClaudeV2,
		ClaudeV21_18k:  ai.ClaudeV2,
		ClaudeV21_200k: ai.ClaudeV2,
		ClaudeV2:       ai.ClaudeV2,

		Claude3Sonnet_20240229:       ai.Claude3Sonnet,
		Claude3Sonnet_20240229_v28k:  ai.Claude3Sonnet,
		Claude3Sonnet_20240229_v200k: ai.Claude3Sonnet,
		Claude3Sonnet_20240229_v1:    ai.Claude3Sonnet,

		Claude3Haiku_20240307:       ai.Claude3Haiku,
		Claude3Haiku_20240307_v48k:  ai.Claude3Haiku,
		Claude3Haiku_20240307_v200k: ai.Claude3Haiku,
		Claude3Haiku_20240307_v1:    ai.Claude3Haiku,

		Claude3Opus_20240229:       ai.Claude3Opus,
		Claude3Opus_20240229_v12k:  ai.Claude3Opus,
		Claude3Opus_20240229_v28k:  ai.Claude3Opus,
		Claude3Opus_20240229_v200k: ai.Claude3Opus,
		Claude3Opus_20240229_v1:    ai.Claude3Opus,
		Claude3OpusLatest:          ai.Claude3Opus,

		Claude35Sonnet_20240620:    ai.Claude35Sonnet,
		Claude35Sonnet_20240620_v1: ai.Claude35Sonnet,
		Claude35Sonnet_20241022:    ai.Claude35Sonnet,
		Claude35Sonnet_20241022_v2: ai.Claude35Sonnet,
		Claude35SonnetLatest:       ai.Claude35Sonnet,

		Claude35Haiku_20241022:    ai.Claude35Haiku,
		Claude35Haiku_20241022_v1: ai.Claude35Haiku,
		Claude35HaikuLatest:       ai.Claude35Haiku,

		Claude37Sonnet_20250219:    ai.Claude37Sonnet,
		Claude37Sonnet_20250219_v1: ai.Claude37Sonnet,
		Claude37SonnetLatest:       ai.Claude37Sonnet,

		Claude4Sonnet_20250514: ai.Claude4Sonnet,
		Claude4SonnetAlias:     ai.Claude4Sonnet,
		Claude4Sonnet40:        ai.Claude4Sonnet,

		Claude4Opus_20250514: ai.Claude4Opus,
		Claude4OpusAlias:     ai.Claude4Opus,
		Claude4Opus40:        ai.Claude4Opus,
	},

	// ---------------- Bedrock ----------------
	ai.Bedrock: {
		// Claude (via Bedrock)
		BedrockClaudeInstantV1:       ai.ClaudeInstant,
		BedrockClaudeInstantV12_100k: ai.ClaudeInstant,
		BedrockClaudeV2:              ai.ClaudeV2,
		BedrockClaudeV20_18k:         ai.ClaudeV2,
		BedrockClaudeV20_100k:        ai.ClaudeV2,
		BedrockClaudeV21:             ai.ClaudeV2,
		BedrockClaudeV21_18k:         ai.ClaudeV2,
		BedrockClaudeV21_200k:        ai.ClaudeV2,

		BedrockClaude3Sonnet20240229:      ai.Claude3Sonnet,
		BedrockClaude3Sonnet20240229_28k:  ai.Claude3Sonnet,
		BedrockClaude3Sonnet20240229_200k: ai.Claude3Sonnet,

		BedrockClaude3Haiku20240307:      ai.Claude3Haiku,
		BedrockClaude3Haiku20240307_48k:  ai.Claude3Haiku,
		BedrockClaude3Haiku20240307_200k: ai.Claude3Haiku,

		BedrockClaude3Opus20240229:      ai.Claude3Opus,
		BedrockClaude3Opus20240229_12k:  ai.Claude3Opus,
		BedrockClaude3Opus20240229_28k:  ai.Claude3Opus,
		BedrockClaude3Opus20240229_200k: ai.Claude3Opus,

		BedrockClaude35Sonnet20240620: ai.Claude35Sonnet,
		BedrockClaude35Sonnet20241022: ai.Claude35Sonnet,
		BedrockClaude35Haiku20241022:  ai.Claude35Haiku,
		BedrockClaude37Sonnet20250219: ai.Claude37Sonnet,
		BedrockClaude35SonnetAPAC:     ai.Claude35Sonnet,

		// Meta Llama
		Llama3_8B:      ai.Llama3_8B,
		Llama3_70B:     ai.Llama3_70B,
		Llama31_8B:     ai.Llama31_8B,
		Llama31_70B:    ai.Llama31_70B,
		Llama32_1B:     ai.Llama32_1B,
		Llama32_3B:     ai.Llama32_3B,
		Llama32_11B:    ai.Llama32_11B,
		Llama32_90B:    ai.Llama32_90B,
		Llama33_70B:    ai.Llama33_70B,
		Llama33_70B_US: ai.Llama33_70B,

		// Mistral
		Mistral7B:    ai.Mistral7B,
		Mixtral8x7B:  ai.Mixtral8x7B,
		MistralLarge: ai.MistralLarge,
		MistralSmall: ai.MistralSmall,

		// Titan
		TitanTG1Large:       ai.TitanTextG1Large,
		TitanTextPremier:    ai.TitanTextPremier,
		TitanTextLite:       ai.TitanTextLite,
		TitanTextLite_0:     ai.TitanTextLite,
		TitanTextLite_4k:    ai.TitanTextLite,
		TitanTextExpress:    ai.TitanTextExpress,
		TitanTextExpress_0:  ai.TitanTextExpress,
		TitanTextExpress_8k: ai.TitanTextExpress,
		TitanEmbedText:      ai.TitanEmbedText,
		TitanEmbedText_2:    ai.TitanEmbedText,
		TitanEmbedText_8k:   ai.TitanEmbedText,
		TitanEmbedTextV2:    ai.TitanEmbedText,
		TitanEmbedTextV2_8k: ai.TitanEmbedText,
		TitanEmbedG1_02:     ai.TitanEmbedText,
		TitanEmbedImage:     ai.TitanEmbedImage,

		// Nova
		NovaPro:       ai.NovaPro,
		NovaPro300k:   ai.NovaPro,
		NovaLite:      ai.NovaLite,
		NovaLite300k:  ai.NovaLite,
		NovaCanvas:    ai.NovaCanvas,
		NovaReel:      ai.NovaReel,
		NovaMicro:     ai.NovaMicro,
		NovaMicro128k: ai.NovaMicro,
	},

	// ---------------- Google ----------------
	ai.Google: {
		Gemini25Flash:         ai.Gemini25Flash,
		Gemini25FlashPreview:  ai.Gemini25Flash,
		Gemini25Pro:           ai.Gemini25Pro,
		Gemini25ProPreview:    ai.Gemini25Pro,
		Gemini25ProPreview506: ai.Gemini25Pro,

		Gemini20Flash:        ai.Gemini20Flash,
		Gemini20FlashLite:    ai.Gemini20FlashLite,
		Gemini20FlashLive:    ai.Gemini20Flash,
		Gemini20FlashLive001: ai.Gemini20Flash,

		Gemini15Flash:   ai.Gemini15Flash,
		Gemini15Flash8B: ai.Gemini15Flash8B,
		Gemini15Pro:     ai.Gemini15Pro,

		TextEmbedding004: ai.GeminiEmbedding,
		GeminiEmbedExp:   ai.GeminiEmbedding,
		PaLM2:            ai.PaLM2,
	},

	// ---------------- xAI ----------------
	ai.XAI: {
		Grok4:      ai.Grok4,
		Grok4_0709: ai.Grok4,
		Grok3:      ai.Grok3,
		Grok3Mini:  ai.Grok3Mini,
	},
}

// Reverse lookup for variants of a given BaseModel
func GetVariantsForBaseModel(base ai.BaseModel) []ModelVariant {
	var variants []ModelVariant
	for _, providerModels := range ProviderModelMapping {
		for variant, model := range providerModels {
			if model == base {
				variants = append(variants, variant)
			}
		}
	}
	return variants
}

// Lookup provider + variant → base model
func GetBaseModel(provider ai.Provider, variant ModelVariant) (ai.BaseModel, bool) {
	models, ok := ProviderModelMapping[provider]
	if !ok {
		return "", false
	}
	base, ok := models[variant]
	return base, ok
}

// // ProviderModelMapping maps provider-specific model names to base models
// var ProviderModelMapping = map[ai.Provider]map[string]ai.BaseModel{
// 	ai.OpenAI: {
// 		// GPT-4 variants
// 		"gpt-4":          ai.GPT4,
// 		"gpt-4-0314":     ai.GPT4,
// 		"gpt-4-0613":     ai.GPT4,
// 		"gpt-4-32k":      ai.GPT4,
// 		"gpt-4-32k-0314": ai.GPT4,
// 		"gpt-4-32k-0613": ai.GPT4,

// 		// GPT-4o variants
// 		"gpt-4o":                          ai.GPT4o,
// 		"gpt-4o-2024-11-20":               ai.GPT4o,
// 		"gpt-4o-2024-08-06":               ai.GPT4o,
// 		"gpt-4o-2024-05-13":               ai.GPT4o,
// 		"gpt-4o-audio-preview":            ai.GPT4o,
// 		"gpt-4o-audio-preview-2024-10-01": ai.GPT4o,
// 		"gpt-4o-audio-preview-2024-12-17": ai.GPT4o,
// 		"chatgpt-4o-latest":               ai.GPT4o,

// 		// GPT-4o Mini variants
// 		"gpt-4o-mini":                          ai.GPT4oMini,
// 		"gpt-4o-mini-2024-07-18":               ai.GPT4oMini,
// 		"gpt-4o-mini-audio-preview":            ai.GPT4oMini,
// 		"gpt-4o-mini-audio-preview-2024-12-17": ai.GPT4oMini,

// 		// GPT-4 Turbo variants
// 		"gpt-4-turbo":            ai.GPT4Turbo,
// 		"gpt-4-turbo-2024-04-09": ai.GPT4Turbo,
// 		"gpt-4-0125-preview":     ai.GPT4Turbo,
// 		"gpt-4-turbo-preview":    ai.GPT4Turbo,
// 		"gpt-4-1106-preview":     ai.GPT4Turbo,
// 		"gpt-4-vision-preview":   ai.GPT4Turbo,

// 		// GPT-3.5 Turbo variants
// 		"gpt-3.5-turbo":          ai.GPT35Turbo,
// 		"gpt-3.5-turbo-16k":      ai.GPT35Turbo,
// 		"gpt-3.5-turbo-0301":     ai.GPT35Turbo,
// 		"gpt-3.5-turbo-0613":     ai.GPT35Turbo,
// 		"gpt-3.5-turbo-1106":     ai.GPT35Turbo,
// 		"gpt-3.5-turbo-0125":     ai.GPT35Turbo,
// 		"gpt-3.5-turbo-16k-0613": ai.GPT35Turbo,

// 		// GPT-5 variants
// 		"gpt-5":      ai.GPT5,
// 		"gpt-5-nano": ai.GPT5Nano,
// 		"gpt-5-mini": ai.GPT5Mini,

// 		// O1 variants
// 		"o1":                    ai.O1,
// 		"o1-2024-12-17":         ai.O1,
// 		"o1-preview":            ai.O1Preview,
// 		"o1-preview-2024-09-12": ai.O1Preview,
// 		"o1-mini":               ai.O1Mini,
// 		"o1-mini-2024-09-12":    ai.O1Mini,
// 	},

// 	ai.Anthropic: {
// 		// Claude Instant variants
// 		"claude-instant-v1.2":      ai.ClaudeInstant,
// 		"claude-instant-v1.2-100k": ai.ClaudeInstant,
// 		"claude-instant-v1":        ai.ClaudeInstant,

// 		// Claude v2 variants
// 		"claude-v2.0":      ai.ClaudeV2,
// 		"claude-v2.0-18k":  ai.ClaudeV2,
// 		"claude-v2.0-100k": ai.ClaudeV2,
// 		"claude-v2.1":      ai.ClaudeV2,
// 		"claude-v2.1-18k":  ai.ClaudeV2,
// 		"claude-v2.1-200k": ai.ClaudeV2,
// 		"claude-v2":        ai.ClaudeV2,

// 		// Claude 3 Sonnet variants
// 		"claude-3-sonnet-20240229":         ai.Claude3Sonnet,
// 		"claude-3-sonnet-20240229-v1:28k":  ai.Claude3Sonnet,
// 		"claude-3-sonnet-20240229-v1:200k": ai.Claude3Sonnet,
// 		"claude-3-sonnet-20240229-v1":      ai.Claude3Sonnet,

// 		// Claude 3 Haiku variants
// 		"claude-3-haiku-20240307":         ai.Claude3Haiku,
// 		"claude-3-haiku-20240307-v1:48k":  ai.Claude3Haiku,
// 		"claude-3-haiku-20240307-v1:200k": ai.Claude3Haiku,
// 		"claude-3-haiku-20240307-v1":      ai.Claude3Haiku,

// 		// Claude 3 Opus variants
// 		"claude-3-opus-20240229":         ai.Claude3Opus,
// 		"claude-3-opus-20240229-v1:12k":  ai.Claude3Opus,
// 		"claude-3-opus-20240229-v1:28k":  ai.Claude3Opus,
// 		"claude-3-opus-20240229-v1:200k": ai.Claude3Opus,
// 		"claude-3-opus-20240229-v1":      ai.Claude3Opus,
// 		"claude-3-opus-latest":           ai.Claude3Opus,

// 		// Claude 3.5 Sonnet variants
// 		"claude-3.5-sonnet-20240620":    ai.Claude35Sonnet,
// 		"claude-3.5-sonnet-20240620-v1": ai.Claude35Sonnet,
// 		"claude-3.5-sonnet-20241022":    ai.Claude35Sonnet,
// 		"claude-3.5-sonnet-20241022-v2": ai.Claude35Sonnet,
// 		"claude-3.5-sonnet-latest":      ai.Claude35Sonnet,

// 		// Claude 3.5 Haiku variants
// 		"claude-3.5-haiku-20241022":    ai.Claude35Haiku,
// 		"claude-3.5-haiku-20241022-v1": ai.Claude35Haiku,
// 		"claude-3.5-haiku-latest":      ai.Claude35Haiku,

// 		// Claude 3.7 Sonnet variants
// 		"claude-3.7-sonnet-20250219":    ai.Claude37Sonnet,
// 		"claude-3.7-sonnet-20250219-v1": ai.Claude37Sonnet,
// 		"claude-3.7-sonnet-latest":      ai.Claude37Sonnet,

// 		// Claude 4 variants
// 		"claude-4-sonnet-20250514": ai.Claude4Sonnet,
// 		"claude-sonnet-4-20250514": ai.Claude4Sonnet,
// 		"claude-sonnet-4.0":        ai.Claude4Sonnet,
// 		"claude-4-opus-20250514":   ai.Claude4Opus,
// 		"claude-opus-4-20250514":   ai.Claude4Opus,
// 		"claude-opus-4.0":          ai.Claude4Opus,
// 	},

// 	ai.Bedrock: {
// 		// Claude models on Bedrock
// 		"anthropic.claude-instant-v1":                    ai.ClaudeInstant,
// 		"anthropic.claude-instant-v1:2:100k":             ai.ClaudeInstant,
// 		"anthropic.claude-v2":                            ai.ClaudeV2,
// 		"anthropic.claude-v2:0:18k":                      ai.ClaudeV2,
// 		"anthropic.claude-v2:0:100k":                     ai.ClaudeV2,
// 		"anthropic.claude-v2:1":                          ai.ClaudeV2,
// 		"anthropic.claude-v2:1:18k":                      ai.ClaudeV2,
// 		"anthropic.claude-v2:1:200k":                     ai.ClaudeV2,
// 		"anthropic.claude-3-sonnet-20240229-v1:0":        ai.Claude3Sonnet,
// 		"anthropic.claude-3-sonnet-20240229-v1:0:28k":    ai.Claude3Sonnet,
// 		"anthropic.claude-3-sonnet-20240229-v1:0:200k":   ai.Claude3Sonnet,
// 		"anthropic.claude-3-haiku-20240307-v1:0":         ai.Claude3Haiku,
// 		"anthropic.claude-3-haiku-20240307-v1:0:48k":     ai.Claude3Haiku,
// 		"anthropic.claude-3-haiku-20240307-v1:0:200k":    ai.Claude3Haiku,
// 		"anthropic.claude-3-opus-20240229-v1:0":          ai.Claude3Opus,
// 		"anthropic.claude-3-opus-20240229-v1:0:12k":      ai.Claude3Opus,
// 		"anthropic.claude-3-opus-20240229-v1:0:28k":      ai.Claude3Opus,
// 		"anthropic.claude-3-opus-20240229-v1:0:200k":     ai.Claude3Opus,
// 		"anthropic.claude-3-5-sonnet-20240620-v1:0":      ai.Claude35Sonnet,
// 		"anthropic.claude-3-5-sonnet-20241022-v2:0":      ai.Claude35Sonnet,
// 		"anthropic.claude-3-5-haiku-20241022-v1:0":       ai.Claude35Haiku,
// 		"us.anthropic.claude-3-7-sonnet-20250219-v1:0":   ai.Claude37Sonnet,
// 		"apac.anthropic.claude-3-5-sonnet-20240620-v1:0": ai.Claude35Sonnet,

// 		// Llama models on Bedrock
// 		"meta.llama3-8b-instruct-v1:0":       ai.Llama3_8B,
// 		"meta.llama3-70b-instruct-v1:0":      ai.Llama3_70B,
// 		"meta.llama3-1-8b-instruct-v1:0":     ai.Llama31_8B,
// 		"meta.llama3-1-70b-instruct-v1:0":    ai.Llama31_70B,
// 		"meta.llama3-2-1b-instruct-v1:0":     ai.Llama32_1B,
// 		"meta.llama3-2-3b-instruct-v1:0":     ai.Llama32_3B,
// 		"meta.llama3-2-11b-instruct-v1:0":    ai.Llama32_11B,
// 		"meta.llama3-2-90b-instruct-v1:0":    ai.Llama32_90B,
// 		"meta.llama3-3-70b-instruct-v1:0":    ai.Llama33_70B,
// 		"us.meta.llama3-3-70b-instruct-v1:0": ai.Llama33_70B,

// 		// Mistral models on Bedrock
// 		"mistral.mistral-7b-instruct-v0:2":   ai.Mistral7B,
// 		"mistral.mixtral-8x7b-instruct-v0:1": ai.Mixtral8x7B,
// 		"mistral.mistral-large-2402-v1:0":    ai.MistralLarge,
// 		"mistral.mistral-small-2402-v1:0":    ai.MistralSmall,

// 		// Amazon Titan models
// 		"amazon.titan-tg1-large":            ai.TitanTextG1Large,
// 		"amazon.titan-text-premier-v1:0":    ai.TitanTextPremier,
// 		"amazon.titan-text-lite-v1":         ai.TitanTextLite,
// 		"amazon.titan-text-lite-v1:0":       ai.TitanTextLite,
// 		"amazon.titan-text-lite-v1:0:4k":    ai.TitanTextLite,
// 		"amazon.titan-text-express-v1":      ai.TitanTextExpress,
// 		"amazon.titan-text-express-v1:0":    ai.TitanTextExpress,
// 		"amazon.titan-text-express-v1:0:8k": ai.TitanTextExpress,
// 		"amazon.titan-embed-text-v1":        ai.TitanEmbedText,
// 		"amazon.titan-embed-text-v1:2":      ai.TitanEmbedText,
// 		"amazon.titan-embed-text-v1:2:8k":   ai.TitanEmbedText,
// 		"amazon.titan-embed-text-v2:0":      ai.TitanEmbedText,
// 		"amazon.titan-embed-text-v2:0:8k":   ai.TitanEmbedText,
// 		"amazon.titan-embed-g1-text-02":     ai.TitanEmbedText,
// 		"amazon.titan-embed-image-v1:0":     ai.TitanEmbedImage,

// 		// Amazon Nova models
// 		"amazon.nova-pro-v1:0":        ai.NovaPro,
// 		"amazon.nova-pro-v1:0:300k":   ai.NovaPro,
// 		"amazon.nova-lite-v1:0":       ai.NovaLite,
// 		"amazon.nova-lite-v1:0:300k":  ai.NovaLite,
// 		"amazon.nova-canvas-v1:0":     ai.NovaCanvas,
// 		"amazon.nova-reel-v1:0":       ai.NovaReel,
// 		"amazon.nova-micro-v1:0":      ai.NovaMicro,
// 		"amazon.nova-micro-v1:0:128k": ai.NovaMicro,
// 	},

// 	ai.Google: {
// 		// Gemini models
// 		"gemini-2.5-flash":               ai.Gemini25Flash,
// 		"gemini-2.5-flash-preview-04-17": ai.Gemini25Flash,
// 		"gemini-2.5-pro":                 ai.Gemini25Pro,
// 		"gemini-2.5-pro-preview":         ai.Gemini25Pro,
// 		"gemini-2.5-pro-preview-05-06":   ai.Gemini25Pro,
// 		"gemini-2.0-flash":               ai.Gemini20Flash,
// 		"gemini-2.0-flash-lite":          ai.Gemini20FlashLite,
// 		"gemini-2.0-flash-live":          ai.Gemini20Flash,
// 		"gemini-2.0-flash-live-001":      ai.Gemini20Flash,
// 		"gemini-1.5-flash":               ai.Gemini15Flash,
// 		"gemini-1.5-flash-8b":            ai.Gemini15Flash8B,
// 		"gemini-1.5-pro":                 ai.Gemini15Pro,

// 		// Embedding models
// 		"text-embedding-004":   ai.GeminiEmbedding,
// 		"gemini-embedding-exp": ai.GeminiEmbedding,

// 		// Legacy models
// 		"palm-2": ai.PaLM2,
// 	},

// 	ai.XAI: {
// 		"grok-4":      ai.Grok4,
// 		"grok-4-0709": ai.Grok4,
// 		"grok-3":      ai.Grok3,
// 		"grok-3-mini": ai.Grok3Mini,
// 	},
// }
