package codex

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/publicsuffix"
)

// DefaultBaseURL is the Codex backend the Codex CLI / Desktop use.
const DefaultBaseURL = "https://chatgpt.com/backend-api"

// Default client identity. We present as the cross-platform Codex CLI
// (originator "codex_cli_rs") with a User-Agent derived from the host OS/arch,
// rather than impersonating the macOS-only Codex Desktop, so the fingerprint is
// honest and generic across machines. All values are overridable via CODEX_*
// env vars in NewClient.
const (
	defaultOriginator = "codex_cli_rs"
	defaultAppVersion = "0.141.0" // last-resort fallback; real version read from the local install
	openAIBetaHeader  = "responses_websockets=2026-02-06"
	defaultResidency  = "us"
)

// APIError is returned for non-2xx HTTP responses and for mid-stream
// `error` / `response.failed` events from the Codex backend. Stream events are
// mapped to an HTTP-equivalent Status (see statusForCode) so callers can apply
// one consistent retry/recovery policy — mirroring codex-proxy's
// codexApiErrorFromEvent.
type APIError struct {
	Status  int
	Body    string
	Headers http.Header // response headers, when from an HTTP response
}

func (e *APIError) Error() string {
	detail := e.Body
	var env struct {
		Detail string `json:"detail"`
		Error  struct {
			Message string `json:"message"`
			Code    string `json:"code"`
			Type    string `json:"type"`
		} `json:"error"`
	}
	code := ""
	if json.Unmarshal([]byte(e.Body), &env) == nil {
		if env.Error.Message != "" {
			detail = env.Error.Message
		} else if env.Detail != "" {
			detail = env.Detail
		}
		code = env.Error.Code
		if code == "" {
			code = env.Error.Type
		}
	}
	if code != "" && code != "unknown" && code != "error" {
		return fmt.Sprintf("codex API error (%d): %s [code=%s]", e.Status, detail, code)
	}
	return fmt.Sprintf("codex API error (%d): %s", e.Status, detail)
}

// Retryable reports whether the failure is transient and worth retrying
// (timeouts, conflicts, rate limits, 5xx — including the generic 502 a codeless
// mid-stream failure maps to — and Cloudflare challenges).
func (e *APIError) Retryable() bool {
	if e.IsCloudflareChallenge() {
		return true
	}
	switch e.Status {
	case 408, 409, 425, 429, 500, 502, 503, 504:
		return true
	}
	return false
}

// IsCloudflareChallenge reports whether the response is a Cloudflare bot
// challenge (vs. an application error). Port of codex-proxy's isCfChallengeError:
// a 403/503 whose body or headers carry CF challenge markers.
func (e *APIError) IsCloudflareChallenge() bool {
	if e.Status != 403 && e.Status != 503 {
		return false
	}
	hay := strings.ToLower(e.Body + " " + headerHaystack(e.Headers))
	for _, marker := range []string{"cf-mitigated", "cf-chl-bypass", "_cf_chl", "cf_chl", "attention required", "just a moment"} {
		if strings.Contains(hay, marker) {
			return true
		}
	}
	return false
}

// RetryAfter returns how long to wait before retrying, from the Retry-After
// header or the error body's resets_in_seconds / resets_at (Codex 429s), or 0.
func (e *APIError) RetryAfter() time.Duration {
	if e.Headers != nil {
		if ra := strings.TrimSpace(e.Headers.Get("Retry-After")); ra != "" {
			if secs, err := strconv.Atoi(ra); err == nil && secs > 0 {
				return time.Duration(secs) * time.Second
			}
			if t, err := http.ParseTime(ra); err == nil {
				if d := time.Until(t); d > 0 {
					return d
				}
			}
		}
	}
	var env struct {
		Error struct {
			ResetsInSeconds float64 `json:"resets_in_seconds"`
			ResetsAt        float64 `json:"resets_at"`
		} `json:"error"`
	}
	if json.Unmarshal([]byte(e.Body), &env) == nil {
		if env.Error.ResetsInSeconds > 0 {
			return time.Duration(env.Error.ResetsInSeconds * float64(time.Second))
		}
		if env.Error.ResetsAt > 0 {
			if d := time.Until(time.Unix(int64(env.Error.ResetsAt), 0)); d > 0 {
				return d
			}
		}
	}
	return 0
}

