package ai

import (
	"errors"
	"strings"

	"github.com/MelloB1989/karma/internal/aws/bedrock_runtime"
	"github.com/MelloB1989/karma/internal/openai"
	"github.com/MelloB1989/karma/models"
	oai "github.com/openai/openai-go"
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

	// Anthropic Models
	ClaudeInstantV1_2_100K       Models = "anthropic.claude-instant-v1:2:100k"
	ClaudeInstantV1              Models = "anthropic.claude-instant-v1"
	ClaudeV2_0_18K               Models = "anthropic.claude-v2:0:18k"
	ClaudeV2_0_100K              Models = "anthropic.claude-v2:0:100k"
	ClaudeV2_1_18K               Models = "anthropic.claude-v2:1:18k"
	ClaudeV2_1_200K              Models = "anthropic.claude-v2:1:200k"
	ClaudeV2_1                   Models = "anthropic.claude-v2:1"
	ClaudeV2                     Models = "anthropic.claude-v2"
	Claude3Sonnet20240229V1_28K  Models = "anthropic.claude-3-sonnet-20240229-v1:0:28k"
	Claude3Sonnet20240229V1_200K Models = "anthropic.claude-3-sonnet-20240229-v1:0:200k"
	Claude3Sonnet20240229V1      Models = "anthropic.claude-3-sonnet-20240229-v1:0"
	Claude3Haiku20240307V1_48K   Models = "anthropic.claude-3-haiku-20240307-v1:0:48k"
	Claude3Haiku20240307V1_200K  Models = "anthropic.claude-3-haiku-20240307-v1:0:200k"
	Claude3Haiku20240307V1       Models = "anthropic.claude-3-haiku-20240307-v1:0"
	Claude3Opus20240229V1_12K    Models = "anthropic.claude-3-opus-20240229-v1:0:12k"
	Claude3Opus20240229V1_28K    Models = "anthropic.claude-3-opus-20240229-v1:0:28k"
	Claude3Opus20240229V1_200K   Models = "anthropic.claude-3-opus-20240229-v1:0:200k"
	Claude3Opus20240229V1        Models = "anthropic.claude-3-opus-20240229-v1:0"
	Claude3_5Sonnet20240620V1    Models = "anthropic.claude-3-5-sonnet-20240620-v1:0"
	Claude3_5Sonnet20241022V2    Models = "anthropic.claude-3-5-sonnet-20241022-v2:0"
	Claude3_5Haiku20241022V1     Models = "anthropic.claude-3-5-haiku-20241022-v1:0"

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
	PaLM2       Models = "palm-2"
	Gemini      Models = "gemini-pro"
	GeminiPro   Models = "gemini-pro"
	GeminiUltra Models = "gemini-ultra"

	// Cohere Models
	Command        Models = "command"
	CommandLight   Models = "command-light"
	CommandNightly Models = "command-nightly"

	// Open Source Models
	Falcon40B Models = "falcon-40b"
	MPT7B     Models = "mpt-7b"
	StableLM  Models = "stablelm-base-7b"
	Dolly12B  Models = "dolly-v2-12b"
	BLOOMZ    Models = "bloomz-7b1"

	// Undefined Model
	J2GrandeInstruct         Models = "ai21.j2-grande-instruct"
	J2JumboInstruct          Models = "ai21.j2-jumbo-instruct"
	J2Mid                    Models = "ai21.j2-mid"
	J2MidV1                  Models = "ai21.j2-mid-v1"
	J2Ultra                  Models = "ai21.j2-ultra"
	J2UltraV1_8K             Models = "ai21.j2-ultra-v1:0:8k"
	J2UltraV1                Models = "ai21.j2-ultra-v1"
	JambaInstructV1          Models = "ai21.jamba-instruct-v1:0"
	Jamba1_5LargeV1          Models = "ai21.jamba-1-5-large-v1:0"
	Jamba1_5MiniV1           Models = "ai21.jamba-1-5-mini-v1:0"
	CommandTextV14_7_4K      Models = "cohere.command-text-v14:7:4k"
	CommandTextV14           Models = "cohere.command-text-v14"
	CommandRV1               Models = "cohere.command-r-v1:0"
	CommandRPlusV1           Models = "cohere.command-r-plus-v1:0"
	CommandLightTextV14_7_4K Models = "cohere.command-light-text-v14:7:4k"
	CommandLightTextV14      Models = "cohere.command-light-text-v14"
	EmbedEnglishV3_512       Models = "cohere.embed-english-v3:0:512"
	EmbedEnglishV3           Models = "cohere.embed-english-v3"
	EmbedMultilingualV3_512  Models = "cohere.embed-multilingual-v3:0:512"
	EmbedMultilingualV3      Models = "cohere.embed-multilingual-v3"
)

