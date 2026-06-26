package ai

import "github.com/MelloB1989/karma/config"

// ProxyProvider describes an unofficial, OpenAI-Chat-Completions-compatible proxy
// backend. These are community / self-hosted relays (for example codex-proxy,
// https://github.com/icebear0828/codex-proxy) that expose a `/v1/chat/completions`
// endpoint and translate it to some upstream API under the hood.
//
// Because every such proxy speaks the OpenAI wire format, all that is needed to
// support a new one is to register a ProxyProvider describing where it lives and
// which models it serves. The dispatch layer in ai.go automatically routes any
// registered proxy through the shared OpenAI-compatible handlers, so no changes
// to the switch statements are required when adding a proxy.
//
// To add a new proxy, create a file like proxy_codex.go and call
// RegisterProxyProvider from its init() function. See proxy_codex.go for a
// worked example.
type ProxyProvider struct {
	// Provider is the unique identifier passed to NewKarmaAI as the provider.
	Provider Provider
	// DefaultBaseURL is the OpenAI-compatible base URL, including the version
	// segment (e.g. "http://localhost:8080/v1").
	DefaultBaseURL string
	// BaseURLEnv, when non-empty and set in the environment, overrides
	// DefaultBaseURL at request time. Useful for self-hosted proxies whose URL
	// differs per deployment.
	BaseURLEnv string
	// APIKeyEnv is the name of the environment variable holding the bearer API
	// key sent as `Authorization: Bearer <key>`.
	APIKeyEnv string
	// Models maps karma BaseModels to the model string the proxy expects. When a
	// requested BaseModel is absent here, GetModelString falls back to the raw
	// base model string, so an empty map still works for arbitrary model names.
	Models map[BaseModel]string
	// SupportsMCP reports whether the proxy can execute MCP / function tools.
	SupportsMCP bool
}

// BaseURL resolves the effective base URL, honoring the BaseURLEnv override.
func (p ProxyProvider) BaseURL() string {
	if p.BaseURLEnv != "" {
		if v := config.GetEnvRaw(p.BaseURLEnv); v != "" {
			return v
		}
	}
	return p.DefaultBaseURL
}

// APIKey resolves the bearer API key from the environment, or "" if unset.
func (p ProxyProvider) APIKey() string {
	if p.APIKeyEnv == "" {
		return ""
	}
	return config.GetEnvRaw(p.APIKeyEnv)
}

// proxyRegistry holds all registered unofficial proxy providers, keyed by
// Provider. It is populated by RegisterProxyProvider, typically from package
// init() functions.
var proxyRegistry = map[Provider]ProxyProvider{}

// RegisterProxyProvider registers (or overrides) an unofficial proxy provider.
// It also merges the proxy's model mappings into ProviderModelMapping so that
// GetModelString resolves the correct model strings for it.
//
// This is safe to call from init(): package-level variable initializers (such as
// ProviderModelMapping) always run before any init function.
func RegisterProxyProvider(p ProxyProvider) {
	proxyRegistry[p.Provider] = p
	if len(p.Models) == 0 {
		return
	}
	if ProviderModelMapping[p.Provider] == nil {
		ProviderModelMapping[p.Provider] = make(map[BaseModel]string, len(p.Models))
	}
	for bm, modelString := range p.Models {
		ProviderModelMapping[p.Provider][bm] = modelString
	}
}

// lookupProxyProvider returns the registered proxy for a provider, if any.
func lookupProxyProvider(p Provider) (ProxyProvider, bool) {
	pp, ok := proxyRegistry[p]
	return pp, ok
}

// resolveProxy returns the registered proxy provider for the current model's
// provider, if it is an unofficial proxy.
func (kai *KarmaAI) resolveProxy() (ProxyProvider, bool) {
	return lookupProxyProvider(kai.Model.GetModelProvider())
}
