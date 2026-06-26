package codex

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestParseModelName(t *testing.T) {
	cases := []struct {
		in               string
		id, tier, effort string
	}{
		{"gpt-5.2-codex", "gpt-5.2-codex", "", ""},
		{"gpt-5.2-codex-high", "gpt-5.2-codex", "", "high"},
		{"gpt-5.4-fast", "gpt-5.4", "fast", ""},
		{"gpt-5.4-high-fast", "gpt-5.4", "fast", "high"},
		{"gpt-5", "gpt-5", "", ""},
	}
	for _, c := range cases {
		id, tier, effort := ParseModelName(c.in)
		if id != c.id || tier != c.tier || effort != c.effort {
			t.Errorf("ParseModelName(%q) = (%q,%q,%q), want (%q,%q,%q)",
				c.in, id, tier, effort, c.id, c.tier, c.effort)
		}
	}
}

func TestBuildRequest(t *testing.T) {
	req := BuildRequest(RequestOptions{
		Model:        "gpt-5.2-codex-high",
		Instructions: "be terse",
		Messages: []Message{
			{Role: "system", Content: "ignored here"},
			{Role: "user", Content: "hi"},
			{Role: "assistant", Content: "", ToolCalls: []ToolCall{{ID: "call_1", Name: "get_time", Arguments: "{}"}}},
			{Role: "tool", ToolCallID: "call_1", Content: "12:00"},
			{Role: "user", Content: "look", Images: []string{"data:image/png;base64,AAAA"}},
		},
		Tools: []Tool{{Type: "function", Name: "get_time"}},
	})

	if req.Model != "gpt-5.2-codex" {
		t.Errorf("model = %q, want gpt-5.2-codex", req.Model)
	}
	if !req.Stream || req.Store {
		t.Errorf("stream/store = %v/%v, want true/false", req.Stream, req.Store)
	}
	if req.Reasoning == nil || req.Reasoning.Effort != "high" {
		t.Errorf("reasoning effort not derived from suffix: %+v", req.Reasoning)
	}
	// system message skipped; user, function_call, function_call_output, user(image)
	if len(req.Input) != 4 {
		t.Fatalf("input items = %d, want 4: %+v", len(req.Input), req.Input)
	}
	if req.Input[1].Type != "function_call" || req.Input[1].CallID != "call_1" {
		t.Errorf("expected function_call item, got %+v", req.Input[1])
	}
	if req.Input[2].Type != "function_call_output" || req.Input[2].Output != "12:00" {
		t.Errorf("expected function_call_output item, got %+v", req.Input[2])
	}
	parts, ok := req.Input[3].Content.([]ContentPart)
	if !ok || len(parts) != 2 || parts[1].Type != "input_image" {
		t.Errorf("expected multimodal content parts, got %+v", req.Input[3].Content)
	}

	// Ensure store:false is serialized (not dropped by omitempty).
	raw, _ := json.Marshal(req)
	if !strings.Contains(string(raw), `"store":false`) {
		t.Errorf("serialized body missing store:false: %s", raw)
	}
}

func fakeResponse(sse string) *http.Response {
	return &http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(strings.NewReader(sse)),
	}
}

func TestConsumeTextAndUsage(t *testing.T) {
	sse := strings.Join([]string{
		`event: response.created`,
		`data: {"response":{"id":"resp_1"}}`,
		``,
		`event: response.output_text.delta`,
		`data: {"delta":"Hel"}`,
		``,
		`event: response.output_text.delta`,
		`data: {"delta":"lo"}`,
		``,
		`event: response.completed`,
		`data: {"response":{"id":"resp_1","usage":{"input_tokens":10,"output_tokens":2,"input_tokens_details":{"cached_tokens":4},"output_tokens_details":{"reasoning_tokens":1}}}}`,
		``,
		`data: [DONE]`,
		``,
	}, "\n")

	var streamed strings.Builder
	res, err := Consume(fakeResponse(sse), func(d string) error {
		streamed.WriteString(d)
		return nil
	}, nil)
	if err != nil {
		t.Fatalf("Consume error: %v", err)
	}
	if res.Text != "Hello" || streamed.String() != "Hello" {
		t.Errorf("text=%q streamed=%q, want Hello", res.Text, streamed.String())
	}
	if res.Usage.InputTokens != 10 || res.Usage.OutputTokens != 2 ||
		res.Usage.CachedTokens != 4 || res.Usage.ReasoningTokens != 1 {
		t.Errorf("usage = %+v", res.Usage)
	}
	if res.ResponseID != "resp_1" {
		t.Errorf("responseID = %q", res.ResponseID)
	}
}

