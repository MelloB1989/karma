package ai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/MelloB1989/karma/internal/codex"
	internalopenai "github.com/MelloB1989/karma/internal/openai"
	"github.com/MelloB1989/karma/models"
)

// handleCodexChatCompletion runs a (non-streaming) chat completion against the
// Codex Responses API, including an optional local tool-execution loop.
func (kai *KarmaAI) handleCodexChatCompletion(history *models.AIChatHistory) (*models.AIChatResponse, error) {
	start := time.Now()
	client, err := kai.newCodexClient()
	if err != nil {
		return nil, err
	}
	if err := kai.enforceRateLimit(); err != nil {
		return nil, err
	}

	ctx, cancel := kai.codexContext()
	defer cancel()

	instructions := kai.codexInstructions(history)
	messages := kai.codexMessages(history)
	tools, toolNames := kai.codexTools()
	execEnabled := kai.ToolsEnabled && kai.UseMCPExecution && len(tools) > 0

	maxPasses := kai.MaxToolPasses
	if maxPasses <= 0 {
		maxPasses = 1
	}

	var final *codex.Result
	for pass := 0; pass <= maxPasses; pass++ {
		req := codex.BuildRequest(codex.RequestOptions{
			Model:           kai.Model.GetModelString(),
			Instructions:    instructions,
			Messages:        messages,
			Tools:           tools,
			ReasoningEffort: kai.codexReasoningEffort(),
		})
		result, err := kai.codexGenerate(ctx, client, req)
		if err != nil {
			return nil, err
		}
		final = result

		if len(result.ToolCalls) == 0 || !execEnabled {
			break
		}
		// Replay the assistant turn (sanitized names, as Codex emitted them) +
		// execute tools under their original names so dispatch resolves them.
		messages = append(messages, codexAssistantTurn(result))
		for _, tc := range result.ToolCalls {
			out, terr := kai.executeCodexTool(ctx, restoreToolName(toolNames, tc.Name), tc.Arguments)
			if terr != nil {
				out = fmt.Sprintf("Error: %v", terr)
			}
			messages = append(messages, codex.Message{Role: "tool", ToolCallID: tc.ID, Content: out})
		}
	}

	return codexResult(final, toolNames, start), nil
}

// handleCodexStreamCompletion streams text deltas via callback. Tool calls are
// surfaced in the returned response (no automatic execution loop while
// streaming).
func (kai *KarmaAI) handleCodexStreamCompletion(history *models.AIChatHistory, callback func(chunk models.StreamedResponse) error) (*models.AIChatResponse, error) {
	start := time.Now()
	client, err := kai.newCodexClient()
	if err != nil {
		return nil, err
	}
	if err := kai.enforceRateLimit(); err != nil {
		return nil, err
	}

	ctx, cancel := kai.codexContext()
	defer cancel()

	tools, toolNames := kai.codexTools()
	req := codex.BuildRequest(codex.RequestOptions{
		Model:           kai.Model.GetModelString(),
		Instructions:    kai.codexInstructions(history),
		Messages:        kai.codexMessages(history),
		Tools:           tools,
		ReasoningEffort: kai.codexReasoningEffort(),
	})

	var lastErr error
	for attempt := 0; attempt <= codexMaxRetries; attempt++ {
		if attempt > 0 {
			if werr := codexWait(ctx, lastErr, attempt); werr != nil {
				return nil, werr
			}
		}
		// Retry is only safe before any text has been emitted to the callback;
		// otherwise a retry would duplicate already-streamed content.
		started := false
		onText := func(delta string) error {
			started = true
			return callback(models.StreamedResponse{AIResponse: delta, TimeTaken: -1})
		}
		result, err := client.Generate(ctx, req, onText, nil)
		if err == nil {
			return codexResult(result, toolNames, start), nil
		}
		lastErr = err
		if started || !codex.IsRetryable(err) {
			return nil, err
		}
	}
	return nil, lastErr
}

// codexMaxRetries is the number of extra attempts on transient Codex failures
// (HTTP 429/5xx or codeless mid-stream response.failed events).
const codexMaxRetries = 2

// codexMaxBackoff caps how long a single retry will wait, even if the backend
// asks for longer via Retry-After / resets_at.
const codexMaxBackoff = 60 * time.Second

// codexWait sleeps before a retry, honoring ctx. It prefers a server-provided
// Retry-After / resets_at delay (from a 429) and otherwise uses linear backoff.
func codexWait(ctx context.Context, err error, attempt int) error {
	delay := time.Duration(attempt) * 750 * time.Millisecond
	var ae *codex.APIError
	if errors.As(err, &ae) {
		if ra := ae.RetryAfter(); ra > 0 {
			delay = ra
		}
	}
	if delay > codexMaxBackoff {
		delay = codexMaxBackoff
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(delay):
		return nil
	}
}

