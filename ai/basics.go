package ai

import (
	"errors"
	"fmt"
	"log"
	"sync"

	"github.com/MelloB1989/karma/config"
	internalopenai "github.com/MelloB1989/karma/internal/openai"
	"github.com/openai/openai-go/v3/shared"
	"github.com/posthog/posthog-go"
)

// BaseModel represents the core model without provider-specific naming
type BaseModel string

// Provider represents the inference provider
type Provider string

// Base Models - Core models without provider prefixes
const (
	// OpenAI Models
	GPT4       BaseModel = "gpt-4"
	GPT4o      BaseModel = "gpt-4o"
	GPT4oMini  BaseModel = "gpt-4o-mini"
	GPT4Turbo  BaseModel = "gpt-4-turbo"
	GPT35Turbo BaseModel = "gpt-3.5-turbo"
	GPT5       BaseModel = "gpt-5"
	GPT5Nano   BaseModel = "gpt-5-nano"
	GPT5Mini   BaseModel = "gpt-5-mini"
	GPT5_1     BaseModel = "gpt-5.1"
	GPT5_2     BaseModel = "gpt-5.2"
	GPT5_2_Pro BaseModel = "gpt-5.2-pro"

	GPT5_1Codex    BaseModel = "gpt-5.1-codex"
	GPT5_1CodexMax BaseModel = "gpt-5.1-codex-max"
	GPT5_2Codex    BaseModel = "gpt-5.2-codex"
	GPT5_2CodexMax BaseModel = "gpt-5.2-codex-max"
	O1             BaseModel = "o1"
	O1Mini         BaseModel = "o1-mini"
	O1Preview      BaseModel = "o1-preview"
	GPTOSS_20B     BaseModel = "gpt-oss-20b"
	GPTOSS_120B    BaseModel = "gpt-oss-120b"

	// Text Embedding Models
	TextEmbeddingAda002 BaseModel = "text-embedding-ada-002"
	TextEmbedding3Large BaseModel = "text-embedding-3-large"
	TextEmbedding3Small BaseModel = "text-embedding-3-small"

	// Claude Models
	Claude35Sonnet  BaseModel = "claude-3.5-sonnet"
	Claude35Haiku   BaseModel = "claude-3.5-haiku"
	Claude3Sonnet   BaseModel = "claude-3-sonnet"
	Claude3Haiku    BaseModel = "claude-3-haiku"
	Claude3Opus     BaseModel = "claude-3-opus"
	Claude37Sonnet  BaseModel = "claude-3.7-sonnet"
	Claude4Sonnet   BaseModel = "claude-4-sonnet"
	Claude4_5Sonnet BaseModel = "claude-4.5-sonnet"
	Claude4Opus     BaseModel = "claude-4-opus"
	Claude4_5Opus   BaseModel = "claude-4.5-opus"
	ClaudeInstant   BaseModel = "claude-instant"
	ClaudeV2        BaseModel = "claude-v2"

	// Llama Models
	Llama3_8B        BaseModel = "llama-3-8b"
	Llama3_70B       BaseModel = "llama-3-70b"
	Llama31_8B       BaseModel = "llama-3.1-8b"
	Llama31_70B      BaseModel = "llama-3.1-70b"
	Llama31_405B     BaseModel = "llama-3.1-405b"
	Llama32_1B       BaseModel = "llama-3.2-1b"
	Llama32_3B       BaseModel = "llama-3.2-3b"
	Llama32_11B      BaseModel = "llama-3.2-11b"
	Llama32_90B      BaseModel = "llama-3.2-90b"
	Llama33_70B      BaseModel = "llama-3.3-70b"
	Llama4_Guard_12B BaseModel = "llama-4-guard-12b"
	Llama4_Scout_17B BaseModel = "llama-4-scout-17b"

	// Mistral Models
	Mistral7B    BaseModel = "mistral-7b"
	Mixtral8x7B  BaseModel = "mixtral-8x7b"
	MistralLarge BaseModel = "mistral-large"
	MistralSmall BaseModel = "mistral-small"

	// Quew Models
	Quew3_32B                 BaseModel = "quew3-32b"
	Quew3_235B_VL_Thinking    BaseModel = "quew3-235b-vl"
	Quew3_235B_VL_Instruct    BaseModel = "quew3-235b-vl-instruct"
	Quew3_235B_Thinking       BaseModel = "quew3-235b-thinking"
	Quew3_235B_Instruct       BaseModel = "quew3-235b-instruct"
	Qwen3_Coder_480B_Instruct BaseModel = "qwen3-coder-480b-instruct"

	// Moonshot Models
	KimiK2Thinking BaseModel = "kimi-k2-thinking"

	MiniMaxM2   BaseModel = "minimax-m2"
	MiniMaxM2P1 BaseModel = "minimax-m2p1"

	GLM4_7 BaseModel = "glm4-7"
	GLM4_6 BaseModel = "glm4-6"

	// Deepseek
	DeepSeekV3P2 BaseModel = "deepseek-v3p2"

	// Amazon Titan Models
	TitanTextG1Large BaseModel = "titan-text-g1-large"
	TitanTextPremier BaseModel = "titan-text-premier"
	TitanTextLite    BaseModel = "titan-text-lite"
	TitanTextExpress BaseModel = "titan-text-express"
	TitanEmbedText   BaseModel = "titan-embed-text"
	TitanEmbedImage  BaseModel = "titan-embed-image"

	// Amazon Nova Models
	NovaPro    BaseModel = "nova-pro"
	NovaLite   BaseModel = "nova-lite"
	NovaCanvas BaseModel = "nova-canvas"
	NovaReel   BaseModel = "nova-reel"
	NovaMicro  BaseModel = "nova-micro"

	// Google Models
	Gemini3FlashPreview BaseModel = "gemini-3-flash-preview"
	Gemini3ProPreview   BaseModel = "gemini-3-pro-preview"
	Gemini25Flash       BaseModel = "gemini-2.5-flash"
	Gemini25Pro         BaseModel = "gemini-2.5-pro"
	Gemini20Flash       BaseModel = "gemini-2.0-flash"
	Gemini20FlashLite   BaseModel = "gemini-2.0-flash-lite"
	Gemini15Flash       BaseModel = "gemini-1.5-flash"
	Gemini15Flash8B     BaseModel = "gemini-1.5-flash-8b"
	Gemini15Pro         BaseModel = "gemini-1.5-pro"
	GeminiEmbedding     BaseModel = "gemini-embedding"
	PaLM2               BaseModel = "palm-2"

	// xAI Models
	Grok4              BaseModel = "grok-4"
	GrokCodeFast       BaseModel = "grok-code-fast-1"
	Grok4ReasoningFast BaseModel = "grok-4-fast-reasoning"
	Grok4Fast          BaseModel = "grok-4-fast-non-reasoning"
	Grok3              BaseModel = "grok-3"
	Grok3Mini          BaseModel = "grok-3-mini"

	// Sarvam AI Models
	SarvamM BaseModel = "sarvam-m"
)

