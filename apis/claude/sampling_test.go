package claude

import "testing"

// Bedrock rejects a request carrying both temperature and top_p. karma's own
// defaults set both, so every call would 400 — the library has to pick one.
func TestNeverSendsBothTemperatureAndTopP(t *testing.T) {
	temp, topP, _ := applySampling(0.7, 0.9, 40, false)
	if !temp.Valid() {
		t.Error("temperature should be sent when configured")
	}
	if topP.Valid() {
		t.Error("top_p must not be sent alongside temperature — Bedrock 400s")
	}
}

// With no temperature, top_p is the caller's only lever and must survive.
func TestTopPUsedWhenNoTemperature(t *testing.T) {
	temp, topP, _ := applySampling(0, 0.9, 0, false)
	if temp.Valid() {
		t.Error("no temperature was configured")
	}
	if !topP.Valid() || topP.Value != 0.9 {
		t.Errorf("top_p = %+v, want 0.9", topP)
	}
}

// Thinking models take neither.
func TestThinkingModelsGetNoSampling(t *testing.T) {
	temp, topP, topK := applySampling(0.7, 0.9, 40, true)
	if temp.Valid() || topP.Valid() || topK.Valid() {
		t.Errorf("thinking model got sampling params: %+v %+v %+v", temp, topP, topK)
	}
}

// top_k is independent of the temperature/top_p choice.
func TestTopKIsIndependent(t *testing.T) {
	_, _, topK := applySampling(0.7, 0.9, 40, false)
	if !topK.Valid() || topK.Value != 40 {
		t.Errorf("top_k = %+v, want 40", topK)
	}
	if _, _, none := applySampling(0.7, 0.9, 0, false); none.Valid() {
		t.Error("top_k of 0 should not be sent")
	}
}

func TestNothingConfiguredSendsNothing(t *testing.T) {
	temp, topP, topK := applySampling(0, 0, 0, false)
	if temp.Valid() || topP.Valid() || topK.Valid() {
		t.Error("no sampling configured, none should be sent")
	}
}
