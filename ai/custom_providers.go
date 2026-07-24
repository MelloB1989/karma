package ai

import (
	"sync"

	"github.com/MelloB1989/karma/config"
)

// CustomProvider describes any OpenAI-Chat-Completions-compatible backend that
// isn't one of the built-in providers — a self-hosted server (vLLM, LM Studio,
// Ollama's OpenAI-compatible endpoint, text-generation-webui, ...), an internal
// gateway, or a community relay (e.g. https://github.com/icebear0828/codex-proxy)
// that translates `/v1/chat/completions` to some upstream API.
//
// Because every such backend speaks the OpenAI wire format, all that is needed
// to support a new one is to describe where it lives and which models it
// serves, via RegisterCustomProvider. The dispatch layer in ai.go automatically
// routes any registered custom provider through the shared OpenAI-compatible
// handlers, so no changes to the switch statements are required when adding
// one.
//
// For a one-off or per-request endpoint that doesn't need a package-wide name,
// use the WithCustomProvider Option on NewKarmaAI instead — it bypasses this
// registry entirely.
type CustomProvider struct {
	// Provider is the unique identifier passed to NewKarmaAI as the provider.
	Provider Provider
	// DefaultBaseURL is the OpenAI-compatible base URL, including the version
	// segment (e.g. "http://localhost:8080/v1").
	DefaultBaseURL string
	// BaseURLEnv, when non-empty and set in the environment, overrides
	// DefaultBaseURL at request time. Useful for deployments where the URL
	// differs per environment.
	BaseURLEnv string
	// APIKey is the bearer API key sent as `Authorization: Bearer <key>`. Set
	// this directly when the key is already in hand (e.g. loaded from a
	// secrets manager). Takes precedence over APIKeyEnv.
	APIKey string
	// APIKeyEnv is the name of an environment variable holding the bearer API
	// key. Only consulted when APIKey is empty.
	APIKeyEnv string
	// Models maps karma BaseModels to the model string the provider expects.
	// When a requested BaseModel is absent here, GetModelString falls back to
	// the raw base model string, so an empty map still works for arbitrary
	// model names.
	Models map[BaseModel]string
	// SupportsMCP reports whether the provider can execute MCP / function tools.
	SupportsMCP bool
}

// BaseURL resolves the effective base URL, honoring the BaseURLEnv override.
func (p CustomProvider) BaseURL() string {
	if p.BaseURLEnv != "" {
		if v := config.GetEnvRaw(p.BaseURLEnv); v != "" {
			return v
		}
	}
	return p.DefaultBaseURL
}

// ResolveAPIKey returns the effective bearer API key: APIKey if set, otherwise
// the value read from APIKeyEnv, otherwise "".
func (p CustomProvider) ResolveAPIKey() string {
	if p.APIKey != "" {
		return p.APIKey
	}
	if p.APIKeyEnv == "" {
		return ""
	}
	return config.GetEnvRaw(p.APIKeyEnv)
}

// customProviderRegistryMu guards customProviderRegistry and the entries that
// RegisterCustomProvider merges into ProviderModelMapping. Registration
// usually happens at startup, but nothing prevents it from happening later
// (e.g. a multi-tenant app registering a provider on demand), so both the
// registry and ProviderModelMapping reads/writes are locked.
var customProviderRegistryMu sync.RWMutex

// customProviderRegistry holds all registered custom providers, keyed by
// Provider. It is populated by RegisterCustomProvider.
var customProviderRegistry = map[Provider]CustomProvider{}

// RegisterCustomProvider registers (or overrides) a custom OpenAI-compatible
// provider. It also merges the provider's model mappings into
// ProviderModelMapping so that GetModelString resolves the correct model
// strings for it.
//
// Safe to call from init() — package-level variable initializers (such as
// ProviderModelMapping) always run before any init function — and safe to
// call concurrently at any other time.
func RegisterCustomProvider(p CustomProvider) {
	customProviderRegistryMu.Lock()
	customProviderRegistry[p.Provider] = p
	customProviderRegistryMu.Unlock()

	if len(p.Models) == 0 {
		return
	}

	providerModelMappingMu.Lock()
	defer providerModelMappingMu.Unlock()
	if ProviderModelMapping[p.Provider] == nil {
		ProviderModelMapping[p.Provider] = make(map[BaseModel]string, len(p.Models))
	}
	for bm, modelString := range p.Models {
		ProviderModelMapping[p.Provider][bm] = modelString
	}
}

// lookupCustomProvider returns the registered custom provider for a provider,
// if any.
func lookupCustomProvider(p Provider) (CustomProvider, bool) {
	customProviderRegistryMu.RLock()
	defer customProviderRegistryMu.RUnlock()
	pp, ok := customProviderRegistry[p]
	return pp, ok
}

// resolveCustomProvider returns the registered custom provider for the
// current model's provider, if any.
func (kai *KarmaAI) resolveCustomProvider() (CustomProvider, bool) {
	return lookupCustomProvider(kai.Model.GetModelProvider())
}

// resolveOpenAICompatibleEndpoint resolves the base URL and API key to use
// for a provider not natively known to the dispatch switch in ai.go: the
// per-instance WithCustomProvider override takes precedence, then the
// CustomProvider registry. ok is false if neither is configured.
func (kai *KarmaAI) resolveOpenAICompatibleEndpoint() (baseURL, apiKey string, ok bool) {
	if kai.CustomProviderBaseURL != "" {
		return kai.CustomProviderBaseURL, kai.CustomProviderAPIKey, true
	}
	if pp, found := kai.resolveCustomProvider(); found {
		return pp.BaseURL(), pp.ResolveAPIKey(), true
	}
	return "", "", false
}