// Providers
const (
	OpenAI      Provider = "openai"
	Anthropic   Provider = "anthropic"
	Bedrock     Provider = "bedrock"
	Google      Provider = "google"
	XAI         Provider = "xai"
	Groq        Provider = "groq"
	FireworksAI Provider = "fireworksai"
	OpenRouter  Provider = "openrouter"
	Sarvam      Provider = "sarvam"
)

// API URLs for different providers
const (
	XAI_API        = "https://api.x.ai/v1"
	GROQ_API       = "https://api.groq.com/openai/v1"
	SARVAM_API     = "https://api.sarvam.ai/v1"
	FIREWORKS_API  = "https://api.fireworks.ai/inference/v1"
	OPENROUTER_API = "https://openrouter.ai/api/v1"
)

// ModelConfig represents a model with its provider configuration
type ModelConfig struct {
	BaseModel         BaseModel
	Provider          Provider
	CustomModelString string // Optional: override the provider-specific model string
}

type providerMap map[Provider]map[BaseModel]string

var (
	ProviderModelMapping providerMap = map[Provider]map[BaseModel]string{
		OpenAI: {
			GPT4:                "gpt-4",
			GPT4o:               "gpt-4o",
			GPT4oMini:           "gpt-4o-mini",
			GPT4Turbo:           "gpt-4-turbo",
			GPT35Turbo:          "gpt-3.5-turbo",
			GPT5:                "gpt-5",
			GPT5Nano:            "gpt-5-nano",
			GPT5Mini:            "gpt-5-mini",
			O1:                  "o1",
			O1Mini:              "o1-mini",
			O1Preview:           "o1-preview",
			TextEmbeddingAda002: "text-embedding-ada-002",
			TextEmbedding3Large: "text-embedding-3-large",
			TextEmbedding3Small: "text-embedding-3-small",
		},
		Anthropic: {
			ClaudeInstant:  "claude-instant-v1.2",
			ClaudeV2:       "claude-v2.1",
			Claude3Sonnet:  "claude-3-sonnet-20240229",
			Claude3Haiku:   "claude-3-haiku-20240307",
			Claude3Opus:    "claude-3-opus-20240229",
			Claude35Sonnet: "claude-3.5-sonnet-20241022",
			Claude35Haiku:  "claude-3.5-haiku-20241022",
			Claude37Sonnet: "claude-3.7-sonnet-20250219",
			Claude4Sonnet:  "claude-4-sonnet-20250514",
			Claude4Opus:    "claude-4-opus-20250514",
		},
		Bedrock: {
			// Claude models on Bedrock
			ClaudeInstant:  "anthropic.claude-instant-v1",
			ClaudeV2:       "anthropic.claude-v2:1",
			Claude3Sonnet:  "anthropic.claude-3-sonnet-20240229-v1:0",
			Claude3Haiku:   "anthropic.claude-3-haiku-20240307-v1:0",
			Claude3Opus:    "anthropic.claude-3-opus-20240229-v1:0",
			Claude35Sonnet: "anthropic.claude-3-5-sonnet-20241022-v2:0",
			Claude35Haiku:  "anthropic.claude-3-5-haiku-20241022-v1:0",
			Claude37Sonnet: "us.anthropic.claude-3-7-sonnet-20250219-v1:0",
			// Llama models on Bedrock
			Llama3_8B:   "meta.llama3-8b-instruct-v1:0",
			Llama3_70B:  "meta.llama3-70b-instruct-v1:0",
			Llama31_8B:  "meta.llama3-1-8b-instruct-v1:0",
			Llama31_70B: "meta.llama3-1-70b-instruct-v1:0",
			Llama32_1B:  "meta.llama3-2-1b-instruct-v1:0",
			Llama32_3B:  "meta.llama3-2-3b-instruct-v1:0",
			Llama32_11B: "meta.llama3-2-11b-instruct-v1:0",
			Llama32_90B: "meta.llama3-2-90b-instruct-v1:0",
			Llama33_70B: "meta.llama3-3-70b-instruct-v1:0",
			// Mistral models on Bedrock
			Mistral7B:    "mistral.mistral-7b-instruct-v0:2",
			Mixtral8x7B:  "mistral.mixtral-8x7b-instruct-v0:1",
			MistralLarge: "mistral.mistral-large-2402-v1:0",
			MistralSmall: "mistral.mistral-small-2402-v1:0",
			// Amazon Titan models
			TitanTextG1Large: "amazon.titan-tg1-large",
			TitanTextPremier: "amazon.titan-text-premier-v1:0",
			TitanTextLite:    "amazon.titan-text-lite-v1:0",
			TitanTextExpress: "amazon.titan-text-express-v1:0",
			TitanEmbedText:   "amazon.titan-embed-text-v1:2",
			TitanEmbedImage:  "amazon.titan-embed-image-v1:0",
			// Amazon Nova models
			NovaPro:    "amazon.nova-pro-v1:0",
			NovaLite:   "amazon.nova-lite-v1:0",
			NovaCanvas: "amazon.nova-canvas-v1:0",
			NovaReel:   "amazon.nova-reel-v1:0",
			NovaMicro:  "amazon.nova-micro-v1:0",
		},
		Google: {
			Gemini3FlashPreview: "gemini-3-flash-preview",
			Gemini3ProPreview:   "gemini-3-pro-preview",
			Gemini25Flash:       "gemini-2.5-flash",
			Gemini25Pro:         "gemini-2.5-pro",
			Gemini20Flash:       "gemini-2.0-flash",
			Gemini20FlashLite:   "gemini-2.0-flash-lite",
			Gemini15Flash:       "gemini-1.5-flash",
			Gemini15Flash8B:     "gemini-1.5-flash-8b",
			Gemini15Pro:         "gemini-1.5-pro",
			GeminiEmbedding:     "text-embedding-004",
			PaLM2:               "palm-2",

			// Meta
			Llama4_Scout_17B: "meta/llama-4-maverick-17b-128e-instruct-maas",
			Llama33_70B:      "meta/llama-3.3-70b-instruct-maas",
			Llama32_90B:      "meta/llama-3.2-90b-vision-instruct-maas",
			Llama31_405B:     "meta/llama-3.1-405b-instruct-maas",

			// Moonshot
			KimiK2Thinking: "moonshotai/kimi-k2-thinking-maas",

			MiniMaxM2: "minimaxai/minimax-m2-maas",

			// OpenAI
			GPTOSS_120B: "openai/gpt-oss-120b-maas",
			GPTOSS_20B:  "openai/gpt-oss-20b-maas",
		},
		XAI: {
			Grok4:              "grok-4",
			Grok4Fast:          "grok-4-fast-non-reasoning",
			Grok4ReasoningFast: "grok-4-fast-reasoning",
			GrokCodeFast:       "grok-code-fast-1",
			Grok3:              "grok-3",
			Grok3Mini:          "grok-3-mini",
		},
		Groq: {
			Llama31_8B:       "llama-3.1-8b-instant",
			Llama33_70B:      "llama-3.3-70b-versatile",
			Llama4_Guard_12B: "meta-llama/llama-guard-4-12b",
			Llama4_Scout_17B: "meta-llama/llama-4-scout-17b-16e-instruct",
			GPTOSS_120B:      "openai/gpt-oss-120b",
			GPTOSS_20B:       "openai/gpt-oss-20b",
			Quew3_32B:        "qwen/qwen3-32b",
		},
		Sarvam: {
			SarvamM: "sarvam-m",
		},
		// Fireworks AI Serverless models
		FireworksAI: {
			MiniMaxM2P1:               "accounts/fireworks/models/minimax-m2p1",
			MiniMaxM2:                 "accounts/fireworks/models/minimax-m2",
			GLM4_7:                    "accounts/fireworks/models/glm-4p7",
			DeepSeekV3P2:              "accounts/fireworks/models/deepseek-v3p2",
			KimiK2Thinking:            "accounts/fireworks/models/kimi-k2-thinking",
			GLM4_6:                    "accounts/fireworks/models/glm-4p6",
			Quew3_235B_VL_Thinking:    "accounts/fireworks/models/qwen3-vl-235b-a22b-thinking",
			Quew3_235B_VL_Instruct:    "accounts/fireworks/models/qwen3-vl-235b-a22b-instruct",
			Qwen3_Coder_480B_Instruct: "accounts/fireworks/models/qwen3-coder-480b-a35b-instruct",
			Quew3_235B_Thinking:       "accounts/fireworks/models/qwen3-235b-a22b-thinking-2507",
			Quew3_235B_Instruct:       "accounts/fireworks/models/qwen3-coder-480b-a35b-instruct",
			GPTOSS_120B:               "accounts/fireworks/models/gpt-oss-120b",
			GPTOSS_20B:                "accounts/fireworks/models/gpt-oss-20b",
			Llama33_70B:               "accounts/fireworks/models/llama-v3p3-70b-instruct",
		},
		// Common model mappings for OpenRouter
		OpenRouter: {
			GPTOSS_120B:               "openai/gpt-oss-120b",
			GPTOSS_20B:                "openai/gpt-oss-20b",
			GPT5Mini:                  "openai/gpt-5-mini",
			GPT5Nano:                  "openai/gpt-5-nano",
			GPT5_1:                    "openai/gpt-5.1",
			GPT5_2:                    "openai/gpt-5.2",
			GPT5_2_Pro:                "openai/gpt-5.2-pro",
			GPT5_1Codex:               "openai/gpt-5.1-codex",
			GPT5_1CodexMax:            "openai/gpt-5.1-codex-max",
			GPT5_2Codex:               "openai/gpt-5.2-codex",
			GPT5_2CodexMax:            "openai/gpt-5.2-codex-max",
			Claude4_5Opus:             "anthropic/claude-sonnet-4.5",
			Claude4_5Sonnet:           "anthropic/claude-sonnet-4.5",
			MiniMaxM2P1:               "minimax/minimax-m2.1",
			MiniMaxM2:                 "minimax/minimax-m2",
			Gemini3FlashPreview:       "google/gemini-3-flash-preview",
			Gemini3ProPreview:         "google/gemini-3-pro-preview",
			Gemini25Pro:               "google/gemini-2.5-pro",
			Gemini25Flash:             "google/gemini-2.5-flash",
			Gemini20FlashLite:         "google/gemini-2.5-flash-lite",
			KimiK2Thinking:            "moonshotai/kimi-k2-thinking",
			Llama33_70B:               "meta-llama/llama-3.3-70b-instruct",
			Llama4_Scout_17B:          "meta-llama/llama-4-scout",
			Qwen3_Coder_480B_Instruct: "qwen/qwen3-coder",
			Quew3_235B_Instruct:       "qwen/qwen3-235b-a22b-2507",
			Quew3_235B_VL_Instruct:    "qwen/qwen3-vl-235b-a22b-instruct",
			Quew3_235B_Thinking:       "qwen/qwen3-vl-235b-a22b-thinking",
		},
	}
)

