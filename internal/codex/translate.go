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
	CallID    string `json:"call_id"`
	Name      string `json:"name"`
}

type sseError struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (e sseError) toError() error {
	msg := e.Error.Message
	if msg == "" {
		msg = e.Message
	}
	code := e.Error.Code
	if code == "" {
		code = e.Code
	}
	if code != "" {
		return fmt.Errorf("codex stream error (%s): %s", code, msg)
	}
	return fmt.Errorf("codex stream error: %s", msg)
}

type toolAccum struct {
	id   string
	name string
	args strings.Builder
}

// Consume reads a Codex Responses SSE stream to completion. When onText is
// non-nil it is invoked for each text delta (streaming); onReasoning, likewise,
// for reasoning-summary deltas. It always returns the fully collected Result.
func Consume(resp *http.Response, onText, onReasoning func(string) error) (*Result, error) {
	defer resp.Body.Close()

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
	perr := parseSSE(resp.Body, func(evt SSEEvent) error {
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
				callID := resolve(fa.CallID)
				argsSeen[callID] = true
				ensure(callID, itemIDToName[fa.CallID]).args.WriteString(fa.Delta)
			}

		case "response.function_call_arguments.done":
			var fa sseFnArgs
			if json.Unmarshal(evt.Data, &fa) == nil {
				callID := resolve(fa.CallID)
				name := fa.Name
				if name == "" {
					name = itemIDToName[fa.CallID]
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
			var e sseError
			_ = json.Unmarshal(evt.Data, &e)
			streamErr = e.toError()
			return streamErr

		case "response.failed":
			var e sseError
			_ = json.Unmarshal(evt.Data, &e)
			var env sseRespEnvelope
			if json.Unmarshal(evt.Data, &env) == nil && env.Response.ID != "" {
				responseID = env.Response.ID
			}
			sawTerminal = true
			streamErr = e.toError()
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