func headerHaystack(h http.Header) string {
	if h == nil {
		return ""
	}
	var b strings.Builder
	for k, vals := range h {
		b.WriteString(k)
		b.WriteByte(' ')
		for _, v := range vals {
			b.WriteString(v)
			b.WriteByte(' ')
		}
	}
	return b.String()
}

// IsRetryable reports whether err (or anything it wraps) is a transient
// *APIError. HTTP- and stream-layer Codex failures both flow through APIError,
// so this is the single retry signal for the Codex provider.
func IsRetryable(err error) bool {
	var ae *APIError
	if errors.As(err, &ae) {
		return ae.Retryable()
	}
	return false
}

// Config configures a Client. Zero values fall back to sensible defaults and
// the CODEX_* environment variables.
type Config struct {
	BaseURL          string
	Originator       string
	AppVersion       string
	UserAgent        string // full User-Agent override; default derived from host
	Residency        string // x-openai-internal-codex-residency override
	ClientID         string // OAuth client id (refresh grant)
	TokenEndpoint    string // OAuth token endpoint
	HTTPClient       *http.Client
	DisableWebSocket bool // force HTTP-SSE transport (skip the WebSocket primary)
}

// Client talks to the Codex Responses API on behalf of a ChatGPT account.
type Client struct {
	cfg        Config
	tokens     *TokenSource
	httpClient *http.Client
	warmupOnce sync.Once
}

var (
	sharedMu      sync.Mutex
	sharedClients = map[string]*Client{}
)

// Shared returns a process-wide cached Client for the given config so the
// session cookie jar (cf_clearance / __cf_bm), warmup state, and token-refresh
// state stay warm across calls — the single-process analogue of codex-proxy's
// long-lived account sessions. Falls back to a fresh client on cache miss.
func Shared(cfg Config) (*Client, error) {
	key := strings.Join([]string{cfg.BaseURL, cfg.Originator, cfg.AppVersion, cfg.UserAgent, cfg.Residency}, "|")
	sharedMu.Lock()
	defer sharedMu.Unlock()
	if c, ok := sharedClients[key]; ok {
		return c, nil
	}
	c, err := NewClient(cfg)
	if err != nil {
		return nil, err
	}
	sharedClients[key] = c
	return c, nil
}

// NewClient builds a Codex client, resolving credentials via NewTokenSource.
func NewClient(cfg Config) (*Client, error) {
	if cfg.BaseURL == "" {
		cfg.BaseURL = envOr("CODEX_BASE_URL", DefaultBaseURL)
	}
	cfg.BaseURL = strings.TrimRight(cfg.BaseURL, "/")
	if cfg.Originator == "" {
		cfg.Originator = envOr("CODEX_ORIGINATOR", defaultOriginator)
	}
	if cfg.AppVersion == "" {
		cfg.AppVersion = resolveAppVersion()
	}
	if cfg.UserAgent == "" {
		cfg.UserAgent = envOr("CODEX_USER_AGENT", defaultUserAgent(cfg.Originator, cfg.AppVersion))
	}
	if cfg.Residency == "" {
		cfg.Residency = strings.TrimSpace(os.Getenv("CODEX_RESIDENCY"))
	}
	if !cfg.DisableWebSocket {
		if v := strings.TrimSpace(os.Getenv("CODEX_DISABLE_WEBSOCKET")); v == "1" || strings.EqualFold(v, "true") {
			cfg.DisableWebSocket = true
		}
	}
	httpClient := cfg.HTTPClient
	if httpClient == nil {
		// No client-wide timeout: streaming responses can run for minutes; the
		// caller's context governs cancellation. A cookie jar keeps the
		// Cloudflare session (cf_clearance / __cf_bm) warm across requests,
		// reducing bot challenges.
		jar, _ := cookiejar.New(&cookiejar.Options{PublicSuffixList: publicsuffix.List})
		httpClient = &http.Client{Jar: jar}
	}
	cfg.HTTPClient = httpClient

	ts, err := NewTokenSource(cfg.ClientID, cfg.TokenEndpoint, &http.Client{Timeout: 30 * time.Second})
	if err != nil {
		return nil, err
	}
	return &Client{cfg: cfg, tokens: ts, httpClient: httpClient}, nil
}