// GetModelString returns the provider-specific model string for API calls
func (mc ModelConfig) GetModelString() string {
	// If custom model string is provided, use it
	if mc.CustomModelString != "" {
		return mc.CustomModelString
	}

	provider := mc.Provider

	// Use canonical mapping for the provider
	canonicalModel := getCanonicalModelString(mc.BaseModel, provider)
	if canonicalModel != "" {
		return canonicalModel
	}

	// Fallback to base model string
	return string(mc.BaseModel)
}

// GetProvider returns the provider for this model config
func (mc ModelConfig) GetProvider() Provider {
	if mc.Provider != "" {
		return mc.Provider
	}
	return ""
}

// IsOpenAICompatibleModel checks if the model is OpenAI API compatible
func (mc ModelConfig) IsOpenAICompatibleModel() bool {
	provider := mc.GetProvider()
	return provider == OpenAI || provider == XAI || provider == Groq
}

// SupportsMCP checks if the model supports MCP
func (mc ModelConfig) SupportsMCP() bool {
	return mc.Provider == OpenAI || mc.Provider == XAI || mc.Provider == Anthropic
}

// GetModelProvider returns the provider for a given model config
func (mc ModelConfig) GetModelProvider() Provider {
	return mc.GetProvider()
}

func getCanonicalModelString(baseModel BaseModel, provider Provider) string {
	if providerMappings, exists := ProviderModelMapping[provider]; exists {
		if modelString, found := providerMappings[baseModel]; found {
			return modelString
		}
	}

	return ""
}

