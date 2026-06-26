package ai

// Codex is a native, in-process provider that talks directly to the Codex
// Responses API — the backend the Codex CLI / Codex Desktop use
// (https://chatgpt.com/backend-api/codex/responses). It lets a ChatGPT
// subscription be used as a karma model provider.
//
// This is a Go port of the proxy core of
// https://github.com/icebear0828/codex-proxy: it authenticates with a ChatGPT
// OAuth token, translates the request to the Codex Responses wire format,
// streams the SSE response, and translates it back. Unlike codex-proxy it does
// not run a local HTTP server — the translation happens in-process (see
// internal/codex and ai/handlers_codex.go).
//
// Credentials are resolved automatically (see internal/codex.NewTokenSource):
//   - CODEX_ACCESS_TOKEN (+ optional CODEX_REFRESH_TOKEN / CODEX_ACCOUNT_ID), or
//   - the Codex CLI auth file at $CODEX_HOME/auth.json (default ~/.codex/auth.json)
//
// Optional overrides: CODEX_BASE_URL, CODEX_ORIGINATOR, CODEX_APP_VERSION.
//
// Usage:
//
//	kai := ai.NewKarmaAI(ai.GPT5_2Codex, ai.Codex)
//	resp, err := kai.GenerateFromSinglePrompt("Refactor this function ...")
//
// Reasoning effort and service tier can be set explicitly via
// WithReasoningEffort, or as a suffix on a custom model string (e.g.
// SetCustomModelVariant("gpt-5.5-high") or "gpt-5.4-mini-fast").
//
// The set of usable models depends on the account/plan and changes over time.
// Use ListCodexModels(ctx) to discover exactly what the signed-in account can
// run, then pass any returned id via SetCustomModelVariant. Note the Codex
// backend accepts its own slugs (e.g. "gpt-5.5"), which differ from the OpenAI
// public API ids.
const Codex Provider = "codex"

func init() {
	// Map existing BaseModel constants to their Codex slugs. Any model the
	// account supports also works via SetCustomModelVariant, and unmapped
	// BaseModels fall back to their raw string.
	if ProviderModelMapping[Codex] == nil {
		ProviderModelMapping[Codex] = make(map[BaseModel]string)
	}
	for bm, modelString := range map[BaseModel]string{
		GPT5_5:     "gpt-5.5",
		GPT5_4:     "gpt-5.4",
		GPT5_4Mini: "gpt-5.4-mini",
		// Older codex slugs (availability varies by account).
		GPT5_1Codex:    "gpt-5.1-codex",
		GPT5_1CodexMax: "gpt-5.1-codex-max",
		GPT5_2Codex:    "gpt-5.2-codex",
		GPT5_2CodexMax: "gpt-5.2-codex-max",
	} {
		ProviderModelMapping[Codex][bm] = modelString
	}
}