// AccountID returns the resolved ChatGPT account id (may be empty).
func (c *Client) AccountID() string {
	_, id, _ := c.tokens.Token(context.Background())
	return id
}

// Generate runs a Responses request over the most reliable available transport
// and consumes the stream, returning the fully collected Result. onText /
// onReasoning, when non-nil, receive streamed deltas.
//
// WebSocket is tried first — it is the transport codex-proxy uses for its
// primary endpoints and the one the backend advertises via prefer_websockets;
// the HTTP-SSE POST path is markedly more prone to transient 502s. On a
// WebSocket *transport* failure (before any output) it transparently falls back
// to HTTP SSE. A genuine upstream API error (rate limit, etc.) is surfaced as
// *APIError and not retried over the other transport.
func (c *Client) Generate(ctx context.Context, req *ResponsesRequest, onText, onReasoning func(string) error) (*Result, error) {
	c.prepareRequest(req)
	c.warmup(ctx)

	if !c.cfg.DisableWebSocket {
		started := false
		wsText := onText
		if onText != nil {
			wsText = func(s string) error { started = true; return onText(s) }
		}
		result, err := c.generateWS(ctx, req, wsText, onReasoning)
		if err == nil {
			return result, nil
		}
		// Surface real upstream errors (and anything emitted) without retrying
		// over HTTP, which would hit the same condition.
		var apiErr *APIError
		if errors.As(err, &apiErr) || started {
			return nil, err
		}
		// WebSocket transport failure before any output -> fall back to HTTP SSE.
	}
	return c.generateHTTP(ctx, req, onText, onReasoning)
}

// generateHTTP runs the request over HTTP SSE and consumes the stream.
func (c *Client) generateHTTP(ctx context.Context, req *ResponsesRequest, onText, onReasoning func(string) error) (*Result, error) {
	resp, err := c.createResponseHTTP(ctx, req)
	if err != nil {
		return nil, err
	}
	return Consume(resp, onText, onReasoning)
}

// CreateResponse POSTs a streaming Responses request over HTTP and returns the
// raw response. The caller owns response.Body and must Close it. Non-2xx
// responses are returned as *APIError. Prefer Generate, which adds the
// WebSocket transport + consumption.
func (c *Client) CreateResponse(ctx context.Context, req *ResponsesRequest) (*http.Response, error) {
	c.prepareRequest(req)
	c.warmup(ctx)
	return c.createResponseHTTP(ctx, req)
}

// prepareRequest enforces the required stream/store flags and attaches the
// stable installation id (header + client_metadata) so the backend load
// balancer pins us to one instance, keeping the prompt cache warm.
func (c *Client) prepareRequest(req *ResponsesRequest) {
	req.Stream = true
	req.Store = false
	if req.ClientMetadata == nil {
		req.ClientMetadata = map[string]string{}
	}
	req.ClientMetadata["x-codex-installation-id"] = getInstallationID()
}

// createResponseHTTP performs the POST /codex/responses. req must already be
// prepared (see prepareRequest) and warmup already run.
func (c *Client) createResponseHTTP(ctx context.Context, req *ResponsesRequest) (*http.Response, error) {
	token, accountID, err := c.tokens.Token(ctx)
	if err != nil {
		return nil, err
	}
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("codex: marshal request: %w", err)
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.cfg.BaseURL+"/codex/responses", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	c.applyHeaders(httpReq.Header, token, accountID, getInstallationID(), "text/event-stream", true)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("codex: request failed: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		errBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
		resp.Body.Close()
		return nil, &APIError{Status: resp.StatusCode, Body: string(errBody), Headers: resp.Header}
	}
	return resp, nil
}

// warmup performs a one-time GET /codex/usage to establish session cookies
// (cf_clearance, __cf_bm, …) before the first real request, so the POST looks
// like a continuous session rather than a cold start. Best-effort: any failure
// is ignored. Port of codex-proxy's CodexApi.warmup.
func (c *Client) warmup(ctx context.Context) {
	c.warmupOnce.Do(func() {
		if c.httpClient.Jar == nil {
			return // no cookie jar -> nothing to prime
		}
		token, accountID, err := c.tokens.Token(ctx)
		if err != nil {
			return
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.cfg.BaseURL+"/codex/usage", nil)
		if err != nil {
			return
		}
		c.applyHeaders(req.Header, token, accountID, getInstallationID(), "application/json", false)
		resp, err := c.httpClient.Do(req)
		if err != nil {
			return
		}
		io.Copy(io.Discard, io.LimitReader(resp.Body, 1<<20))
		resp.Body.Close()
	})
}

