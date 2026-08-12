package claude

import (
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
)

// Prompt caching, decided by the library rather than the caller.
//
// Anthropic bills a cached prefix at roughly a tenth of fresh input, which for
// an agent that resends the same system prompt and tool definitions on every
// turn is most of the bill. Getting it right needs three things a caller should
// not have to know: that the render order is tools -> system -> messages, so
// the breakpoint belongs on the last system block to cover both; that a prefix
// below the model's minimum silently does not cache; and that anything volatile
// above the breakpoint invalidates the whole thing.
//
// So callers get it on by default and can turn it off. They do not place
// breakpoints, count tokens, or read provider documentation.

// minCacheableChars is the point below which caching is not attempted.
//
// The provider minimum is expressed in tokens (1024 for most models, 2048 for
// some) and a prefix under it caches nothing while still being billed as a
// cache write. Four characters per token is the usual rough conversion, and
// this errs high: failing to cache a borderline prompt costs a little, paying
// write premiums on prompts that never cache costs more.
const minCacheableChars = 5000

// CachePolicy is how a client caches. The zero value is the default: cache when
// it is worth it.
//
// There is no TTL knob because the pinned SDK exposes none — cache_control only
// takes {"type":"ephemeral"}, which is the five-minute cache. An agent that
// answers messages is well inside that window anyway; a one-hour option would
// need an SDK bump and is not worth pretending to support.
type CachePolicy struct {
	// Disabled turns caching off entirely.
	Disabled bool
}

// cacheable reports whether this prompt is worth a breakpoint. A system prompt
// is the only thing cached here: messages grow and change every turn, and the
// tool definitions ride in front of the system block anyway.
func (p CachePolicy) cacheable(systemPrompt string) bool {
	if p.Disabled {
		return false
	}
	return len(systemPrompt) >= minCacheableChars
}

// systemBlocks renders the system prompt, marking it as a cache breakpoint when
// that is worth doing. Returning the blocks rather than mutating the request
// keeps the three call sites (message, tool-enabled message, stream) identical.
func (p CachePolicy) systemBlocks(systemPrompt string) []anthropic.TextBlockParam {
	if systemPrompt == "" {
		return nil
	}
	block := anthropic.TextBlockParam{Text: systemPrompt}
	if p.cacheable(systemPrompt) {
		block.CacheControl = anthropic.NewCacheControlEphemeralParam()
	}
	return []anthropic.TextBlockParam{block}
}

// systemBlocks is the client's own policy applied to its prompt.
func (cc *ClaudeClient) systemBlocks() []anthropic.TextBlockParam {
	return cc.Cache.systemBlocks(cc.SystemPrompt)
}

// CacheStats reports what the cache did on one call, so a caller can tell it is
// working without reading provider documentation or the raw usage object.
type CacheStats struct {
	Read    int64 // tokens served from cache, billed at ~10%
	Written int64 // tokens written to cache this call, billed at a premium
}

// Hit reports whether any of this call's prefix came from cache.
func (c CacheStats) Hit() bool { return c.Read > 0 }

func cacheStatsFrom(u anthropic.Usage) CacheStats {
	return CacheStats{Read: u.CacheReadInputTokens, Written: u.CacheCreationInputTokens}
}

// volatilePrefixWarning names the things that, placed in a system prompt,
// silently defeat caching by changing its bytes every call. Returned for
// logging rather than enforced: a caller may have a good reason, and a library
// that refuses to send a prompt is worse than one that explains the bill.
func volatilePrefixWarning(systemPrompt string) string {
	lower := strings.ToLower(systemPrompt)
	for marker, why := range map[string]string{
		"current date & time": "a timestamp in the system prompt changes its bytes every call",
		"right now it is":     "a timestamp in the system prompt changes its bytes every call",
	} {
		if strings.Contains(lower, marker) {
			return why
		}
	}
	return ""
}
