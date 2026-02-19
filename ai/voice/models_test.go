package voice

import "testing"

func TestGetAvailableModels(t *testing.T) {
	sttOpenAI := GetAvailableSTTModels(ProviderOpenAI)
	if len(sttOpenAI) == 0 {
		t.Fatal("expected openai stt models")
	}

	ttsTogether := GetAvailableTTSModels(ProviderTogether)
	if len(ttsTogether) == 0 {
		t.Fatal("expected together tts models")
	}

	sttUnknown := GetAvailableSTTModels(Provider("unknown"))
	if len(sttUnknown) != 0 {
		t.Fatalf("expected zero models for unknown provider, got %d", len(sttUnknown))
	}
}

func TestResolveProviderModel(t *testing.T) {
	model, err := resolveProviderModel(ProviderOpenAI, ModelKindSTT, OpenAIWhisper1, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if model != "whisper-1" {
		t.Fatalf("unexpected model string: %s", model)
	}

	model, err = resolveProviderModel(ProviderTogether, ModelKindTTS, "", TogetherHexgradKokoro82M)
	if err != nil {
		t.Fatalf("unexpected error from fallback model: %v", err)
	}
	if model != "hexgrad/Kokoro-82M" {
		t.Fatalf("unexpected fallback model string: %s", model)
	}

	_, err = resolveProviderModel(ProviderOpenAI, ModelKindTTS, TogetherHexgradKokoro82M, OpenAIGPT4oMiniTTS)
	if err == nil {
		t.Fatal("expected unsupported model error")
	}
}
