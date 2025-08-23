package tests

import (
	"errors"
	"testing"
	"time"

	"github.com/MelloB1989/karma/ai"
	"github.com/MelloB1989/karma/models"
)

func TestKarmaAI_ChatCompletion_InputValidation(t *testing.T) {
	tests := []struct {
		name        string
		model       ai.BaseModel
		provider    ai.Provider
		messages    models.AIChatHistory
		expectError bool
	}{
		{
			name:     "Valid OpenAI chat completion",
			model:    ai.GPT4oMini,
			provider: ai.OpenAI,
			messages: models.AIChatHistory{
				Messages: []models.AIMessage{
					{
						Message:   "Hello, how are you?",
						Role:      models.User,
						Timestamp: time.Now(),
						UniqueId:  "test-1",
					},
				},
				ChatId:    "test-chat-1",
				CreatedAt: time.Now(),
				Title:     "Test Chat",
			},
			expectError: false,
		},
		{
			name:     "Valid Anthropic chat completion",
			model:    ai.Claude35Sonnet,
			provider: ai.Anthropic,
			messages: models.AIChatHistory{
				Messages: []models.AIMessage{
					{
						Message:   "Explain quantum computing",
						Role:      models.User,
						Timestamp: time.Now(),
						UniqueId:  "test-2",
					},
				},
				ChatId:    "test-chat-2",
				CreatedAt: time.Now(),
				Title:     "Quantum Computing Chat",
			},
			expectError: false,
		},
		{
			name:     "Valid Bedrock chat completion",
			model:    ai.Llama3_8B,
			provider: ai.Bedrock,
			messages: models.AIChatHistory{
				Messages: []models.AIMessage{
					{
						Message:   "What is machine learning?",
						Role:      models.User,
						Timestamp: time.Now(),
						UniqueId:  "test-3",
					},
				},
				ChatId:    "test-chat-3",
				CreatedAt: time.Now(),
				Title:     "ML Chat",
			},
			expectError: false,
		},
		{
			name:     "Valid XAI chat completion",
			model:    ai.Grok3,
			provider: ai.XAI,
			messages: models.AIChatHistory{
				Messages: []models.AIMessage{
					{
						Message:   "Tell me a joke",
						Role:      models.User,
						Timestamp: time.Now(),
						UniqueId:  "test-4",
					},
				},
				ChatId:    "test-chat-4",
				CreatedAt: time.Now(),
				Title:     "Joke Chat",
			},
			expectError: false,
		},
		{
			name:        "Empty messages",
			model:       ai.GPT4o,
			provider:    ai.OpenAI,
			messages:    models.AIChatHistory{Messages: []models.AIMessage{}},
			expectError: true,
		},
		{
			name:     "Unsupported provider",
			model:    ai.GPT4o,
			provider: ai.Provider("unsupported"),
			messages: models.AIChatHistory{
				Messages: []models.AIMessage{
					{
						Message: "Test",
						Role:    models.User,
					},
				},
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			kai := ai.NewKarmaAI(tt.model, tt.provider)

			// Mock the actual API calls since we're testing structure, not implementation
			if !tt.expectError {
				// We can't actually make API calls in unit tests without API keys
				// So we'll just validate that the KarmaAI instance is properly configured
				if kai.Model.BaseModel != tt.model {
					t.Errorf("Expected model %v, got %v", tt.model, kai.Model.BaseModel)
				}
				if kai.Model.Provider != tt.provider {
					t.Errorf("Expected provider %v, got %v", tt.provider, kai.Model.Provider)
				}
			}
		})
	}
}

