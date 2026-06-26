package codex

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// Message is a provider-agnostic chat message used to build a Responses request.
type Message struct {
	Role       string // "user" | "assistant" | "system" | "developer" | "tool"
	Content    string
	Images     []string   // image URLs or data: URIs (user messages)
	ToolCalls  []ToolCall // assistant tool calls to replay
	ToolCallID string     // tool result correlation id
}

// RequestOptions describes a Codex Responses request in protocol-neutral terms.
type RequestOptions struct {
	Model           string
	Instructions    string // collected system/developer text
	Messages        []Message
	Tools           []Tool
	ToolChoice      any
	ReasoningEffort string // explicit override; suffix on Model is used otherwise
	ServiceTier     string // explicit override; suffix on Model is used otherwise
}

// Service-tier and reasoning-effort suffixes recognized on model names, e.g.
// "gpt-5.2-codex-high" or "gpt-5.4-fast" (port of stripKnownModelSuffixes).
var (
	serviceTierSuffixes = []string{"fast", "flex"}
	effortSuffixes      = []string{"none", "minimal", "low", "medium", "high", "xhigh"}
)

// ParseModelName splits a model string into its base id and any trailing
// service-tier / reasoning-effort suffixes.
func ParseModelName(input string) (modelID, serviceTier, effort string) {
	remaining := strings.TrimSpace(input)
	for _, tier := range serviceTierSuffixes {
		if strings.HasSuffix(remaining, "-"+tier) {
			serviceTier = tier
			remaining = remaining[:len(remaining)-len(tier)-1]
			break
		}
	}
	for _, eff := range effortSuffixes {
		if strings.HasSuffix(remaining, "-"+eff) {
			effort = eff
			remaining = remaining[:len(remaining)-len(eff)-1]
			break
		}
	}
	return remaining, serviceTier, effort
}

// BuildRequest translates protocol-neutral options into a CodexResponsesRequest.
func BuildRequest(opts RequestOptions) *ResponsesRequest {
	instructions := strings.TrimSpace(opts.Instructions)
	if instructions == "" {
		instructions = "You are a helpful assistant."
	}

	input := make([]InputItem, 0, len(opts.Messages))
	for _, msg := range opts.Messages {
		switch msg.Role {
		case "system", "developer":
			// Folded into instructions by the caller; skip here.
			continue
		case "assistant":
			if msg.Content != "" || len(msg.ToolCalls) == 0 {
				input = append(input, InputItem{Role: "assistant", Content: msg.Content})
			}
			for _, tc := range msg.ToolCalls {
				input = append(input, InputItem{
					Type:      "function_call",
					CallID:    tc.ID,
					Name:      tc.Name,
					Arguments: tc.Arguments,
				})
			}
		case "tool", "function":
			callID := msg.ToolCallID
			if callID == "" {
				callID = "unknown"
			}
			input = append(input, InputItem{
				Type:   "function_call_output",
				CallID: callID,
				Output: msg.Content,
			})
		default: // user
			input = append(input, InputItem{Role: "user", Content: userContent(msg)})
		}
	}
	if len(input) == 0 {
		input = append(input, InputItem{Role: "user", Content: ""})
	}

	modelID, suffixTier, suffixEffort := ParseModelName(opts.Model)

	req := &ResponsesRequest{
		Model:        modelID,
		Instructions: instructions,
		Input:        input,
		Stream:       true,
		Store:        false,
		Tools:        opts.Tools,
		ToolChoice:   opts.ToolChoice,
	}

	if effort := firstNonEmpty(opts.ReasoningEffort, suffixEffort); effort != "" {
		req.Reasoning = &Reasoning{Effort: effort, Summary: "auto"}
	}
	if tier := firstNonEmpty(opts.ServiceTier, suffixTier); tier != "" {
		req.ServiceTier = tier
	}
	return req
}

