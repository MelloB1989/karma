package voice

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestBuildVapiAssistantDTO_ConfiguresModelVoiceAndTranscriber(t *testing.T) {
	temperature := 0.7
	maxTokens := 180.0
	maxDuration := 300.0

	dto, bodyProperties, err := buildVapiAssistantDTO(VapiConfig{
		AssistantName:       "support",
		FirstMessage:        "Hi, how can I help?",
		SystemPrompt:        "Keep answers short.",
		ModelProvider:       "openai",
		Model:               "gpt-4o",
		Temperature:         &temperature,
		MaxTokens:           &maxTokens,
		VoiceProvider:       "11labs",
		VoiceID:             "voice-123",
		TranscriberProvider: "deepgram",
		TranscriberModel:    "nova-2",
		TranscriberLanguage: "en-US",
		MaxDurationSeconds:  &maxDuration,
		ServerURL:           "https://example.com/vapi",
		Metadata:            map[string]any{"team": "support"},
	}, VapiAssistantRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(bodyProperties) != 0 {
		t.Fatalf("expected no extra body properties, got %#v", bodyProperties)
	}

	var payload map[string]any
	marshalIntoMap(t, dto, &payload)

	if payload["name"] != "support" {
		t.Fatalf("unexpected name: %#v", payload["name"])
	}
	if payload["firstMessage"] != "Hi, how can I help?" {
		t.Fatalf("unexpected first message: %#v", payload["firstMessage"])
	}
	if payload["maxDurationSeconds"] != maxDuration {
		t.Fatalf("unexpected max duration: %#v", payload["maxDurationSeconds"])
	}

	model := payload["model"].(map[string]any)
	if model["provider"] != "openai" || model["model"] != "gpt-4o" {
		t.Fatalf("unexpected model payload: %#v", model)
	}
	if model["temperature"] != temperature || model["maxTokens"] != maxTokens {
		t.Fatalf("unexpected model params: %#v", model)
	}
	messages := model["messages"].([]any)
	system := messages[0].(map[string]any)
	if system["role"] != "system" || system["content"] != "Keep answers short." {
		t.Fatalf("unexpected system message: %#v", system)
	}

	voice := payload["voice"].(map[string]any)
	if voice["provider"] != "11labs" || voice["voiceId"] != "voice-123" {
		t.Fatalf("unexpected voice payload: %#v", voice)
	}

	transcriber := payload["transcriber"].(map[string]any)
	if transcriber["provider"] != "deepgram" || transcriber["model"] != "nova-2" || transcriber["language"] != "en-US" {
		t.Fatalf("unexpected transcriber payload: %#v", transcriber)
	}

	server := payload["server"].(map[string]any)
	if server["url"] != "https://example.com/vapi" {
		t.Fatalf("unexpected server payload: %#v", server)
	}
}

func TestBuildVapiCallDTO_UsesConfiguredAssistantAndMetadata(t *testing.T) {
	checkNumber := false
	dto, bodyProperties, err := buildVapiCallDTO(VapiConfig{
		AssistantID:   "assistant-123",
		PhoneNumberID: "phone-123",
		CallMetadata:  map[string]any{"source": "test"},
	}, VapiCallRequest{
		Name:                           "demo call",
		CustomerNumber:                 "+1234567890",
		CustomerName:                   "Ada",
		CustomerNumberE164CheckEnabled: &checkNumber,
		Metadata:                       map[string]any{"lead": "warm"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var payload map[string]any
	marshalIntoMap(t, dto, &payload)

	if payload["assistantId"] != "assistant-123" {
		t.Fatalf("expected assistant id, got %#v", payload["assistantId"])
	}
	if payload["phoneNumberId"] != "phone-123" {
		t.Fatalf("expected phone number id, got %#v", payload["phoneNumberId"])
	}
	customer := payload["customer"].(map[string]any)
	if customer["number"] != "+1234567890" || customer["name"] != "Ada" {
		t.Fatalf("unexpected customer payload: %#v", customer)
	}
	if customer["numberE164CheckEnabled"] != false {
		t.Fatalf("unexpected number check payload: %#v", customer)
	}

	metadata := bodyProperties["metadata"].(map[string]any)
	if metadata["source"] != "test" || metadata["lead"] != "warm" {
		t.Fatalf("unexpected metadata body properties: %#v", metadata)
	}
}

func TestBuildVapiCallDTO_CanUseTransientAssistant(t *testing.T) {
	dto, _, err := buildVapiCallDTO(VapiConfig{
		PhoneNumberID: "phone-123",
		SystemPrompt:  "You are concise.",
		ModelProvider: "together-ai",
		Model:         "meta-llama/Llama-3.3-70B-Instruct-Turbo",
	}, VapiCallRequest{
		CustomerNumber: "+1234567890",
		UseTransient:   true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var payload map[string]any
	marshalIntoMap(t, dto, &payload)

	if _, exists := payload["assistantId"]; exists {
		t.Fatalf("expected transient assistant, got assistantId: %#v", payload["assistantId"])
	}
	assistant := payload["assistant"].(map[string]any)
	model := assistant["model"].(map[string]any)
	if model["provider"] != "together-ai" || model["model"] != "meta-llama/Llama-3.3-70B-Instruct-Turbo" {
		t.Fatalf("unexpected transient assistant model: %#v", model)
	}
}

func TestNewAgentRejectsVapiSpeechProvider(t *testing.T) {
	_, err := NewAgent(&fakeTextAI{}, ProviderVapi)
	if err == nil {
		t.Fatal("expected ProviderVapi to be rejected by STT/TTS agent")
	}
	if !strings.Contains(err.Error(), "NewVapiAgent") {
		t.Fatalf("expected NewVapiAgent guidance, got %v", err)
	}
}

func marshalIntoMap(t *testing.T, value any, out *map[string]any) {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}
	if err := json.Unmarshal(data, out); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}
}