type ModelProviders string

const (
	OpenAI     ModelProviders = "OpenAI"
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

// IsAnthropicModel checks if the model is from Anthropic
func (m Models) IsAnthropicModel() bool {
	return strings.HasPrefix(string(m), "claude-")
}

// IsMetaModel checks if the model is from Meta
func (m Models) IsMetaModel() bool {
	return strings.HasPrefix(string(m), "llama-") || strings.HasPrefix(string(m), "meta.")
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

type KarmaAI struct {
	Model          Models
	SystemMessage  string
	Context        string
	UserPrePrompt  string
	Temperature    float64
	TopP           float64
	MaxTokens      int64
	StreamResponse bool
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

// WithStreamResponse sets the stream response flag
func WithStreamResponse() Option {
	return func(k *KarmaAI) {
		k.StreamResponse = true
	}
}

// NewKarmaAI creates a new KarmaAI instance with required parameters and optional configurations
func NewKarmaAI(model interface{}, opts ...Option) *KarmaAI {
	modelVal, ok := model.(Models)
	if !ok {
		panic("model must be of type Models")
	}
	karma := &KarmaAI{
		Model:          modelVal,
		StreamResponse: false, // default value
	}

	// Apply all provided options
	for _, opt := range opts {
		opt(karma)
	}

	return karma
}

func (kai *KarmaAI) ChatCompletion(messages models.AIChatHistory) (*models.AIChatResponse, error) {
	//Check if model is OpenAI
	if kai.Model.IsOpenAIModel() {
		o := openai.NewOpenAI(string(kai.Model), kai.Temperature, kai.MaxTokens)
		chat, err := o.CreateChat(messages)
		if err != nil {
			return nil, err
		}
		return &models.AIChatResponse{
			AIResponse: chat.Choices[0].Message.Content,
			Tokens:     int(chat.Usage.TotalTokens),
			TimeTaken:  int(chat.Created),
		}, nil
	} else {
		return nil, errors.New("This model is not supported yet.")
	}
}

func (kai *KarmaAI) GenerateFromSinglePrompt(prompt string) (*models.AIChatResponse, error) {
	//Check if model is OpenAI
	if kai.Model.IsOpenAIModel() {
		o := openai.NewOpenAI(string(kai.Model), kai.Temperature, kai.MaxTokens)
		chat, err := o.CreateChat(models.AIChatHistory{
			Messages: []models.AIMessage{
				{
					Message: kai.UserPrePrompt + " " + prompt,
					Role:    models.User,
				},
			}})
		if err != nil {
			return nil, err
		}
		return &models.AIChatResponse{
			AIResponse: chat.Choices[0].Message.Content,
			Tokens:     int(chat.Usage.TotalTokens),
			TimeTaken:  int(chat.Created),
		}, nil
	} else if kai.Model.IsMetaModel() {
		response, err := bedrock_runtime.InvokeBedrockConverseAPI(string(kai.Model), bedrock_runtime.CreateBedrockRequest(int(kai.MaxTokens), kai.Temperature, kai.TopP, models.AIChatHistory{
			Messages: []models.AIMessage{
				{
					Message: kai.UserPrePrompt + " " + prompt,
					Role:    models.User,
				},
			}}, kai.SystemMessage))
		if err != nil {
			return nil, err
		}
		return &models.AIChatResponse{
			AIResponse: string(response),
			Tokens:     0,
			TimeTaken:  0,
		}, nil
	} else {
		return nil, errors.New("This model is not supported yet.")
	}
}

func (kai *KarmaAI) ChatCompletionStream(messages models.AIChatHistory, chunkHandler func(chuck oai.ChatCompletionChunk)) (*models.AIChatResponse, error) {
	//Check if model is OpenAI
	if kai.Model.IsOpenAIModel() {
		o := openai.NewOpenAI(string(kai.Model), kai.Temperature, kai.MaxTokens)
		chat, err := o.CreateChatStream(messages, chunkHandler)
		if err != nil {
			return nil, err
		}
		return &models.AIChatResponse{
			AIResponse: chat.Choices[0].Message.Content,
			Tokens:     int(chat.Usage.TotalTokens),
			TimeTaken:  int(chat.Created),
		}, nil
	} else {
		return nil, errors.New("This model is not supported yet.")
	}
}