// MCPTool represents an MCP tool configuration
type MCPTool struct {
	FriendlyName string `json:"friendly_name"`
	ToolName     string `json:"tool_name"`
	Description  string `json:"description"`
	InputSchema  any    `json:"input_schema"`
}

// MCPServer represents an MCP server configuration
type MCPServer struct {
	URL       string    `json:"url"`
	AuthToken string    `json:"auth_token,omitempty"`
	Tools     []MCPTool `json:"tools"`
}

// Analytics represents analytics configuration
type Analytics struct {
	DistinctID         string         `json:"distinct_id"`
	TraceId            string         `json:"trace_id"`
	CaptureUserPrompts bool           `json:"capture_user_prompts"`
	CaptureAIResponses bool           `json:"capture_ai_responses"`
	CaptureToolCalls   bool           `json:"capture_tool_calls"`
	on                 bool           `json:"-"`
	client             posthog.Client `json:"-"`
	properties         map[string]any `json:"-"`
	mu                 sync.RWMutex   `json:"-"`
}

type SpecialConfig string

const (
	GoogleProjectID SpecialConfig = "google_project_id"
	GoogleLocation  SpecialConfig = "google_location"
	GoogleAPIKey    SpecialConfig = "google_api_key"
)

// KarmaAI represents the main AI configuration
type KarmaAI struct {
	Model           ModelConfig                     `json:"model"`
	SystemMessage   string                          `json:"system_message"`
	Context         string                          `json:"context"`
	UserPrePrompt   string                          `json:"user_pre_prompt"`
	Temperature     float32                         `json:"temperature"`
	TopP            float32                         `json:"top_p"`
	TopK            int                             `json:"top_k"`
	MaxTokens       int                             `json:"max_tokens"`
	ReasoningEffort *shared.ReasoningEffort         `json:"reasoning_effort"`
	ResponseType    string                          `json:"response_type"`
	MCPConfig       map[string]MCPTool              `json:"mcp_config"`
	MCPUrl          string                          `json:"mcp_url"`
	AuthToken       string                          `json:"auth_token"`
	MCPTools        []MCPTool                       `json:"mcp_tools"`
	GoFunctionTools []internalopenai.GoFunctionTool `json:"go_function_tools"`
	ToolsEnabled    bool                            `json:"tools_enabled"`
	UseMCPExecution bool                            `json:"use_mcp_execution"`
	Analytics       *Analytics                      `json:"analytics"`
	Features        *F                              `json:"features"`
	MaxToolPasses   int                             `json:"max_tool_passes"`
	// Deprecated: Use MCPServers instead
	MCPServers []MCPServer `json:"mcp_servers"`
	// Provider-specific configuration
	SpecialConfig map[SpecialConfig]any `json:"special_config"`
}