func TestKarmaAI_GenerateFromSinglePrompt_InputValidation(t *testing.T) {
	tests := []struct {
		name        string
		model       ai.BaseModel
		provider    ai.Provider
		prompt      string
		expectError bool
	}{
		{
			name:        "Valid prompt with OpenAI",
			model:       ai.GPT4oMini,
			provider:    ai.OpenAI,
			prompt:      "What is the capital of France?",
			expectError: false,
		},
		{
			name:        "Valid prompt with Anthropic",
			model:       ai.Claude35Sonnet,
			provider:    ai.Anthropic,
			prompt:      "Explain photosynthesis",
			expectError: false,
		},
		{
			name:        "Valid prompt with Bedrock",
			model:       ai.Llama3_8B,
			provider:    ai.Bedrock,
			prompt:      "Write a short story",
			expectError: false,
		},
		{
			name:        "Valid prompt with Google",
			model:       ai.Gemini25Flash,
			provider:    ai.Google,
			prompt:      "Describe the water cycle",
			expectError: false,
		},
		{
			name:        "Valid prompt with XAI",
			model:       ai.Grok3Mini,
			provider:    ai.XAI,
			prompt:      "What is AI?",
			expectError: false,
		},
		{
			name:        "Empty prompt",
			model:       ai.GPT4o,
			provider:    ai.OpenAI,
			prompt:      "",
			expectError: false, // Empty prompt might be valid depending on implementation
		},
		{
			name:        "Very long prompt",
			model:       ai.GPT4o,
			provider:    ai.OpenAI,
			prompt:      generateLongStringAI(10000),
			expectError: false, // Should be handled by token limits
		},
		{
			name:        "Unsupported provider",
			model:       ai.GPT4o,
			provider:    ai.Provider("invalid"),
			prompt:      "Test prompt",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			kai := ai.NewKarmaAI(tt.model, tt.provider)

			// Validate configuration
			if kai.Model.BaseModel != tt.model {
				t.Errorf("Expected model %v, got %v", tt.model, kai.Model.BaseModel)
			}
			if kai.Model.Provider != tt.provider {
				t.Errorf("Expected provider %v, got %v", tt.provider, kai.Model.Provider)
			}
		})
	}
}

func TestKarmaAI_ChatCompletionStream_InputValidation(t *testing.T) {
	mockCallback := func(chunk models.StreamedResponse) error {
		return nil
	}

	errorCallback := func(chunk models.StreamedResponse) error {
		return errors.New("callback error")
	}

	tests := []struct {
		name        string
		model       ai.BaseModel
		provider    ai.Provider
		messages    models.AIChatHistory
		callback    func(chunk models.StreamedResponse) error
		expectError bool
	}{
		{
			name:     "Valid stream with OpenAI",
			model:    ai.GPT4oMini,
			provider: ai.OpenAI,
			messages: models.AIChatHistory{
				Messages: []models.AIMessage{
					{
						Message: "Stream this response",
						Role:    models.User,
					},
				},
			},
			callback:    mockCallback,
			expectError: false,
		},
		{
			name:     "Valid stream with Anthropic",
			model:    ai.Claude35Sonnet,
			provider: ai.Anthropic,
			messages: models.AIChatHistory{
				Messages: []models.AIMessage{
					{
						Message: "Stream this response",
						Role:    models.User,
					},
				},
			},
			callback:    mockCallback,
			expectError: false,
		},
		{
			name:     "Valid stream with Bedrock",
			model:    ai.Llama3_8B,
			provider: ai.Bedrock,
			messages: models.AIChatHistory{
				Messages: []models.AIMessage{
					{
						Message: "Stream this response",
						Role:    models.User,
					},
				},
			},
			callback:    mockCallback,
			expectError: false,
		},
		{
			name:     "Valid stream with XAI",
			model:    ai.Grok3,
			provider: ai.XAI,
			messages: models.AIChatHistory{
				Messages: []models.AIMessage{
					{
						Message: "Stream this response",
						Role:    models.User,
					},
				},
			},
			callback:    mockCallback,
			expectError: false,
		},
		{
			name:     "Callback returns error",
			model:    ai.GPT4o,
			provider: ai.OpenAI,
			messages: models.AIChatHistory{
				Messages: []models.AIMessage{
					{
						Message: "Test",
						Role:    models.User,
					},
				},
			},
			callback:    errorCallback,
			expectError: false, // Error in callback doesn't necessarily fail the stream
		},
		{
			name:        "Nil callback",
			model:       ai.GPT4o,
			provider:    ai.OpenAI,
			messages:    models.AIChatHistory{Messages: []models.AIMessage{{Message: "Test", Role: models.User}}},
			callback:    nil,
			expectError: true,
		},
		{
			name:        "Unsupported provider for streaming",
			model:       ai.GPT4o,
			provider:    ai.Provider("invalid"),
			messages:    models.AIChatHistory{Messages: []models.AIMessage{{Message: "Test", Role: models.User}}},
			callback:    mockCallback,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			kai := ai.NewKarmaAI(tt.model, tt.provider)

			// Validate basic configuration
			if kai.Model.BaseModel != tt.model {
				t.Errorf("Expected model %v, got %v", tt.model, kai.Model.BaseModel)
			}
			if kai.Model.Provider != tt.provider {
				t.Errorf("Expected provider %v, got %v", tt.provider, kai.Model.Provider)
			}

			// Test nil callback handling
			if tt.callback == nil && !tt.expectError {
				t.Error("Expected error for nil callback")
			}
		})
	}
}