// codexGenerate performs a single (non-streaming) model call, retrying transient
// failures with backoff.
func (kai *KarmaAI) codexGenerate(ctx context.Context, client *codex.Client, req *codex.ResponsesRequest) (*codex.Result, error) {
	var lastErr error
	for attempt := 0; attempt <= codexMaxRetries; attempt++ {
		if attempt > 0 {
			if werr := codexWait(ctx, lastErr, attempt); werr != nil {
				return nil, werr
			}
		}
		result, err := client.Generate(ctx, req, nil, nil)
		if err == nil {
			return result, nil
		}
		lastErr = err
		if !codex.IsRetryable(err) {
			return nil, err
		}
	}
	return nil, lastErr
}

// IsCodexRetryable reports whether err is a transient Codex failure (HTTP 429/5xx
// or a codeless mid-stream response.failed). Exposed so callers driving their own
// retry/fallback loop (e.g. a karmahelper) can classify Codex errors without
// string-matching.
func IsCodexRetryable(err error) bool {
	return codex.IsRetryable(err)
}

// CodexModel describes a model available to the authenticated Codex account.
type CodexModel struct {
	ID                        string   `json:"id"`
	DisplayName               string   `json:"display_name"`
	Description               string   `json:"description,omitempty"`
	SupportedReasoningEfforts []string `json:"supported_reasoning_efforts,omitempty"`
}

// ListCodexModels returns every model the locally-authenticated ChatGPT/Codex
// subscription exposes, discovered live from the Codex backend. Credentials are
// extracted automatically from $CODEX_HOME/auth.json (or the CODEX_* env vars).
// Any returned id can be used directly via SetCustomModelVariant.
func ListCodexModels(ctx context.Context) ([]CodexModel, error) {
	client, err := codex.Shared(codex.Config{})
	if err != nil {
		return nil, err
	}
	models, err := client.ListModels(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]CodexModel, len(models))
	for i, m := range models {
		out[i] = CodexModel{
			ID:                        m.ID,
			DisplayName:               m.DisplayName,
			Description:               m.Description,
			SupportedReasoningEfforts: m.SupportedReasoningEfforts,
		}
	}
	return out, nil
}

// ---- helpers ----

func (kai *KarmaAI) newCodexClient() (*codex.Client, error) {
	// Shared, process-wide client so the session cookie jar and token-refresh
	// state stay warm across calls.
	return codex.Shared(codex.Config{})
}

func (kai *KarmaAI) codexContext() (context.Context, context.CancelFunc) {
	if kai.RequestTimeout > 0 {
		return context.WithTimeout(context.Background(), kai.RequestTimeout)
	}
	return context.WithCancel(context.Background())
}

// codexInstructions folds the configured system message and any system/developer
// messages in the history into the Responses `instructions` field.
func (kai *KarmaAI) codexInstructions(history *models.AIChatHistory) string {
	parts := make([]string, 0, 2)
	if strings.TrimSpace(kai.SystemMessage) != "" {
		parts = append(parts, kai.SystemMessage)
	}
	for _, m := range history.Messages {
		if m.Role == models.System || m.Role == "developer" {
			if strings.TrimSpace(m.Message) != "" {
				parts = append(parts, m.Message)
			}
		}
	}
	return strings.Join(parts, "\n\n")
}

// codexMessages maps karma chat history to protocol-neutral codex messages
// (system/developer messages are folded into instructions and skipped here).
func (kai *KarmaAI) codexMessages(history *models.AIChatHistory) []codex.Message {
	out := make([]codex.Message, 0, len(history.Messages))
	for _, m := range history.Messages {
		if m.Role == models.System || m.Role == "developer" {
			continue
		}
		msg := codex.Message{
			Role:       string(m.Role),
			Content:    m.Message,
			Images:     m.Images,
			ToolCallID: m.ToolCallId,
		}
		for _, tc := range m.ToolCalls {
			// Sanitize so replayed assistant tool calls in the history don't
			// carry dotted names upstream (which 502 the Responses API). Outputs
			// correlate by call_id, so only the name needs rewriting.
			msg.ToolCalls = append(msg.ToolCalls, codex.ToolCall{
				ID:        tc.ID,
				Name:      sanitizeToolName(tc.Function.Name),
				Arguments: tc.Function.Arguments,
			})
		}
		out = append(out, msg)
	}
	return out
}

// codexTools builds Codex tool definitions from Go function tools and MCP tools.
// Tool names are sanitized to satisfy the OpenAI/Codex constraint
// (^[a-zA-Z0-9_-]{1,64}$); the returned map (sanitized -> original) lets the
// caller restore the real name on tool-call return. Returns nil when tools are
// disabled.
func (kai *KarmaAI) codexTools() ([]codex.Tool, map[string]string) {
	if !kai.ToolsEnabled {
		return nil, nil
	}
	var tools []codex.Tool
	nameMap := map[string]string{} // sanitized -> original (only when they differ)
	addTool := func(name, description string, params map[string]any, strict bool) {
		sanitized := sanitizeToolName(name)
		if sanitized != name {
			nameMap[sanitized] = name
		}
		tools = append(tools, codex.Tool{
			Type:        "function",
			Name:        sanitized,
			Description: description,
			Parameters:  params,
			Strict:      strict,
		})
	}
	for _, fn := range kai.GoFunctionTools {
		addTool(fn.Name, fn.Description, sanitizeToolSchema(map[string]any(fn.Parameters)), fn.Strict)
	}
	for _, t := range kai.MCPTools {
		addTool(t.ToolName, t.Description, sanitizeToolSchema(toSchemaMap(t.InputSchema)), false)
	}
	return tools, nameMap
}

