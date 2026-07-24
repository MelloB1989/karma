package tests

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/MelloB1989/karma/ai"
	"github.com/MelloB1989/karma/models"
)

func TestCustomProvider_BaseURL(t *testing.T) {
	const envKey = "KARMA_TEST_CUSTOM_BASE_URL"
	os.Unsetenv(envKey)
	t.Cleanup(func() { os.Unsetenv(envKey) })

	p := ai.CustomProvider{
		DefaultBaseURL: "https://default.example.com/v1",
		BaseURLEnv:     envKey,
	}
	AssertEqual(t, "https://default.example.com/v1", p.BaseURL())

	os.Setenv(envKey, "https://override.example.com/v1")
	AssertEqual(t, "https://override.example.com/v1", p.BaseURL())
}

func TestCustomProvider_ResolveAPIKey(t *testing.T) {
	const envKey = "KARMA_TEST_CUSTOM_API_KEY"
	os.Unsetenv(envKey)
	t.Cleanup(func() { os.Unsetenv(envKey) })

	// Neither set.
	AssertEqual(t, "", ai.CustomProvider{}.ResolveAPIKey())

	// Only env var set.
	os.Setenv(envKey, "env-key")
	AssertEqual(t, "env-key", ai.CustomProvider{APIKeyEnv: envKey}.ResolveAPIKey())

	// Direct APIKey takes precedence over APIKeyEnv.
	AssertEqual(t, "direct-key", ai.CustomProvider{APIKey: "direct-key", APIKeyEnv: envKey}.ResolveAPIKey())
}

func TestRegisterCustomProvider_ModelResolution(t *testing.T) {
	provider := ai.Provider("test-registry-provider")
	mapped := ai.BaseModel("test-mapped-model")
	unmapped := ai.BaseModel("test-unmapped-model")

	ai.RegisterCustomProvider(ai.CustomProvider{
		Provider:       provider,
		DefaultBaseURL: "https://vendor.example.com/v1",
		Models: map[ai.BaseModel]string{
			mapped: "vendor/mapped-model-v1",
		},
	})

	kai := ai.NewKarmaAI(mapped, provider)
	AssertEqual(t, "vendor/mapped-model-v1", kai.Model.GetModelString())

	// A BaseModel absent from Models falls back to its raw string.
	kaiUnmapped := ai.NewKarmaAI(unmapped, provider)
	AssertEqual(t, string(unmapped), kaiUnmapped.Model.GetModelString())
}

func TestWithCustomProvider_SetsInstanceFields(t *testing.T) {
	kai := ai.NewKarmaAI(
		ai.BaseModel("any-model"),
		ai.Provider("any-provider"),
		ai.WithCustomProvider("https://instance.example.com/v1", "instance-key"),
	)
	AssertEqual(t, "https://instance.example.com/v1", kai.CustomProviderBaseURL)
	AssertEqual(t, "instance-key", kai.CustomProviderAPIKey)
}

// mockChatCompletionsServer starts an httptest server that speaks just enough
// of the OpenAI Chat Completions response format to exercise the
// OpenAI-compatible dispatch path end to end.
func mockChatCompletionsServer(t *testing.T, wantAuth string, reply string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); wantAuth != "" && got != "Bearer "+wantAuth {
			t.Errorf("Authorization header = %q, want %q", got, "Bearer "+wantAuth)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"id":      "chatcmpl-test",
			"object":  "chat.completion",
			"created": time.Now().Unix(),
			"model":   "test-model",
			"choices": []map[string]any{
				{
					"index": 0,
					"message": map[string]any{
						"role":    "assistant",
						"content": reply,
					},
					"finish_reason": "stop",
				},
			},
			"usage": map[string]any{
				"prompt_tokens":     5,
				"completion_tokens": 3,
				"total_tokens":      8,
			},
		})
	}))
	t.Cleanup(srv.Close)
	return srv
}

func testChatHistory(prompt string) models.AIChatHistory {
	return models.AIChatHistory{
		Messages: []models.AIMessage{
			{
				Message:   prompt,
				Role:      models.User,
				Timestamp: time.Now(),
				UniqueId:  "test-msg-1",
			},
		},
		ChatId:    "test-chat",
		CreatedAt: time.Now(),
		Title:     "Custom Provider Test",
	}
}

func TestChatCompletion_RegisteredCustomProvider(t *testing.T) {
	srv := mockChatCompletionsServer(t, "registry-key", "hello from the registry provider")

	provider := ai.Provider("test-live-registry-provider")
	ai.RegisterCustomProvider(ai.CustomProvider{
		Provider:       provider,
		DefaultBaseURL: srv.URL + "/v1",
		APIKey:         "registry-key",
	})

	kai := ai.NewKarmaAI(ai.BaseModel("any-model"), provider)
	resp, err := kai.ChatCompletion(testChatHistory("hi"))
	AssertNil(t, err)
	AssertNotNil(t, resp)
	AssertEqual(t, "hello from the registry provider", resp.AIResponse)
	AssertEqual(t, 8, resp.Tokens)
}

func TestChatCompletion_InstanceCustomProviderOverridesRegistry(t *testing.T) {
	registrySrv := mockChatCompletionsServer(t, "", "should not be called")
	instanceSrv := mockChatCompletionsServer(t, "instance-key", "hello from the instance override")

	provider := ai.Provider("test-live-override-provider")
	ai.RegisterCustomProvider(ai.CustomProvider{
		Provider:       provider,
		DefaultBaseURL: registrySrv.URL + "/v1",
		APIKey:         "should-not-be-sent",
	})

	kai := ai.NewKarmaAI(
		ai.BaseModel("any-model"),
		provider,
		ai.WithCustomProvider(instanceSrv.URL+"/v1", "instance-key"),
	)
	resp, err := kai.ChatCompletion(testChatHistory("hi"))
	AssertNil(t, err)
	AssertNotNil(t, resp)
	AssertEqual(t, "hello from the instance override", resp.AIResponse)
}
