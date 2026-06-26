// Package codex implements a native client for the Codex Responses API — the
// same backend the Codex CLI / Codex Desktop talk to
// (POST https://chatgpt.com/backend-api/codex/responses).
//
// It is a Go port of the proxy core of https://github.com/icebear0828/codex-proxy:
// it authenticates with a ChatGPT account (OAuth access token, refreshed as
// needed), translates an OpenAI-style chat request into the Codex Responses
// wire format, streams the Server-Sent-Events response back, and translates the
// events into plain text + tool calls + usage.
//
// Unlike codex-proxy this does not stand up an HTTP server; it is an in-process
// client so karma can use a ChatGPT subscription as just another model provider.
package codex

import "encoding/json"

// Reasoning controls reasoning effort + summary mode on the Responses API.
type Reasoning struct {
	Effort  string `json:"effort,omitempty"`
	Summary string `json:"summary,omitempty"`
}

// ContentPart is a single part of a multimodal user message.
type ContentPart struct {
	Type     string `json:"type"`                // "input_text" | "input_image"
	Text     string `json:"text,omitempty"`      // for input_text
	ImageURL string `json:"image_url,omitempty"` // for input_image (url or data: URI)
}

// InputItem is one entry in the Responses `input` array. It is intentionally a
// flat struct covering every item shape the Codex API accepts:
//
//   - message:               {role, content}                     (Type == "")
//   - function_call:         {type, call_id, name, arguments}
//   - function_call_output:  {type, call_id, output}
//
// Content is either a string or a []ContentPart (for images).
type InputItem struct {
	Type      string `json:"type,omitempty"`
	Role      string `json:"role,omitempty"`
	Content   any    `json:"content,omitempty"`
	CallID    string `json:"call_id,omitempty"`
	Name      string `json:"name,omitempty"`
	Arguments string `json:"arguments,omitempty"`
	Output    string `json:"output,omitempty"`
}

// Tool is a function tool definition in the Codex Responses format. Note the
// flattened shape (name/parameters at the top level) — this differs from the
// OpenAI Chat Completions nesting under `function`.
type Tool struct {
	Type        string         `json:"type"` // "function"
	Name        string         `json:"name,omitempty"`
	Description string         `json:"description,omitempty"`
	Parameters  map[string]any `json:"parameters,omitempty"`
	Strict      bool           `json:"strict,omitempty"`
}

// TextFormat carries structured-output / JSON-mode configuration.
type TextFormat struct {
	Format struct {
		Type   string         `json:"type"` // "text" | "json_object" | "json_schema"
		Name   string         `json:"name,omitempty"`
		Schema map[string]any `json:"schema,omitempty"`
		Strict *bool          `json:"strict,omitempty"`
	} `json:"format"`
}

// ResponsesRequest is the body POSTed to /codex/responses. The Codex backend
// requires stream=true and store=false, so those fields are always serialized.
type ResponsesRequest struct {
	Model             string            `json:"model"`
	Instructions      string            `json:"instructions,omitempty"`
	Input             []InputItem       `json:"input"`
	Stream            bool              `json:"stream"`
	Store             bool              `json:"store"`
	Reasoning         *Reasoning        `json:"reasoning,omitempty"`
	ServiceTier       string            `json:"service_tier,omitempty"`
	Tools             []Tool            `json:"tools,omitempty"`
	ToolChoice        any               `json:"tool_choice,omitempty"`
	ParallelToolCalls *bool             `json:"parallel_tool_calls,omitempty"`
	Text              *TextFormat       `json:"text,omitempty"`
	PromptCacheKey    string            `json:"prompt_cache_key,omitempty"`
	ClientMetadata    map[string]string `json:"client_metadata,omitempty"`
	Include           []string          `json:"include,omitempty"`
}

// SSEEvent is a single parsed Server-Sent Event from the Codex stream.
type SSEEvent struct {
	Event string
	Data  json.RawMessage
}

// Usage holds token accounting extracted from the terminal response event.
type Usage struct {
	InputTokens     int
	OutputTokens    int
	CachedTokens    int
	ReasoningTokens int
}

// ToolCall is a function call emitted by the model.
type ToolCall struct {
	ID        string
	Name      string
	Arguments string
}

// Result is the fully-collected outcome of a (non-streaming) Codex response.
type Result struct {
	Text       string
	Reasoning  string
	ToolCalls  []ToolCall
	Usage      Usage
	ResponseID string
}