// applyHeaders sets the Codex CLI request headers. Accept-Encoding is left unset
// so Go's transport negotiates gzip and transparently decompresses the response.
func (c *Client) applyHeaders(h http.Header, token, accountID, installID, accept string, withContentType bool) {
	h.Set("Authorization", "Bearer "+token)
	if accountID != "" {
		h.Set("ChatGPT-Account-Id", accountID)
	}
	h.Set("originator", c.cfg.Originator)
	h.Set("version", c.cfg.AppVersion)
	h.Set("x-openai-internal-codex-residency", c.residency())
	h.Set("x-client-request-id", newUUID())
	h.Set("x-codex-installation-id", installID)
	h.Set("OpenAI-Beta", openAIBetaHeader)
	h.Set("User-Agent", c.cfg.UserAgent)
	if withContentType {
		h.Set("Content-Type", "application/json")
	}
	if accept != "" {
		h.Set("Accept", accept)
	}
}

// residency resolves the compute-residency header value: explicit config/env >
// the account's JWT claim > "us".
func (c *Client) residency() string {
	if c.cfg.Residency != "" {
		return c.cfg.Residency
	}
	if r := c.tokens.Residency(); r != "" {
		return r
	}
	return defaultResidency
}

// defaultUserAgent builds a Codex-CLI-style User-Agent from the host runtime,
// e.g. "codex_cli_rs/0.20.0 (linux; amd64)".
func defaultUserAgent(originator, version string) string {
	name := originator
	if name == "" {
		name = defaultOriginator
	}
	return fmt.Sprintf("%s/%s (%s; %s)", name, version, runtime.GOOS, runtime.GOARCH)
}

// ModelInfo describes a model available to the authenticated account.
type ModelInfo struct {
	ID                        string
	DisplayName               string
	Description               string
	SupportedReasoningEfforts []string
}

// ListModels discovers the models available to the account by probing the same
// Codex backend endpoints the Codex CLI uses, returning the flattened list. It
// returns an empty slice (no error) if the backend exposes no discovery
// endpoint — any model the account supports can still be requested by id.
func (c *Client) ListModels(ctx context.Context) ([]ModelInfo, error) {
	token, accountID, err := c.tokens.Token(ctx)
	if err != nil {
		return nil, err
	}
	installID := getInstallationID()
	endpoints := []string{
		c.cfg.BaseURL + "/codex/models?client_version=" + url.QueryEscape(c.cfg.AppVersion),
		c.cfg.BaseURL + "/models",
		c.cfg.BaseURL + "/sentinel/chat-requirements",
	}
	var lastErr error
	for _, endpoint := range endpoints {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
		if err != nil {
			lastErr = err
			continue
		}
		c.applyHeaders(req.Header, token, accountID, installID, "application/json", false)
		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = err
			continue
		}
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
		resp.Body.Close()
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			lastErr = &APIError{Status: resp.StatusCode, Body: string(body)}
			continue
		}
		if models := parseModelList(body); len(models) > 0 {
			return models, nil
		}
	}
	if lastErr != nil {
		return nil, lastErr
	}
	return []ModelInfo{}, nil
}

// backendModelEntry is a lenient view of a model entry from the backend. The
// shape varies across endpoints; categories nest their models under `models`.
type backendModelEntry struct {
	Slug                      string               `json:"slug"`
	ID                        string               `json:"id"`
	Name                      string               `json:"name"`
	DisplayName               string               `json:"display_name"`
	Description               string               `json:"description"`
	Models                    []backendModelEntry  `json:"models"`
	SupportedReasoningEfforts []reasoningEffortRow `json:"supported_reasoning_efforts"`
	SupportedReasoningLevels  []reasoningEffortRow `json:"supported_reasoning_levels"`
}

type reasoningEffortRow struct {
	ReasoningEffort string `json:"reasoning_effort"`
	Effort          string `json:"effort"`
}

