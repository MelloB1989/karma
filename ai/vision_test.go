package ai

import (
	"testing"
)

func TestVisionSupport(t *testing.T) {
	tests := []struct {
		name      string
		model     BaseModel
		provider  Provider
		wantVision bool
	}{
		// OpenAI models
		{"GPT-4o supports vision", GPT4o, OpenAI, true},
		{"GPT-4o-mini supports vision", GPT4oMini, OpenAI, true},
		{"GPT-4-turbo supports vision", GPT4Turbo, OpenAI, true},
		{"GPT-4 supports vision", GPT4, OpenAI, true},
		{"GPT-3.5-turbo does not support vision", GPT35Turbo, OpenAI, false},
		
		// Anthropic models
		{"Claude-3.5-sonnet supports vision", Claude35Sonnet, Anthropic, true},
		{"Claude-3-sonnet supports vision", Claude3Sonnet, Anthropic, true},
		{"Claude-3-haiku supports vision", Claude3Haiku, Anthropic, true},
		{"Claude-3-opus supports vision", Claude3Opus, Anthropic, true},
		
		// Google models
		{"Gemini-2.5-flash supports vision", Gemini25Flash, Google, true},
		{"Gemini-2.0-flash supports vision", Gemini20Flash, Google, true},
		{"Gemini-1.5-pro supports vision", Gemini15Pro, Google, true},
		
		// Groq models (currently no vision support)
		{"Llama-3.1-8b on Groq does not support vision", Llama31_8B, Groq, false},
		{"Llama-3.3-70b on Groq does not support vision", Llama33_70B, Groq, false},
		{"Llama-4-scout-17b on Groq does not support vision", Llama4_Scout_17B, Groq, false},
		
		// XAI models
		{"Grok-4 supports vision", Grok4, XAI, true},
		{"Grok-3 supports vision", Grok3, XAI, true},
		{"Grok-3-mini does not support vision", Grok3Mini, XAI, false},
		
		// Bedrock models
		{"Claude-3.5-sonnet on Bedrock supports vision", Claude35Sonnet, Bedrock, true},
		{"Nova-pro on Bedrock supports vision", NovaPro, Bedrock, true},
		{"Llama-3-8b on Bedrock does not support vision", Llama3_8B, Bedrock, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := ModelConfig{
				BaseModel: tt.model,
				Provider:  tt.provider,
			}
			
			got := model.SupportsVision()
			if got != tt.wantVision {
				t.Errorf("ModelConfig.SupportsVision() = %v, want %v", got, tt.wantVision)
			}
		})
	}
}

func TestKarmaAIVisionSupport(t *testing.T) {
	// Test that KarmaAI instances correctly report vision support
	tests := []struct {
		name       string
		model      BaseModel
		provider   Provider
		wantVision bool
	}{
		{"OpenAI GPT-4o", GPT4o, OpenAI, true},
		{"Groq Llama", Llama4_Scout_17B, Groq, false},
		{"Anthropic Claude", Claude35Sonnet, Anthropic, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			kai := NewKarmaAI(tt.model, tt.provider)
			got := kai.Model.SupportsVision()
			if got != tt.wantVision {
				t.Errorf("KarmaAI vision support = %v, want %v", got, tt.wantVision)
			}
		})
	}
}