func TestConsumeToolCall(t *testing.T) {
	sse := strings.Join([]string{
		`event: response.output_item.added`,
		`data: {"item":{"type":"function_call","id":"item_1","call_id":"call_42","name":"get_weather"}}`,
		``,
		`event: response.function_call_arguments.delta`,
		`data: {"call_id":"item_1","delta":"{\"city\":"}`,
		``,
		`event: response.function_call_arguments.delta`,
		`data: {"call_id":"item_1","delta":"\"NYC\"}"}`,
		``,
		`event: response.function_call_arguments.done`,
		`data: {"call_id":"item_1","name":"get_weather","arguments":"{\"city\":\"NYC\"}"}`,
		``,
		`event: response.completed`,
		`data: {"response":{"id":"resp_2","usage":{"input_tokens":5,"output_tokens":3}}}`,
		``,
	}, "\n")

	res, err := Consume(fakeResponse(sse), nil, nil)
	if err != nil {
		t.Fatalf("Consume error: %v", err)
	}
	if len(res.ToolCalls) != 1 {
		t.Fatalf("tool calls = %d, want 1: %+v", len(res.ToolCalls), res.ToolCalls)
	}
	tc := res.ToolCalls[0]
	if tc.ID != "call_42" || tc.Name != "get_weather" || tc.Arguments != `{"city":"NYC"}` {
		t.Errorf("tool call = %+v", tc)
	}
}

// The real Responses API keys argument events by item_id (not call_id); the
// arguments must still correlate to the call_id from output_item.added.
func TestConsumeToolCallByItemID(t *testing.T) {
	sse := strings.Join([]string{
		`event: response.output_item.added`,
		`data: {"item":{"type":"function_call","id":"fc_1","call_id":"call_99","name":"calendar_add"}}`,
		``,
		`event: response.function_call_arguments.delta`,
		`data: {"item_id":"fc_1","delta":"{\"title\":"}`,
		``,
		`event: response.function_call_arguments.delta`,
		`data: {"item_id":"fc_1","delta":"\"Dentist\"}"}`,
		``,
		`event: response.function_call_arguments.done`,
		`data: {"item_id":"fc_1","arguments":"{\"title\":\"Dentist\"}"}`,
		``,
		`event: response.completed`,
		`data: {"response":{"id":"r","usage":{"input_tokens":1,"output_tokens":1}}}`,
		``,
	}, "\n")
	res, err := Consume(fakeResponse(sse), nil, nil)
	if err != nil {
		t.Fatalf("Consume error: %v", err)
	}
	if len(res.ToolCalls) != 1 {
		t.Fatalf("tool calls = %d, want 1: %+v", len(res.ToolCalls), res.ToolCalls)
	}
	tc := res.ToolCalls[0]
	if tc.ID != "call_99" || tc.Name != "calendar_add" || tc.Arguments != `{"title":"Dentist"}` {
		t.Errorf("tool call (item_id correlation) = %+v", tc)
	}
}

func TestConsumeStreamError(t *testing.T) {
	// error event with a rate-limit code -> APIError(429), retryable.
	sse := "event: error\ndata: {\"error\":{\"code\":\"rate_limit_exceeded\",\"message\":\"slow down\"}}\n\n"
	_, err := Consume(fakeResponse(sse), nil, nil)
	if err == nil {
		t.Fatal("expected error")
	}
	var ae *APIError
	if !errors.As(err, &ae) {
		t.Fatalf("expected *APIError, got %T: %v", err, err)
	}
	if ae.Status != 429 {
		t.Errorf("status = %d, want 429", ae.Status)
	}
	if !IsRetryable(err) {
		t.Errorf("rate-limit error should be retryable: %v", err)
	}
	if !strings.Contains(err.Error(), "slow down") || !strings.Contains(err.Error(), "rate_limit") {
		t.Errorf("error message lost detail: %v", err)
	}
}

func TestConsumeResponseFailedNestedError(t *testing.T) {
	// The real response.failed shape nests the error under response.error.
	sse := `event: response.failed
data: {"type":"response.failed","response":{"id":"resp_9","status":"failed","error":{"type":"server_error","code":"","message":"An error occurred while processing the request."}}}

`
	_, err := Consume(fakeResponse(sse), nil, nil)
	if err == nil {
		t.Fatal("expected error")
	}
	var ae *APIError
	if !errors.As(err, &ae) {
		t.Fatalf("expected *APIError, got %T", err)
	}
	// No code -> generic 502, which is retryable (transient backend failure).
	if ae.Status != 502 {
		t.Errorf("status = %d, want 502", ae.Status)
	}
	if !IsRetryable(err) {
		t.Errorf("codeless response.failed should be retryable")
	}
	if !strings.Contains(err.Error(), "An error occurred while processing the request") {
		t.Errorf("nested message not surfaced: %v", err)
	}
}

func TestAPIErrorRetryAfter(t *testing.T) {
	// Numeric Retry-After header.
	e := &APIError{Status: 429, Headers: http.Header{"Retry-After": {"5"}}}
	if d := e.RetryAfter(); d != 5*time.Second {
		t.Errorf("Retry-After header = %v, want 5s", d)
	}
	// Body resets_in_seconds.
	e = &APIError{Status: 429, Body: `{"error":{"code":"rate_limit_exceeded","resets_in_seconds":12}}`}
	if d := e.RetryAfter(); d != 12*time.Second {
		t.Errorf("resets_in_seconds = %v, want 12s", d)
	}
	if !e.Retryable() {
		t.Errorf("429 should be retryable")
	}
}

