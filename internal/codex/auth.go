package codex

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const (
	// DefaultOAuthClientID is the public OAuth client id used by Codex (and
	// codex-proxy) for the refresh-token grant.
	DefaultOAuthClientID = "app_EMoamEEZ73f0CkXaXp7hrann"
	// DefaultOAuthTokenEndpoint is the OpenAI OAuth token endpoint.
	DefaultOAuthTokenEndpoint = "https://auth.openai.com/oauth/token"
	// refreshMargin refreshes the access token this long before it actually
	// expires, mirroring codex-proxy's refresh_margin_seconds default.
	refreshMargin = 5 * time.Minute
)

// TokenSource provides a valid ChatGPT access token + account id for Codex
// requests, transparently refreshing the token when it is close to expiry.
//
// It is safe for concurrent use. Tokens are loaded from (in order):
//
//  1. The CODEX_ACCESS_TOKEN env var (with optional CODEX_REFRESH_TOKEN /
//     CODEX_ACCOUNT_ID), or
//  2. The Codex CLI auth file at $CODEX_HOME/auth.json (default
//     ~/.codex/auth.json), supporting both the nested `tokens` object the CLI
//     writes and a flat token layout.
//
// On refresh the new tokens are written back to the auth file when one was the
// source, so the refreshed access token survives process restarts.
type TokenSource struct {
	clientID      string
	tokenEndpoint string
	httpClient    *http.Client

	mu           sync.Mutex
	accessToken  string
	refreshToken string
	accountID    string
	residency    string    // chatgpt_compute_residency claim ("" -> caller default)
	expiresAt    time.Time // zero means "derive from JWT / unknown"
	authFilePath string    // non-empty when tokens came from a file (for persistence)
}

// cliAuthFile mirrors the Codex CLI ~/.codex/auth.json layout. The CLI nests
// tokens under "tokens"; some tools write them flat. Both are supported.
type cliAuthFile struct {
	OpenAIAPIKey *string `json:"OPENAI_API_KEY,omitempty"`
	Tokens       *struct {
		IDToken      string `json:"id_token,omitempty"`
		AccessToken  string `json:"access_token,omitempty"`
		RefreshToken string `json:"refresh_token,omitempty"`
		AccountID    string `json:"account_id,omitempty"`
	} `json:"tokens,omitempty"`
	LastRefresh string `json:"last_refresh,omitempty"`

	// Flat fallback layout.
	AccessToken  string `json:"access_token,omitempty"`
	RefreshToken string `json:"refresh_token,omitempty"`
	IDToken      string `json:"id_token,omitempty"`
	ExpiresAt    int64  `json:"expires_at,omitempty"`
}