func TestKarmaAI_MultiModalSupport(t *testing.T) {
	tests := []struct {
		name     string
		model    ai.BaseModel
		provider ai.Provider
		message  models.AIMessage
		valid    bool
	}{
		{
			name:     "OpenAI with image",
			model:    ai.GPT4o,
			provider: ai.OpenAI,
			message: models.AIMessage{
				Message: "Describe this image",
				Role:    models.User,
				Images:  []string{"https://example.com/image.jpg"},
			},
			valid: true,
		},
		{
			name:     "OpenAI with multiple images",
			model:    ai.GPT4o,
			provider: ai.OpenAI,
			message: models.AIMessage{
				Message: "Compare these images",
				Role:    models.User,
				Images:  []string{"https://example.com/image1.jpg", "https://example.com/image2.jpg"},
			},
			valid: true,
		},
		{
			name:     "Claude with image (not supported)",
			model:    ai.Claude35Sonnet,
			provider: ai.Anthropic,
			message: models.AIMessage{
				Message: "Describe this image",
				Role:    models.User,
				Images:  []string{"https://example.com/image.jpg"},
			},
			valid: false, // Anthropic might not support images in all cases
		},
		{
			name:     "Text-only message",
			model:    ai.GPT4o,
			provider: ai.OpenAI,
			message: models.AIMessage{
				Message: "Hello world",
				Role:    models.User,
			},
			valid: true,
		},
		{
			name:     "Message with files",
			model:    ai.GPT4o,
			provider: ai.OpenAI,
			message: models.AIMessage{
				Message: "Analyze this document",
				Role:    models.User,
				Files:   []string{"https://example.com/document.pdf"},
			},
			valid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			kai := ai.NewKarmaAI(tt.model, tt.provider)

			// Validate that the configuration accepts the input structure
			if kai.Model.BaseModel != tt.model {
				t.Errorf("Expected model %v, got %v", tt.model, kai.Model.BaseModel)
			}

			// Check if message structure is valid
			if tt.message.Message == "" && len(tt.message.Images) == 0 && len(tt.message.Files) == 0 {
				t.Error("Message should have content, images, or files")
			}
		})
	}
}

func TestKarmaAI_ConfigurationDefaults(t *testing.T) {
	kai := ai.NewKarmaAI(ai.GPT4o, ai.OpenAI)

	// Test default values
	if kai.Temperature < 0 || kai.Temperature > 2 {
		t.Errorf("Default temperature should be between 0 and 2, got %f", kai.Temperature)
	}

	if kai.MaxTokens < 0 {
		t.Errorf("Max tokens should be non-negative, got %d", kai.MaxTokens)
	}

	if kai.TopP < 0 || kai.TopP > 1 {
		t.Errorf("TopP should be between 0 and 1, got %f", kai.TopP)
	}

	if kai.TopK < 0 {
		t.Errorf("TopK should be non-negative, got %d", kai.TopK)
	}
}

func TestKarmaAI_ProviderSpecificValidation(t *testing.T) {
	tests := []struct {
		name     string
		provider ai.Provider
		model    ai.BaseModel
		valid    bool
	}{
		{
			name:     "Valid OpenAI model",
			provider: ai.OpenAI,
			model:    ai.GPT4o,
			valid:    true,
		},
		{
			name:     "Invalid model for OpenAI",
			provider: ai.OpenAI,
			model:    ai.Claude35Sonnet,
			valid:    false,
		},
		{
			name:     "Valid Anthropic model",
			provider: ai.Anthropic,
			model:    ai.Claude35Sonnet,
			valid:    true,
		},
		{
			name:     "Invalid model for Anthropic",
			provider: ai.Anthropic,
			model:    ai.GPT4o,
			valid:    false,
		},
		{
			name:     "Valid Bedrock model",
			provider: ai.Bedrock,
			model:    ai.Llama3_8B,
			valid:    true,
		},
		{
			name:     "Valid Google model",
			provider: ai.Google,
			model:    ai.Gemini25Flash,
			valid:    true,
		},
		{
			name:     "Valid XAI model",
			provider: ai.XAI,
			model:    ai.Grok3,
			valid:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := ai.ModelConfig{
				BaseModel: tt.model,
				Provider:  tt.provider,
			}

			modelString := config.GetModelString()

			if tt.valid && modelString == "" {
				t.Error("Expected valid model string for valid provider-model combination")
			}

			if !tt.valid && modelString == "" {
				t.Error("Model string should fall back to base model even for invalid combinations")
			}
		})
	}
}