func TestCloudflareChallenge(t *testing.T) {
	cf := &APIError{Status: 403, Body: "<html><title>Just a moment...</title>"}
	if !cf.IsCloudflareChallenge() || !cf.Retryable() {
		t.Errorf("CF challenge should be detected and retryable: %+v", cf)
	}
	cfHdr := &APIError{Status: 503, Headers: http.Header{"Cf-Mitigated": {"challenge"}}}
	if !cfHdr.IsCloudflareChallenge() {
		t.Errorf("CF header challenge not detected")
	}
	ban := &APIError{Status: 403, Body: `{"error":{"message":"account banned"}}`}
	if ban.IsCloudflareChallenge() {
		t.Errorf("plain 403 ban must not be classified as CF challenge")
	}
	if ban.Retryable() {
		t.Errorf("403 ban should not be retryable")
	}
}

func TestStatusForCode(t *testing.T) {
	cases := map[string]int{
		"rate_limit_exceeded": 429,
		"usage_limit_reached": 429,
		"invalid_request":     400,
		"unauthorized":        401,
		"insufficient_quota":  402,
		"":                    502,
		"weird_unknown":       502,
	}
	for code, want := range cases {
		if got := statusForCode(code); got != want {
			t.Errorf("statusForCode(%q) = %d, want %d", code, got, want)
		}
	}
}

func TestParseModelList(t *testing.T) {
	// Shape returned by /codex/models (slug + supported_reasoning_levels).
	body := []byte(`{"models":[
		{"slug":"gpt-5.5","display_name":"GPT-5.5","supported_reasoning_levels":[{"effort":"low"},{"effort":"high"}]},
		{"slug":"gpt-5.4-mini","display_name":"GPT-5.4-Mini"}
	]}`)
	got := parseModelList(body)
	if len(got) != 2 {
		t.Fatalf("models = %d, want 2: %+v", len(got), got)
	}
	if got[0].ID != "gpt-5.5" || got[0].DisplayName != "GPT-5.5" {
		t.Errorf("model[0] = %+v", got[0])
	}
	if len(got[0].SupportedReasoningEfforts) != 2 || got[0].SupportedReasoningEfforts[0] != "low" {
		t.Errorf("efforts = %v", got[0].SupportedReasoningEfforts)
	}

	// Nested categories + chat_models wrapper variants.
	nested := []byte(`{"categories":[{"models":[{"id":"a"},{"id":"b"}]}]}`)
	if got := parseModelList(nested); len(got) != 2 || got[0].ID != "a" {
		t.Errorf("nested parse = %+v", got)
	}
	chat := []byte(`{"chat_models":{"models":[{"slug":"x"}]}}`)
	if got := parseModelList(chat); len(got) != 1 || got[0].ID != "x" {
		t.Errorf("chat_models parse = %+v", got)
	}
}

func TestToWSURL(t *testing.T) {
	cases := map[string]string{
		"https://chatgpt.com/backend-api": "wss://chatgpt.com/backend-api",
		"http://localhost:8080/v1":        "ws://localhost:8080/v1",
		"wss://already":                   "wss://already",
	}
	for in, want := range cases {
		if got := toWSURL(in); got != want {
			t.Errorf("toWSURL(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestWSCreateMessageShape(t *testing.T) {
	req := BuildRequest(RequestOptions{
		Model:    "gpt-5.5",
		Messages: []Message{{Role: "user", Content: "hi"}},
	})
	msg := wsCreateMessage{Type: "response.create", ResponsesRequest: req}
	raw, err := json.Marshal(msg)
	if err != nil {
		t.Fatal(err)
	}
	s := string(raw)
	for _, want := range []string{`"type":"response.create"`, `"model":"gpt-5.5"`, `"stream":true`, `"store":false`, `"input":`} {
		if !strings.Contains(s, want) {
			t.Errorf("ws message missing %s: %s", want, s)
		}
	}
	if typ := wsEventType([]byte(`{"type":"response.completed","x":1}`)); typ != "response.completed" {
		t.Errorf("wsEventType = %q", typ)
	}
}

func TestJWTHelpers(t *testing.T) {
	// header.payload.signature with payload containing the auth claim + exp.
	payload := `{"exp":4102444800,"https://api.openai.com/auth":{"chatgpt_account_id":"acct_123"}}`
	token := "x." + b64url(payload) + ".y"
	if got := accountIDFromJWT(token); got != "acct_123" {
		t.Errorf("accountIDFromJWT = %q, want acct_123", got)
	}
	if _, ok := expiryFromJWT(token); !ok {
		t.Errorf("expiryFromJWT failed to parse exp")
	}
}

func b64url(s string) string {
	// base64.RawURLEncoding without importing in test scope noise.
	const tbl = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_"
	var out strings.Builder
	data := []byte(s)
	for i := 0; i < len(data); i += 3 {
		var n uint32
		var k int
		for j := 0; j < 3; j++ {
			n <<= 8
			if i+j < len(data) {
				n |= uint32(data[i+j])
				k++
			}
		}
		for j := 0; j < 4; j++ {
			if j <= k {
				out.WriteByte(tbl[(n>>(18-6*j))&0x3f])
			}
		}
	}
	return out.String()
}