// userContent returns either a plain string (text only) or a []ContentPart when
// the message carries images.
func userContent(msg Message) any {
	if len(msg.Images) == 0 {
		return msg.Content
	}
	parts := make([]ContentPart, 0, len(msg.Images)+1)
	if msg.Content != "" {
		parts = append(parts, ContentPart{Type: "input_text", Text: msg.Content})
	}
	for _, img := range msg.Images {
		if img != "" {
			parts = append(parts, ContentPart{Type: "input_image", ImageURL: img})
		}
	}
	if len(parts) == 0 {
		return ""
	}
	return parts
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

// ---- SSE event consumption ----

type sseRespEnvelope struct {
	Response struct {
		ID    string `json:"id"`
		Usage *struct {
			InputTokens        int `json:"input_tokens"`
			OutputTokens       int `json:"output_tokens"`
			InputTokensDetails struct {
				CachedTokens int `json:"cached_tokens"`
			} `json:"input_tokens_details"`
			OutputTokensDetails struct {
				ReasoningTokens int `json:"reasoning_tokens"`
			} `json:"output_tokens_details"`
		} `json:"usage"`
	} `json:"response"`
}

type sseDelta struct {
	Delta string `json:"delta"`
}

type sseItem struct {
	Item struct {
		Type   string `json:"type"`
		ID     string `json:"id"`
		CallID string `json:"call_id"`
		Name   string `json:"name"`
	} `json:"item"`
}

type sseFnArgs struct {
	Delta     string `json:"delta"`
	Arguments string `json:"arguments"`
	// The Responses API keys argument deltas by item_id (the output item that
	// response.output_item.added registered), though some events also carry
	// call_id. We accept either.
	ItemID string `json:"item_id"`
	CallID string `json:"call_id"`
	Name   string `json:"name"`
}

// key returns the identifier used to correlate an arguments event back to the
// function-call item registered by response.output_item.added.
func (fa sseFnArgs) key() string {
	return firstNonEmpty(fa.ItemID, fa.CallID)
}

// codexErrorInfo is the normalized error extracted from an `error` /
// `response.failed` SSE event.
type codexErrorInfo struct {
	Type    string
	Code    string
	Message string
}

// extractCodexError mirrors codex-proxy's extractCodexError: it locates the
// error record in `error`, then `response.error`, then the event root, and
// pulls type/code/message with the same fallbacks so the real reason surfaces
// even when the backend nests it (response.failed) or omits a code.
func extractCodexError(data json.RawMessage) codexErrorInfo {
	root := objOf(data)
	if root == nil {
		// Non-object payload (e.g. a bare string) — surface it verbatim.
		return codexErrorInfo{Type: "error", Code: "unknown", Message: rawString(data)}
	}

	response := objField(root, "response")
	errRec := objField(root, "error")
	if errRec == nil {
		errRec = objField(response, "error")
	}
	if errRec == nil {
		errRec = root
	}

	message := firstStringField(
		errRec["message"], errRec["detail"], errRec["error_description"],
		root["message"], root["detail"],
		response["message"], response["detail"],
	)
	if message == "" {
		message = rawString(data)
	}
	typ := firstStringField(errRec["type"], root["type"])
	if typ == "" {
		typ = "error"
	}
	code := firstStringField(errRec["code"], response["code"])
	if code == "" {
		if typ != "error" && typ != "response.failed" {
			code = typ
		} else {
			code = "unknown"
		}
	}
	return codexErrorInfo{Type: typ, Code: code, Message: message}
}

// streamErrorToAPIError maps a mid-stream error event onto an HTTP-equivalent
// *APIError (port of codex-proxy's codexApiErrorFromEvent), so a single retry
// policy covers both HTTP- and stream-layer failures.
func streamErrorToAPIError(info codexErrorInfo) *APIError {
	body, _ := json.Marshal(map[string]any{
		"error": map[string]string{
			"type":    info.Type,
			"code":    info.Code,
			"message": info.Message,
		},
	})
	return &APIError{Status: statusForCode(info.Code), Body: string(body)}
}

// statusForCode maps a Codex error code to an HTTP-equivalent status, matching
// codex-proxy's statusForCode. Unknown/generic failures become 502.
func statusForCode(code string) int {
	lower := strings.ToLower(code)
	switch {
	case strings.Contains(lower, "invalid_request"), strings.Contains(lower, "not_found"):
		return 400
	case strings.Contains(lower, "rate_limit"), strings.Contains(lower, "usage_limit"):
		return 429
	case strings.Contains(lower, "unauthorized"), strings.Contains(lower, "invalid_api_key"):
		return 401
	case strings.Contains(lower, "forbidden"), strings.Contains(lower, "banned"):
		return 403
	case strings.Contains(lower, "payment"), strings.Contains(lower, "quota"):
		return 402
	default:
		return 502
	}
}

// objOf unmarshals data into a JSON object map, or nil if it is not an object.
func objOf(data json.RawMessage) map[string]json.RawMessage {
	var out map[string]json.RawMessage
	if json.Unmarshal(data, &out) != nil {
		return nil
	}
	return out
}

// objField returns m[key] decoded as a JSON object, or nil. Safe on a nil map.
func objField(m map[string]json.RawMessage, key string) map[string]json.RawMessage {
	if m == nil {
		return nil
	}
	return objOf(m[key])
}

// firstStringField returns the first value that decodes to a non-empty string.
func firstStringField(vals ...json.RawMessage) string {
	for _, v := range vals {
		if len(v) == 0 {
			continue
		}
		var s string
		if json.Unmarshal(v, &s) == nil && s != "" {
			return s
		}
	}
	return ""
}

// rawString returns the JSON string value, or the raw JSON text if not a string.
func rawString(data json.RawMessage) string {
	var s string
	if json.Unmarshal(data, &s) == nil {
		return s
	}
	return string(data)
}

type toolAccum struct {
	id   string
	name string
	args strings.Builder
}

// Consume reads a Codex Responses SSE stream (HTTP transport) to completion.
// When onText is non-nil it is invoked for each text delta (streaming);
// onReasoning, likewise, for reasoning-summary deltas. It always returns the
// fully collected Result.
func Consume(resp *http.Response, onText, onReasoning func(string) error) (*Result, error) {
	defer resp.Body.Close()
	return processEvents(onText, onReasoning, func(fn func(SSEEvent) error) error {
		return parseSSE(resp.Body, fn)
	})
}

// processEvents runs the Codex event state machine, pulling events from pump
// (HTTP SSE or WebSocket) and assembling text, tool calls, reasoning and usage.
// pump must call fn for each event until the stream ends; if fn returns an
// error (terminal upstream error), pump should stop and return it.
func processEvents(onText, onReasoning func(string) error, pump func(func(SSEEvent) error) error) (*Result, error) {
	var text, reasoning strings.Builder
	var usage Usage
	var responseID string
	sawTerminal := false

	itemIDToCall := map[string]string{} // item_id -> call_id
	itemIDToName := map[string]string{}
	order := []string{}
	tools := map[string]*toolAccum{}
	argsSeen := map[string]bool{}

	ensure := func(callID, name string) *toolAccum {
		t, ok := tools[callID]
		if !ok {
			t = &toolAccum{id: callID, name: name}
			tools[callID] = t
			order = append(order, callID)
		}
		if name != "" && t.name == "" {
			t.name = name
		}
		return t
	}
	resolve := func(rawCallID string) string {
		if mapped, ok := itemIDToCall[rawCallID]; ok {
			return mapped
		}
		return rawCallID
	}

	var streamErr error
	perr := pump(func(evt SSEEvent) error {
		switch evt.Event {
		case "response.created", "response.in_progress", "response.queued":
			var e sseRespEnvelope
			if json.Unmarshal(evt.Data, &e) == nil && e.Response.ID != "" {
				responseID = e.Response.ID
			}

		case "response.output_text.delta":
			var d sseDelta
			if json.Unmarshal(evt.Data, &d) == nil && d.Delta != "" {
				text.WriteString(d.Delta)
				if onText != nil {
					return onText(d.Delta)
				}
			}

		case "response.reasoning_summary_text.delta":
			var d sseDelta
			if json.Unmarshal(evt.Data, &d) == nil && d.Delta != "" {
				reasoning.WriteString(d.Delta)
				if onReasoning != nil {
					return onReasoning(d.Delta)
				}
			}

		case "response.output_item.added":
			var it sseItem
			if json.Unmarshal(evt.Data, &it) == nil &&
				it.Item.Type == "function_call" && it.Item.CallID != "" && it.Item.Name != "" {
				itemIDToCall[it.Item.ID] = it.Item.CallID
				itemIDToName[it.Item.ID] = it.Item.Name
				ensure(it.Item.CallID, it.Item.Name)
			}

		case "response.function_call_arguments.delta":
			var fa sseFnArgs
			if json.Unmarshal(evt.Data, &fa) == nil {
				key := fa.key()
				callID := resolve(key)
				argsSeen[callID] = true
				ensure(callID, itemIDToName[key]).args.WriteString(fa.Delta)
			}

		case "response.function_call_arguments.done":
			var fa sseFnArgs
			if json.Unmarshal(evt.Data, &fa) == nil {
				key := fa.key()
				callID := resolve(key)
				name := fa.Name
				if name == "" {
					name = itemIDToName[key]
				}
				t := ensure(callID, name)
				if !argsSeen[callID] {
					t.args.Reset()
					t.args.WriteString(fa.Arguments)
				}
			}

		case "response.completed", "response.incomplete":
			var e sseRespEnvelope
			if json.Unmarshal(evt.Data, &e) == nil {
				if e.Response.ID != "" {
					responseID = e.Response.ID
				}
				if e.Response.Usage != nil {
					usage = Usage{
						InputTokens:     e.Response.Usage.InputTokens,
						OutputTokens:    e.Response.Usage.OutputTokens,
						CachedTokens:    e.Response.Usage.InputTokensDetails.CachedTokens,
						ReasoningTokens: e.Response.Usage.OutputTokensDetails.ReasoningTokens,
					}
				}
			}
			if evt.Event == "response.completed" {
				sawTerminal = true
			}

		case "error":
			streamErr = streamErrorToAPIError(extractCodexError(evt.Data))
			return streamErr

		case "response.failed":
			var env sseRespEnvelope
			if json.Unmarshal(evt.Data, &env) == nil && env.Response.ID != "" {
				responseID = env.Response.ID
			}
			sawTerminal = true
			streamErr = streamErrorToAPIError(extractCodexError(evt.Data))
			return streamErr
		}
		return nil
	})
	if streamErr != nil {
		return nil, streamErr
	}
	if perr != nil {
		return nil, fmt.Errorf("codex: read stream: %w", perr)
	}

	toolCalls := make([]ToolCall, 0, len(order))
	for _, callID := range order {
		t := tools[callID]
		toolCalls = append(toolCalls, ToolCall{ID: t.id, Name: t.name, Arguments: t.args.String()})
	}

	result := &Result{
		Text:       text.String(),
		Reasoning:  reasoning.String(),
		ToolCalls:  toolCalls,
		Usage:      usage,
		ResponseID: responseID,
	}

	if result.Text == "" && len(result.ToolCalls) == 0 && usage.OutputTokens == 0 && !sawTerminal {
		return nil, fmt.Errorf("codex: upstream closed the stream without producing a response")
	}
	return result, nil
}