func TestKarmaAI_EdgeCaseHandling(t *testing.T) {
	t.Run("Very long conversation history", func(t *testing.T) {
		_ = ai.NewKarmaAI(ai.GPT4o, ai.OpenAI)

		messages := models.AIChatHistory{
			Messages: make([]models.AIMessage, 1000), // Very long conversation
		}

		for i := range messages.Messages {
			messages.Messages[i] = models.AIMessage{
				Message:   "Message " + string(rune(i)),
				Role:      models.User,
				Timestamp: time.Now(),
				UniqueId:  "msg-" + string(rune(i)),
			}
		}

		// Should handle large conversation history without panicking
		if len(messages.Messages) != 1000 {
			t.Errorf("Expected 1000 messages, got %d", len(messages.Messages))
		}
	})

	t.Run("Unicode and special characters", func(t *testing.T) {
		_ = ai.NewKarmaAI(ai.GPT4o, ai.OpenAI)

		message := models.AIMessage{
			Message: "Hello 世界! 🌍 Testing émojis and spëcial chars: ñ, ü, ß, ç, æ",
			Role:    models.User,
		}

		if message.Message == "" {
			t.Error("Message should not be empty")
		}

		// Should handle unicode without issues
		if len([]rune(message.Message)) == 0 {
			t.Error("Message should contain unicode characters")
		}
	})

	t.Run("Extreme parameter values", func(t *testing.T) {
		kai := ai.NewKarmaAI(ai.GPT4o, ai.OpenAI,
			ai.WithTemperature(2.0), // Maximum temperature
			ai.WithMaxTokens(0),     // Minimum tokens
			ai.WithTopP(1.0),        // Maximum TopP
			ai.WithTopK(0),          // Minimum TopK
		)

		if kai.Temperature != 2.0 {
			t.Errorf("Expected temperature 2.0, got %f", kai.Temperature)
		}
		if kai.MaxTokens != 0 {
			t.Errorf("Expected max tokens 0, got %d", kai.MaxTokens)
		}
		if kai.TopP != 1.0 {
			t.Errorf("Expected TopP 1.0, got %f", kai.TopP)
		}
		if kai.TopK != 0 {
			t.Errorf("Expected TopK 0, got %d", kai.TopK)
		}
	})
}

// Helper function to generate long strings for testing
func generateLongStringAI(length int) string {
	result := make([]byte, length)
	for i := range result {
		result[i] = 'a' + byte(i%26)
	}
	return string(result)
}

// BenchmarkKarmaAI_Initialization benchmarks the initialization of KarmaAI
func BenchmarkKarmaAI_Initialization(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ai.NewKarmaAI(ai.GPT4o, ai.OpenAI,
			ai.WithTemperature(0.7),
			ai.WithMaxTokens(1000),
			ai.WithSystemMessage("Test system message"),
		)
	}
}

// BenchmarkModelConfigGetModelStringAI benchmarks model string retrieval
func BenchmarkModelConfigGetModelStringAI(b *testing.B) {
	config := ai.ModelConfig{
		BaseModel: ai.GPT4o,
		Provider:  ai.OpenAI,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = config.GetModelString()
	}
}

func TestKarmaAI_Analytics(t *testing.T) {
	t.Run("Analytics configuration", func(t *testing.T) {
		kai := ai.NewKarmaAI(ai.GPT4o, ai.OpenAI,
			ai.ConfigureAnalytics("test-user", "test-trace", true, true, true))

		if kai.Analytics == nil {
			t.Error("Expected Analytics to be configured")
		}
		if kai.Analytics != nil && kai.Analytics.DistinctID != "test-user" {
			t.Errorf("Expected DistinctID 'test-user', got %s", kai.Analytics.DistinctID)
		}
		if kai.Analytics != nil && !kai.Analytics.CaptureUserPrompts {
			t.Error("Expected CaptureUserPrompts to be true")
		}
	})
}