// sanitizeToolSchema strips karma-internal artifacts from a tool's parameter
// schema before it goes upstream. NewFuncParams stuffs a "history" key into the
// FuncParams map (a runtime hack to pass chat history to handlers); it is not
// valid JSON Schema and the Codex Responses API returns 502 server_error on it.
// Handlers read history from the runtime call arguments, not the schema, so
// removing it is side-effect free. Clones only when a change is needed.
func sanitizeToolSchema(params map[string]any) map[string]any {
	if params == nil {
		return nil
	}
	if _, ok := params["history"]; !ok {
		return params
	}
	cloned := make(map[string]any, len(params))
	for k, v := range params {
		if k == "history" {
			continue
		}
		cloned[k] = v
	}
	return cloned
}

// sanitizeToolName rewrites a tool name to satisfy the OpenAI/Codex function
// name constraint ^[a-zA-Z0-9_-]{1,64}$. Any disallowed character — notably the
// dots in KARMAX names like "calendar.add" — becomes "_". The Codex Responses
// API 502s on invalid names rather than returning a clean error, so every name
// sent upstream must pass through this. It is deterministic, so a given original
// maps to the same sanitized name in both tool definitions and replayed history.
func sanitizeToolName(name string) string {
	var b strings.Builder
	b.Grow(len(name))
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '_', r == '-':
			b.WriteRune(r)
		default:
			b.WriteByte('_')
		}
	}
	s := b.String()
	if len(s) > 64 {
		s = s[:64]
	}
	return s
}

// restoreToolName maps a sanitized tool name back to its original via nameMap,
// falling back to the name itself when it was never rewritten.
func restoreToolName(nameMap map[string]string, sanitized string) string {
	if orig, ok := nameMap[sanitized]; ok {
		return orig
	}
	return sanitized
}

func (kai *KarmaAI) codexReasoningEffort() string {
	if kai.ReasoningEffort != nil {
		return string(*kai.ReasoningEffort)
	}
	return ""
}

// executeCodexTool runs a tool call locally: Go function tools by their handler,
// otherwise MCP tools via the multi-manager.
func (kai *KarmaAI) executeCodexTool(ctx context.Context, name, argsJSON string) (string, error) {
	args := map[string]any{}
	if strings.TrimSpace(argsJSON) != "" {
		if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
			return "", fmt.Errorf("invalid tool arguments for %q: %w", name, err)
		}
	}
	for _, fn := range kai.GoFunctionTools {
		if fn.Name == name && fn.Handler != nil {
			return fn.Handler(ctx, internalopenai.FuncParams(args))
		}
	}
	if len(kai.MCPServers) > 0 || len(kai.MCPTools) > 0 {
		res, err := kai.getOrBuildMultiMCP().CallTool(ctx, name, args)
		if err != nil {
			return "", err
		}
		if res.IsError {
			return "", fmt.Errorf("tool %q error: %s", name, res.Content)
		}
		return res.Content, nil
	}
	return "", fmt.Errorf("no handler registered for tool %q", name)
}

// codexAssistantTurn rebuilds the assistant message (text + tool calls) to
// replay before tool outputs in the next pass.
func codexAssistantTurn(r *codex.Result) codex.Message {
	msg := codex.Message{Role: "assistant", Content: r.Text}
	msg.ToolCalls = append(msg.ToolCalls, r.ToolCalls...)
	return msg
}

func codexResult(r *codex.Result, nameMap map[string]string, start time.Time) *models.AIChatResponse {
	res := &models.AIChatResponse{
		AIResponse:   r.Text,
		InputTokens:  r.Usage.InputTokens,
		OutputTokens: r.Usage.OutputTokens,
		Tokens:       r.Usage.InputTokens + r.Usage.OutputTokens,
		TimeTaken:    int(time.Since(start).Milliseconds()),
	}
	for _, tc := range r.ToolCalls {
		res.ToolCalls = append(res.ToolCalls, models.ToolCall{
			ID:       tc.ID,
			Type:     "function",
			Function: models.ToolCallFunction{Name: restoreToolName(nameMap, tc.Name), Arguments: tc.Arguments},
		})
	}
	return res
}

// toSchemaMap coerces an arbitrary JSON-schema value into a map[string]any.
func toSchemaMap(schema any) map[string]any {
	if schema == nil {
		return nil
	}
	if m, ok := schema.(map[string]any); ok {
		return m
	}
	data, err := json.Marshal(schema)
	if err != nil {
		return nil
	}
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		return nil
	}
	return m
}