// tokenResponse is the OAuth token endpoint response.
type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	IDToken      string `json:"id_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
}

// NewTokenSource builds a TokenSource, resolving tokens from the environment or
// the Codex CLI auth file. clientID/tokenEndpoint may be empty to use defaults.
func NewTokenSource(clientID, tokenEndpoint string, httpClient *http.Client) (*TokenSource, error) {
	if clientID == "" {
		clientID = DefaultOAuthClientID
	}
	if tokenEndpoint == "" {
		tokenEndpoint = DefaultOAuthTokenEndpoint
	}
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second}
	}

	ts := &TokenSource{
		clientID:      clientID,
		tokenEndpoint: tokenEndpoint,
		httpClient:    httpClient,
	}

	if tok := os.Getenv("CODEX_ACCESS_TOKEN"); tok != "" {
		ts.accessToken = strings.TrimSpace(tok)
		ts.refreshToken = strings.TrimSpace(os.Getenv("CODEX_REFRESH_TOKEN"))
		ts.accountID = strings.TrimSpace(os.Getenv("CODEX_ACCOUNT_ID"))
		ts.finalizeLoad()
		return ts, nil
	}

	if err := ts.loadFromFile(authFilePath()); err != nil {
		return nil, err
	}
	ts.finalizeLoad()
	return ts, nil
}

// authFilePath resolves $CODEX_HOME/auth.json (default ~/.codex/auth.json).
func authFilePath() string {
	home := os.Getenv("CODEX_HOME")
	if home == "" {
		if h, err := os.UserHomeDir(); err == nil {
			home = filepath.Join(h, ".codex")
		}
	}
	return filepath.Join(home, "auth.json")
}

func (ts *TokenSource) loadFromFile(path string) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("codex: no credentials: set CODEX_ACCESS_TOKEN or log in with the Codex CLI (%s): %w", path, err)
	}
	var f cliAuthFile
	if err := json.Unmarshal(raw, &f); err != nil {
		return fmt.Errorf("codex: failed to parse %s: %w", path, err)
	}
	if f.Tokens != nil {
		ts.accessToken = f.Tokens.AccessToken
		ts.refreshToken = f.Tokens.RefreshToken
		ts.accountID = f.Tokens.AccountID
	}
	if ts.accessToken == "" {
		ts.accessToken = f.AccessToken
		ts.refreshToken = f.RefreshToken
		if f.ExpiresAt > 0 {
			ts.expiresAt = time.Unix(f.ExpiresAt, 0)
		}
	}
	if ts.accessToken == "" {
		return fmt.Errorf("codex: %s does not contain an access_token", path)
	}
	ts.authFilePath = path
	return nil
}

// finalizeLoad derives account id, residency and expiry from the JWT when not
// already known.
func (ts *TokenSource) finalizeLoad() {
	if ts.accountID == "" {
		ts.accountID = accountIDFromJWT(ts.accessToken)
	}
	if ts.residency == "" {
		ts.residency = residencyFromJWT(ts.accessToken)
	}
	if ts.expiresAt.IsZero() {
		if exp, ok := expiryFromJWT(ts.accessToken); ok {
			ts.expiresAt = exp
		}
	}
}

// Residency returns the account's compute residency (e.g. "us"), or "" if the
// token does not carry the claim.
func (ts *TokenSource) Residency() string {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	return ts.residency
}

// Token returns a currently-valid access token and account id, refreshing if
// the token is expired (or within the refresh margin) and a refresh token is
// available.
func (ts *TokenSource) Token(ctx context.Context) (accessToken, accountID string, err error) {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	if !ts.needsRefreshLocked() {
		return ts.accessToken, ts.accountID, nil
	}
	if ts.refreshToken == "" {
		// Can't refresh; return what we have and let the upstream 401 surface.
		return ts.accessToken, ts.accountID, nil
	}
	if err := ts.refreshLocked(ctx); err != nil {
		return "", "", err
	}
	return ts.accessToken, ts.accountID, nil
}

func (ts *TokenSource) needsRefreshLocked() bool {
	if ts.expiresAt.IsZero() {
		return false // unknown expiry — assume valid, upstream 401 triggers nothing here
	}
	return time.Now().After(ts.expiresAt.Add(-refreshMargin))
}

func (ts *TokenSource) refreshLocked(ctx context.Context) error {
	form := url.Values{
		"grant_type":    {"refresh_token"},
		"client_id":     {ts.clientID},
		"refresh_token": {ts.refreshToken},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, ts.tokenEndpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return fmt.Errorf("codex: build refresh request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := ts.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("codex: token refresh failed: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("codex: token refresh failed (%d): %s", resp.StatusCode, string(body))
	}

	var tr tokenResponse
	if err := json.Unmarshal(body, &tr); err != nil {
		return fmt.Errorf("codex: parse refresh response: %w", err)
	}
	if tr.AccessToken == "" {
		return fmt.Errorf("codex: token refresh returned no access_token")
	}

	ts.accessToken = tr.AccessToken
	if tr.RefreshToken != "" {
		ts.refreshToken = tr.RefreshToken
	}
	if id := accountIDFromJWT(tr.AccessToken); id != "" {
		ts.accountID = id
	}
	if res := residencyFromJWT(tr.AccessToken); res != "" {
		ts.residency = res
	}
	if tr.ExpiresIn > 0 {
		ts.expiresAt = time.Now().Add(time.Duration(tr.ExpiresIn) * time.Second)
	} else if exp, ok := expiryFromJWT(tr.AccessToken); ok {
		ts.expiresAt = exp
	}

	ts.persistLocked(tr.IDToken)
	return nil
}

// persistLocked writes refreshed tokens back to the auth file (best effort),
// preserving the Codex CLI nested layout so the CLI keeps working too.
func (ts *TokenSource) persistLocked(idToken string) {
	if ts.authFilePath == "" {
		return
	}
	var f cliAuthFile
	if raw, err := os.ReadFile(ts.authFilePath); err == nil {
		_ = json.Unmarshal(raw, &f)
	}
	if f.Tokens == nil {
		f.Tokens = &struct {
			IDToken      string `json:"id_token,omitempty"`
			AccessToken  string `json:"access_token,omitempty"`
			RefreshToken string `json:"refresh_token,omitempty"`
			AccountID    string `json:"account_id,omitempty"`
		}{}
	}
	f.Tokens.AccessToken = ts.accessToken
	f.Tokens.RefreshToken = ts.refreshToken
	if ts.accountID != "" {
		f.Tokens.AccountID = ts.accountID
	}
	if idToken != "" {
		f.Tokens.IDToken = idToken
	}
	f.LastRefresh = time.Now().UTC().Format(time.RFC3339)
	// Clear flat fields so we don't keep stale duplicates.
	f.AccessToken, f.RefreshToken, f.IDToken, f.ExpiresAt = "", "", "", 0

	if out, err := json.MarshalIndent(&f, "", "  "); err == nil {
		_ = os.WriteFile(ts.authFilePath, out, 0o600)
	}
}

// ---- JWT helpers (no signature verification, payload extraction only) ----

func decodeJWTPayload(token string) map[string]any {
	parts := strings.Split(token, ".")
	if len(parts) < 2 {
		return nil
	}
	data, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		// Tolerate padded base64url.
		data, err = base64.URLEncoding.DecodeString(parts[1])
		if err != nil {
			return nil
		}
	}
	var payload map[string]any
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil
	}
	return payload
}

// accountIDFromJWT extracts the chatgpt_account_id claim from
// `https://api.openai.com/auth`.
func accountIDFromJWT(token string) string {
	payload := decodeJWTPayload(token)
	if payload == nil {
		return ""
	}
	auth, ok := payload["https://api.openai.com/auth"].(map[string]any)
	if !ok {
		return ""
	}
	id, _ := auth["chatgpt_account_id"].(string)
	return id
}

// residencyFromJWT extracts the chatgpt_compute_residency claim (lower-cased).
func residencyFromJWT(token string) string {
	payload := decodeJWTPayload(token)
	if payload == nil {
		return ""
	}
	auth, ok := payload["https://api.openai.com/auth"].(map[string]any)
	if !ok {
		return ""
	}
	res, _ := auth["chatgpt_compute_residency"].(string)
	return strings.ToLower(strings.TrimSpace(res))
}

func expiryFromJWT(token string) (time.Time, bool) {
	payload := decodeJWTPayload(token)
	if payload == nil {
		return time.Time{}, false
	}
	exp, ok := payload["exp"].(float64)
	if !ok {
		return time.Time{}, false
	}
	return time.Unix(int64(exp), 0), true
}