type F struct {
	optionalFields map[string]any
}

// Option represents a configuration option for KarmaAI
type Option func(*KarmaAI)

// WithSystemMessage sets the system message
func WithSystemMessage(message string) Option {
	return func(kai *KarmaAI) {
		kai.SystemMessage = message
	}
}

func WithSpecialConfig(config map[SpecialConfig]any) Option {
	return func(kai *KarmaAI) {
		kai.SpecialConfig = config
	}
}

// WithContext sets the context
func WithContext(context string) Option {
	return func(kai *KarmaAI) {
		kai.Context = context
	}
}

// WithUserPrePrompt sets the user pre-prompt
func WithUserPrePrompt(prompt string) Option {
	return func(kai *KarmaAI) {
		kai.UserPrePrompt = prompt
	}
}

// WithTemperature sets the temperature
func WithTemperature(temp float32) Option {
	return func(kai *KarmaAI) {
		kai.Temperature = temp
	}
}

// WithMaxTokens sets the maximum tokens
func WithMaxTokens(tokens int) Option {
	return func(kai *KarmaAI) {
		kai.MaxTokens = tokens
	}
}

// WithTopP sets the top-p value
func WithTopP(topP float32) Option {
	return func(kai *KarmaAI) {
		kai.TopP = topP
	}
}