func parseModelList(body []byte) []ModelInfo {
	var root map[string]json.RawMessage
	if err := json.Unmarshal(body, &root); err != nil {
		return nil
	}
	var raw []json.RawMessage
	if cm, ok := root["chat_models"]; ok {
		var obj map[string]json.RawMessage
		if json.Unmarshal(cm, &obj) == nil {
			_ = json.Unmarshal(obj["models"], &raw)
		}
	}
	if len(raw) == 0 {
		for _, key := range []string{"models", "data", "categories"} {
			if v, ok := root[key]; ok {
				if json.Unmarshal(v, &raw) == nil && len(raw) > 0 {
					break
				}
			}
		}
	}

	var out []ModelInfo
	seen := map[string]bool{}
	var walk func(json.RawMessage)
	walk = func(item json.RawMessage) {
		var e backendModelEntry
		if json.Unmarshal(item, &e) != nil {
			return
		}
		for _, sub := range e.Models {
			b, _ := json.Marshal(sub)
			walk(b)
		}
		id := firstNonEmpty(e.Slug, e.ID, e.Name)
		if id == "" || seen[id] {
			return
		}
		seen[id] = true
		var efforts []string
		for _, r := range append(e.SupportedReasoningEfforts, e.SupportedReasoningLevels...) {
			if eff := firstNonEmpty(r.ReasoningEffort, r.Effort); eff != "" {
				efforts = append(efforts, eff)
			}
		}
		out = append(out, ModelInfo{
			ID:                        id,
			DisplayName:               firstNonEmpty(e.DisplayName, e.Name, id),
			Description:               e.Description,
			SupportedReasoningEfforts: efforts,
		})
	}
	for _, item := range raw {
		walk(item)
	}
	return out
}

func envOr(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

// resolveAppVersion determines the Codex client version to advertise. It prefers
// CODEX_APP_VERSION, then the version recorded by the local Codex install
// ($CODEX_HOME/models_cache.json, then version.json), so the value matches the
// user's actual CLI — the backend gates model discovery on it. Falls back to a
// recent constant.
func resolveAppVersion() string {
	if v := strings.TrimSpace(os.Getenv("CODEX_APP_VERSION")); v != "" {
		return v
	}
	dir := filepath.Dir(authFilePath()) // $CODEX_HOME
	if raw, err := os.ReadFile(filepath.Join(dir, "models_cache.json")); err == nil {
		var d struct {
			ClientVersion string `json:"client_version"`
		}
		if json.Unmarshal(raw, &d) == nil && d.ClientVersion != "" {
			return d.ClientVersion
		}
	}
	if raw, err := os.ReadFile(filepath.Join(dir, "version.json")); err == nil {
		var d struct {
			LatestVersion string `json:"latest_version"`
		}
		if json.Unmarshal(raw, &d) == nil && d.LatestVersion != "" {
			return d.LatestVersion
		}
	}
	return defaultAppVersion
}

// ---- installation id (port of installation-id.ts) ----

var (
	uuidRE          = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
	installIDOnce   sync.Once
	cachedInstallID string
)

func getInstallationID() string {
	installIDOnce.Do(func() {
		// 1. ~/.codex/installation_id if it's a valid UUID (mirrors a real install).
		if home, err := os.UserHomeDir(); err == nil {
			if id := readUUIDFile(filepath.Join(home, ".codex", "installation_id")); id != "" {
				cachedInstallID = id
				return
			}
		}
		// 2. $CODEX_HOME/installation_id (our own persisted id).
		ourFile := filepath.Join(filepath.Dir(authFilePath()), "installation_id")
		if id := readUUIDFile(ourFile); id != "" {
			cachedInstallID = id
			return
		}
		// 3. Generate + persist.
		id := newUUID()
		_ = os.MkdirAll(filepath.Dir(ourFile), 0o755)
		_ = os.WriteFile(ourFile, []byte(id), 0o600)
		cachedInstallID = id
	})
	return cachedInstallID
}

func readUUIDFile(path string) string {
	raw, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	s := strings.ToLower(strings.TrimSpace(string(raw)))
	if uuidRE.MatchString(s) {
		return s
	}
	return ""
}

// newUUID returns a random RFC-4122 v4 UUID string.
func newUUID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		// Extremely unlikely; fall back to a time-derived value.
		return fmt.Sprintf("00000000-0000-4000-8000-%012x", time.Now().UnixNano())
	}
	b[6] = (b[6] & 0x0f) | 0x40 // version 4
	b[8] = (b[8] & 0x3f) | 0x80 // variant 10
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
