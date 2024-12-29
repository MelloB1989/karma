package ai

import (
	"errors"
	"strings"

	"github.com/MelloB1989/karma/internal/openai"
	"github.com/MelloB1989/karma/models"
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
	Claude2       Models = "claude-2"
	Claude1       Models = "claude-1"
	ClaudeInstant Models = "claude-instant-1"

	// Meta's LLaMA Models
	Llama2_7B       Models = "llama-2-7b"
	Llama2_13B      Models = "llama-2-13b"
	Llama2_70B      Models = "llama-2-70b"
	Llama2_7B_Chat  Models = "llama-2-7b-chat"
	Llama2_13B_Chat Models = "llama-2-13b-chat"
	Llama2_70B_Chat Models = "llama-2-70b-chat"

	// Mistral AI Models
	Mistral7B            Models = "mistral-7b"
	Mixtral8x7B          Models = "mixtral-8x7b"
	Mistral7B_Instruct   Models = "mistral-7b-instruct"
	Mixtral8x7B_Instruct Models = "mixtral-8x7b-instruct"

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

// GetModelProvider returns the provider of the model
func (m Models) GetModelProvider() ModelProviders {
	switch {
	case m.IsOpenAIModel():
		return OpenAI
	case m.IsAnthropicModel():
		return Anthropic
	default:
		return Undefined
	}
}

type KarmaAI struct {
	Model          Models
	SystemMessage  string
	Context        string
	UserPrePrompt  string
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
		chat, err := openai.CreateChat(messages, string(kai.Model))
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