// WithTopK sets the top-k value
func WithTopK(topK int) Option {
	return func(kai *KarmaAI) {
		kai.TopK = topK
	}
}

// WithReasoningEffort sets the reasoning effort for supported models
func WithReasoningEffort(effort shared.ReasoningEffort) Option {
	return func(kai *KarmaAI) {
		kai.ReasoningEffort = &effort
	}
}

// WithResponseType sets the response type
func WithResponseType(responseType string) Option {
	return func(kai *KarmaAI) {
		kai.ResponseType = responseType
	}
}

func WithMaxToolPasses(max int) Option {
	return func(kai *KarmaAI) {
		kai.MaxToolPasses = max
	}
}

// SetMCPTools sets the MCP tools
func SetMCPTools(tools []MCPTool) Option {
	return func(kai *KarmaAI) {
		kai.MCPTools = tools
		if kai.MCPConfig == nil {
			kai.MCPConfig = make(map[string]MCPTool)
		}
		for _, tool := range tools {
			kai.MCPConfig[tool.ToolName] = tool
		}
		// Also populate tools in any existing MCPServers
		for i := range kai.MCPServers {
			kai.MCPServers[i].Tools = tools
		}
	}
}

func SetGoFunctionTools(tools []internalopenai.GoFunctionTool) Option {
	return func(kai *KarmaAI) {
		kai.GoFunctionTools = tools
	}
}

func AddGoFunctionTool(tool internalopenai.GoFunctionTool) Option {
	return func(kai *KarmaAI) {
		kai.GoFunctionTools = append(kai.GoFunctionTools, tool)
	}
}

// AddGoFunctionTool adds a Go function tool to the KarmaAI instance after construction.
// Returns an error if the tool name or handler is missing.
func (kai *KarmaAI) AddGoFunctionTool(tool internalopenai.GoFunctionTool) error {
	if tool.Name == "" {
		return fmt.Errorf("tool name required")
	}
	if tool.Handler == nil {
		return fmt.Errorf("tool handler required")
	}
	kai.GoFunctionTools = append(kai.GoFunctionTools, tool)
	return nil
}

// ClearGoFunctionTools removes all Go function tools from the KarmaAI instance.
func (kai *KarmaAI) ClearGoFunctionTools() {
	kai.GoFunctionTools = nil
}

