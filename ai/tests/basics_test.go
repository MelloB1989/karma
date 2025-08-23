package tests

import (
	"reflect"
	"testing"

	"github.com/MelloB1989/karma/ai"
	"github.com/MelloB1989/karma/ai/variants"
)

func TestModelConfig_GetModelString(t *testing.T) {
	tests := []struct {
		name     string
		config   ai.ModelConfig
		expected string
	}{
		{
			name: "Custom model string takes precedence",
			config: ai.ModelConfig{
				BaseModel:         ai.GPT4o,
				Provider:          ai.OpenAI,
				CustomModelString: "custom-model-v1",
			},
			expected: "custom-model-v1",
		},
		{
			name: "Valid OpenAI model",
			config: ai.ModelConfig{
				BaseModel: ai.GPT4o,
				Provider:  ai.OpenAI,
			},
			expected: "gpt-4o",
		},
		{
			name: "Valid Anthropic model",
			config: ai.ModelConfig{
				BaseModel: ai.Claude35Sonnet,
				Provider:  ai.Anthropic,
			},
			expected: "claude-3.5-sonnet-20241022",
		},
		{
			name: "Valid Bedrock model",
			config: ai.ModelConfig{
				BaseModel: ai.Llama3_8B,
				Provider:  ai.Bedrock,
			},
			expected: "meta.llama3-8b-instruct-v1:0",
		},
		{
			name: "Valid Google model",
			config: ai.ModelConfig{
				BaseModel: ai.Gemini25Flash,
				Provider:  ai.Google,
			},
			expected: "gemini-2.5-flash",
		},
		{
			name: "Valid XAI model",
			config: ai.ModelConfig{
				BaseModel: ai.Grok3,
				Provider:  ai.XAI,
			},
			expected: "grok-3",
		},
		{
			name: "Invalid provider falls back to base model",
			config: ai.ModelConfig{
				BaseModel: ai.GPT4o,
				Provider:  ai.Provider("invalid"),
			},
			expected: "gpt-4o",
		},
		{
			name: "Invalid model for provider falls back to base model",
			config: ai.ModelConfig{
				BaseModel: ai.GPT4o,
				Provider:  ai.Anthropic, // GPT4o not available on Anthropic
			},
			expected: "gpt-4o",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.config.GetModelString()
			if result != tt.expected {
				t.Errorf("GetModelString() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestModelConfig_GetProvider(t *testing.T) {
	tests := []struct {
		name     string
		config   ai.ModelConfig
		expected ai.Provider
	}{
		{
			name: "OpenAI provider",
			config: ai.ModelConfig{
				Provider: ai.OpenAI,
			},
			expected: ai.OpenAI,
		},
		{
			name: "Anthropic provider",
			config: ai.ModelConfig{
				Provider: ai.Anthropic,
			},
			expected: ai.Anthropic,
		},
		{
			name:     "Empty provider",
			config:   ai.ModelConfig{},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.config.GetProvider()
			if result != tt.expected {
				t.Errorf("GetProvider() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestModelConfig_IsOpenAICompatibleModel(t *testing.T) {
	tests := []struct {
		name     string
		config   ai.ModelConfig
		expected bool
	}{
		{
			name: "OpenAI model is compatible",
			config: ai.ModelConfig{
				Provider: ai.OpenAI,
			},
			expected: true,
		},
		{
			name: "XAI model is compatible",
			config: ai.ModelConfig{
				Provider: ai.XAI,
			},
			expected: true,
		},
		{
			name: "Anthropic model is not compatible",
			config: ai.ModelConfig{
				Provider: ai.Anthropic,
			},
			expected: false,
		},
		{
			name: "Bedrock model is not compatible",
			config: ai.ModelConfig{
				Provider: ai.Bedrock,
			},
			expected: false,
		},
		{
			name: "Google model is not compatible",
			config: ai.ModelConfig{
				Provider: ai.Google,
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.config.IsOpenAICompatibleModel()
			if result != tt.expected {
				t.Errorf("IsOpenAICompatibleModel() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestModelConfig_SupportsMCP(t *testing.T) {
	tests := []struct {
		name     string
		config   ai.ModelConfig
		expected bool
	}{
		{
			name: "OpenAI supports MCP",
			config: ai.ModelConfig{
				Provider: ai.OpenAI,
			},
			expected: true,
		},
		{
			name: "XAI supports MCP",
			config: ai.ModelConfig{
				Provider: ai.XAI,
			},
			expected: true,
		},
		{
			name: "Anthropic supports MCP",
			config: ai.ModelConfig{
				Provider: ai.Anthropic,
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.config.SupportsMCP()
			if result != tt.expected {
				t.Errorf("SupportsMCP() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestModelConfig_GetModelProvider(t *testing.T) {
	tests := []struct {
		name     string
		config   ai.ModelConfig
		expected ai.Provider
	}{
		{
			name: "Returns configured provider",
			config: ai.ModelConfig{
				Provider: ai.OpenAI,
			},
			expected: ai.OpenAI,
		},
		{
			name:     "Returns empty for unset provider",
			config:   ai.ModelConfig{},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.config.GetModelProvider()
			if result != tt.expected {
				t.Errorf("GetModelProvider() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// TestAllProvidersHaveModels ensures all providers in the enum have at least one model
func TestAllProvidersHaveModels(t *testing.T) {
	providers := []ai.Provider{
		ai.OpenAI,
		ai.Anthropic,
		ai.Bedrock,
		ai.Google,
		ai.XAI,
		ai.Groq,
	}

	for _, provider := range providers {
		t.Run(string(provider), func(t *testing.T) {
			models, exists := ai.ProviderModelMapping[provider]
			if !exists {
				t.Errorf("Provider %s not found in ProviderModelMapping", provider)
				return
			}
			if len(models) == 0 {
				t.Errorf("Provider %s has no models defined", provider)
			}
		})
	}
}

// TestProviderModelMappingConsistency checks that all mapped models actually exist as constants
func TestProviderModelMappingConsistency(t *testing.T) {
	// Get all BaseModel constants using reflection
	validModels := make(map[ai.BaseModel]bool)

	// Use reflection to get all BaseModel constants
	aiPackage := reflect.TypeOf(ai.GPT4o)
	for i := 0; i < aiPackage.NumMethod(); i++ {
		// This is a simplified check - in a real scenario, you'd enumerate constants differently
	}

	// Manually define expected models for now (this could be automated with build tags or reflection)
	expectedModels := []ai.BaseModel{
		ai.GPT4, ai.GPT4o, ai.GPT4oMini, ai.GPT4Turbo, ai.GPT35Turbo,
		ai.GPT5, ai.GPT5Nano, ai.GPT5Mini, ai.O1, ai.O1Mini, ai.O1Preview,
		ai.Claude35Sonnet, ai.Claude35Haiku, ai.Claude3Sonnet, ai.Claude3Haiku,
		ai.Claude3Opus, ai.Claude37Sonnet, ai.Claude4Sonnet, ai.Claude4Opus,
		ai.ClaudeInstant, ai.ClaudeV2,
		ai.Llama3_8B, ai.Llama3_70B, ai.Llama31_8B, ai.Llama31_70B,
		ai.Llama32_1B, ai.Llama32_3B, ai.Llama32_11B, ai.Llama32_90B, ai.Llama33_70B,
		ai.Mistral7B, ai.Mixtral8x7B, ai.MistralLarge, ai.MistralSmall,
		ai.TitanTextG1Large, ai.TitanTextPremier, ai.TitanTextLite, ai.TitanTextExpress,
		ai.TitanEmbedText, ai.TitanEmbedImage,
		ai.NovaPro, ai.NovaLite, ai.NovaCanvas, ai.NovaReel, ai.NovaMicro,
		ai.Gemini25Flash, ai.Gemini25Pro, ai.Gemini20Flash, ai.Gemini20FlashLite,
		ai.Gemini15Flash, ai.Gemini15Flash8B, ai.Gemini15Pro, ai.GeminiEmbedding, ai.PaLM2,
		ai.Grok4, ai.Grok3, ai.Grok3Mini,
	}

	for _, model := range expectedModels {
		validModels[model] = true
	}

	for provider, models := range ai.ProviderModelMapping {
		for model, modelString := range models {
			t.Run(string(provider)+"_"+string(model), func(t *testing.T) {
				if !validModels[model] {
					t.Errorf("Model %s for provider %s is not a valid BaseModel constant", model, provider)
				}
				if modelString == "" {
					t.Errorf("Model string is empty for %s on provider %s", model, provider)
				}
			})
		}
	}
}

// TestNewKarmaAI tests the constructor with various configurations
func TestNewKarmaAI(t *testing.T) {
	tests := []struct {
		name        string
		model       ai.BaseModel
		provider    ai.Provider
		options     []ai.Option
		expectError bool
	}{
		{
			name:     "Valid OpenAI configuration",
			model:    ai.GPT4o,
			provider: ai.OpenAI,
			options:  []ai.Option{ai.WithTemperature(0.5)},
		},
		{
			name:     "Valid Anthropic configuration",
			model:    ai.Claude35Sonnet,
			provider: ai.Anthropic,
			options:  []ai.Option{ai.WithMaxTokens(1000)},
		},
		{
			name:     "Valid Bedrock configuration",
			model:    ai.Llama3_8B,
			provider: ai.Bedrock,
			options:  []ai.Option{ai.WithTopP(0.9)},
		},
		{
			name:     "Multiple options",
			model:    ai.GPT4oMini,
			provider: ai.OpenAI,
			options: []ai.Option{
				ai.WithTemperature(0.7),
				ai.WithMaxTokens(500),
				ai.WithSystemMessage("Test system message"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			kai := ai.NewKarmaAI(tt.model, tt.provider, tt.options...)

			if kai == nil {
				t.Error("NewKarmaAI returned nil")
				return
			}

			if kai.Model.BaseModel != tt.model {
				t.Errorf("Expected model %v, got %v", tt.model, kai.Model.BaseModel)
			}

			if kai.Model.Provider != tt.provider {
				t.Errorf("Expected provider %v, got %v", tt.provider, kai.Model.Provider)
			}
		})
	}
}

// TestWithOptions tests all option functions
func TestWithOptions(t *testing.T) {
	t.Run("WithSystemMessage", func(t *testing.T) {
		msg := "Test system message"
		kai := ai.NewKarmaAI(ai.GPT4o, ai.OpenAI, ai.WithSystemMessage(msg))
		if kai.SystemMessage != msg {
			t.Errorf("Expected system message %s, got %s", msg, kai.SystemMessage)
		}
	})

	t.Run("WithContext", func(t *testing.T) {
		ctx := "Test context"
		kai := ai.NewKarmaAI(ai.GPT4o, ai.OpenAI, ai.WithContext(ctx))
		if kai.Context != ctx {
			t.Errorf("Expected context %s, got %s", ctx, kai.Context)
		}
	})

	t.Run("WithTemperature", func(t *testing.T) {
		temp := float32(0.8)
		kai := ai.NewKarmaAI(ai.GPT4o, ai.OpenAI, ai.WithTemperature(temp))
		if kai.Temperature != temp {
			t.Errorf("Expected temperature %f, got %f", temp, kai.Temperature)
		}
	})

	t.Run("WithMaxTokens", func(t *testing.T) {
		tokens := 1000
		kai := ai.NewKarmaAI(ai.GPT4o, ai.OpenAI, ai.WithMaxTokens(tokens))
		if kai.MaxTokens != tokens {
			t.Errorf("Expected max tokens %d, got %d", tokens, kai.MaxTokens)
		}
	})

	t.Run("WithTopP", func(t *testing.T) {
		topP := float32(0.9)
		kai := ai.NewKarmaAI(ai.GPT4o, ai.OpenAI, ai.WithTopP(topP))
		if kai.TopP != topP {
			t.Errorf("Expected TopP %f, got %f", topP, kai.TopP)
		}
	})

	t.Run("WithTopK", func(t *testing.T) {
		topK := 50
		kai := ai.NewKarmaAI(ai.GPT4o, ai.OpenAI, ai.WithTopK(topK))
		if kai.TopK != topK {
			t.Errorf("Expected TopK %d, got %d", topK, kai.TopK)
		}
	})
}

// TestMCPConfiguration tests MCP-related configurations
func TestMCPConfiguration(t *testing.T) {
	t.Run("SetMCPUrl", func(t *testing.T) {
		url := "http://localhost:8086/mcp"
		kai := ai.NewKarmaAI(ai.GPT4o, ai.OpenAI, ai.SetMCPUrl(url))
		if kai.MCPUrl != url {
			t.Errorf("Expected MCP URL %s, got %s", url, kai.MCPUrl)
		}
	})

	t.Run("SetMCPAuthToken", func(t *testing.T) {
		token := "test-token"
		kai := ai.NewKarmaAI(ai.GPT4o, ai.OpenAI, ai.SetMCPAuthToken(token))
		if kai.AuthToken != token {
			t.Errorf("Expected auth token %s, got %s", token, kai.AuthToken)
		}
	})

	t.Run("SetMCPTools", func(t *testing.T) {
		tools := []ai.MCPTool{
			{
				FriendlyName: "Calculator",
				ToolName:     "calc",
				Description:  "Basic calculator",
			},
		}
		kai := ai.NewKarmaAI(ai.GPT4o, ai.OpenAI, ai.SetMCPTools(tools))
		if len(kai.MCPTools) != 1 {
			t.Errorf("Expected 1 MCP tool, got %d", len(kai.MCPTools))
		}
		if kai.MCPTools[0].FriendlyName != "Calculator" {
			t.Errorf("Expected tool name Calculator, got %s", kai.MCPTools[0].FriendlyName)
		}
	})

	t.Run("NewMCPServer", func(t *testing.T) {
		server := ai.NewMCPServer("http://localhost:8080", "token", []ai.MCPTool{})
		if server.URL != "http://localhost:8080" {
			t.Errorf("Expected URL http://localhost:8080, got %s", server.URL)
		}
		if server.AuthToken != "token" {
			t.Errorf("Expected token 'token', got %s", server.AuthToken)
		}
	})
}

// TestEdgeCases tests various edge cases and boundary conditions
func TestEdgeCases(t *testing.T) {
	t.Run("Zero temperature", func(t *testing.T) {
		kai := ai.NewKarmaAI(ai.GPT4o, ai.OpenAI, ai.WithTemperature(0))
		if kai.Temperature != 0 {
			t.Errorf("Expected temperature 0, got %f", kai.Temperature)
		}
	})

	t.Run("Maximum temperature", func(t *testing.T) {
		kai := ai.NewKarmaAI(ai.GPT4o, ai.OpenAI, ai.WithTemperature(2.0))
		if kai.Temperature != 2.0 {
			t.Errorf("Expected temperature 2.0, got %f", kai.Temperature)
		}
	})

	t.Run("Empty system message", func(t *testing.T) {
		kai := ai.NewKarmaAI(ai.GPT4o, ai.OpenAI, ai.WithSystemMessage(""))
		if kai.SystemMessage != "" {
			t.Errorf("Expected empty system message, got %s", kai.SystemMessage)
		}
	})

	t.Run("Very large max tokens", func(t *testing.T) {
		kai := ai.NewKarmaAI(ai.GPT4o, ai.OpenAI, ai.WithMaxTokens(1000000))
		if kai.MaxTokens != 1000000 {
			t.Errorf("Expected max tokens 1000000, got %d", kai.MaxTokens)
		}
	})

	t.Run("Custom model string with invalid provider", func(t *testing.T) {
		config := ai.ModelConfig{
			BaseModel:         ai.GPT4o,
			Provider:          ai.Provider("nonexistent"),
			CustomModelString: "custom-model",
		}
		result := config.GetModelString()
		if result != "custom-model" {
			t.Errorf("Expected custom-model, got %s", result)
		}
	})
}

// TestVariantsIntegration tests integration with variants package
func TestVariantsIntegration(t *testing.T) {
	t.Run("GetVariantsForBaseModel", func(t *testing.T) {
		variants := variants.GetVariantsForBaseModel(ai.GPT4o)
		if len(variants) == 0 {
			t.Error("Expected variants for GPT4o, got none")
		}
	})

	t.Run("GetBaseModel", func(t *testing.T) {
		baseModel, ok := variants.GetBaseModel(ai.OpenAI, variants.GPT4o_20241120)
		if !ok || baseModel != ai.GPT4o {
			t.Errorf("Expected base model %s, got %s", ai.GPT4o, baseModel)
		}
	})
}

// BenchmarkModelConfigGetModelString benchmarks the GetModelString method
func BenchmarkModelConfigGetModelString(b *testing.B) {
	config := ai.ModelConfig{
		BaseModel: ai.GPT4o,
		Provider:  ai.OpenAI,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = config.GetModelString()
	}
}

// BenchmarkNewKarmaAI benchmarks the NewKarmaAI constructor
func BenchmarkNewKarmaAI(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ai.NewKarmaAI(ai.GPT4o, ai.OpenAI, ai.WithTemperature(0.5))
	}
}