// SetMCPServers sets the MCP servers
func SetMCPServers(servers []MCPServer) Option {
	return func(kai *KarmaAI) {
		kai.MCPServers = servers
		var allTools []MCPTool
		for _, server := range servers {
			allTools = append(allTools, server.Tools...)
		}
		kai.MCPTools = allTools
	}
}

// NewMCPServer creates a new MCP server configuration
func NewMCPServer(url, authToken string, tools []MCPTool) MCPServer {
	return MCPServer{
		URL:       url,
		AuthToken: authToken,
		Tools:     tools,
	}
}

// AddMCPServer adds an MCP server to the configuration
func AddMCPServer(server MCPServer) Option {
	return func(kai *KarmaAI) {
		kai.MCPServers = append(kai.MCPServers, server)
		kai.MCPTools = append(kai.MCPTools, server.Tools...)
		if kai.MCPConfig == nil {
			kai.MCPConfig = make(map[string]MCPTool)
		}
		for _, tool := range server.Tools {
			kai.MCPConfig[tool.ToolName] = tool
		}
	}
}

// SetMCPUrl sets the MCP URL
func SetMCPUrl(url string) Option {
	return func(kai *KarmaAI) {
		kai.MCPUrl = url
		// If no servers exist, create a default one
		if len(kai.MCPServers) == 0 {
			kai.MCPServers = append(kai.MCPServers, MCPServer{URL: url})
		}
	}
}

// SetMCPAuthToken sets the MCP auth token
func SetMCPAuthToken(token string) Option {
	return func(kai *KarmaAI) {
		kai.AuthToken = token
		// Update all servers with the auth token
		for i := range kai.MCPServers {
			kai.MCPServers[i].AuthToken = token
		}
	}
}

// Use a custom model variant
func SetCustomModelVariant(m string) Option {
	return func(kai *KarmaAI) {
		kai.Model.CustomModelString = m
	}
}

// ConfigureAnalytics configures analytics settings
func ConfigureAnalytics(distinctID, traceID string) Option {
	return func(kai *KarmaAI) {
		if kai.Analytics == nil {
			kai.Analytics = &Analytics{
				properties: make(map[string]any),
			}
		}
		client, err := posthog.NewWithConfig(config.GetEnvRaw("POSTHOG_KEY"), posthog.Config{Endpoint: config.GetEnvRaw("POSTHOG_ENDPOINT")})
		if err != nil {
			log.Println("Failed to initialize posthog client!")
		}
		if client != nil {
			kai.Analytics.client = client
		}
		kai.Analytics.DistinctID = distinctID
		kai.Analytics.TraceId = traceID
		kai.Analytics.CaptureUserPrompts = true
		kai.Analytics.CaptureAIResponses = true
		kai.Analytics.CaptureToolCalls = true
		kai.Analytics.on = true
	}
}

// EnableTools enables tool usage
func (kai *KarmaAI) EnableTools() {
	kai.ToolsEnabled = true
}

func (kai *KarmaAI) GetSpecialConfig(c SpecialConfig) (any, error) {
	if kai.SpecialConfig == nil {
		return nil, errors.New("special config not initialized")
	}
	return kai.SpecialConfig[c], nil
}

// WithToolsEnabled enables MCP tools
func WithToolsEnabled() Option {
	return func(kai *KarmaAI) {
		kai.ToolsEnabled = true
	}
}

// WithDirectToolCalls enables tools without MCP execution (for LangChain/n8n)
func WithDirectToolCalls() Option {
	return func(kai *KarmaAI) {
		kai.ToolsEnabled = true
		kai.UseMCPExecution = false
	}
}

// NewKarmaAI creates a new KarmaAI instance with the specified model and options
func NewKarmaAI(baseModel BaseModel, provider Provider, options ...Option) *KarmaAI {
	kai := &KarmaAI{
		Model: ModelConfig{
			BaseModel: baseModel,
			Provider:  provider,
		},
		Temperature:     1,
		TopP:            0.9,
		TopK:            40,
		MaxTokens:       1024,
		MCPConfig:       make(map[string]MCPTool),
		ToolsEnabled:    false,
		UseMCPExecution: true,
		Analytics:       &Analytics{},
		Features: &F{
			optionalFields: make(map[string]any),
		},
		MaxToolPasses: 4,
		SpecialConfig: make(map[SpecialConfig]any),
	}

	for _, option := range options {
		option(kai)
	}

	return kai
}
