package claude

import (
	"encoding/json"
	"log"
	"strings"
	"sync"

	"github.com/anthropics/anthropic-sdk-go"
)

// Prompt caching, decided by the library rather than the caller.
//
// Anthropic bills a cached prefix at roughly a tenth of fresh input, which for
// an agent that resends the same tools, system prompt and history on every turn
// is most of the bill. Getting it right needs things a caller should not have to
// know: that the render order is tools -> system -> messages, so a breakpoint
// covers everything before it; that a prefix below the model's minimum silently
// does not cache; and that anything volatile above a breakpoint invalidates
// everything under it.
//
// So callers get it on by default and can turn it off. They do not place
// breakpoints, count tokens, or read provider documentation.

// minCacheableChars is the point below which caching is not attempted, in
// characters, per model family.
//
// The provider minimum is expressed in TOKENS and differs by model: 1024 for
// Sonnet and Opus, 2048 for Haiku. A prefix under its model's floor is not
// cached and not refused — the breakpoint is silently ignored, which is the
// hardest kind of failure to notice. Measured on Bedrock: a Haiku sub-agent
// with a 5,002-character prefix (~1,250 tokens) reported cache_read AND
// cache_write of exactly zero, call after call, while a Sonnet agent on the
// same build cached 16,600 tokens a turn.
//
// Roughly five characters per token, erring high: failing to cache a
// borderline prompt costs a little, marking prefixes that can never cache
// costs the write premium for nothing.
const (
	minCacheableCharsDefault = 5000  // 1024-token floor: Sonnet, Opus
	minCacheableCharsHaiku   = 10000 // 2048-token floor
)

// minCacheableCharsFor is the floor this model actually enforces.
func minCacheableCharsFor(model string) int {
	if strings.Contains(strings.ToLower(model), "haiku") {
		return minCacheableCharsHaiku
	}
	return minCacheableCharsDefault
}

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
	// Model is the model this prompt will be sent to, which decides the
	// minimum prefix size the provider will actually cache. Empty assumes the
	// lower, more common floor.
	Model string
	// OnWarning overrides where cache warnings go. Left nil they are logged
	// once each, because a warning nobody wired up is a warning nobody gets —
	// this one sat behind a function no code called at all. Advisory either
	// way: a caller may have a good reason, and a library that refuses to send
	// a prompt is worse than one that explains the bill.
	OnWarning func(string)
}

// cacheable reports whether a prefix of this size is worth a breakpoint.
//
// The size that matters is the WHOLE prefix the breakpoint would cover, not the
// system prompt alone. This used to measure only the system string, which meant
// a 2,400-character system prompt riding in front of seven tool schemas — about
// 11,800 input tokens a call, more than ten times the provider minimum — was
// never cached at all, and was re-billed in full on every request.
func (p CachePolicy) cacheable(prefixChars int) bool {
	if p.Disabled {
		return false
	}
	return prefixChars >= minCacheableCharsFor(p.Model)
}

// toolChars measures the tool definitions that render in front of the system
// prompt. Names, descriptions and schemas are all sent on every request and all
// count toward the cached prefix.
func toolChars(tools []anthropic.ToolUnionParam) int {
	n := 0
	for _, t := range tools {
		if t.OfTool == nil {
			continue
		}
		n += len(t.OfTool.Name) + len(t.OfTool.Description.Value)
		// The schema is measured by serialising it: it is an untyped tree, and
		// its JSON is exactly what goes on the wire and into the prefix.
		if raw, err := json.Marshal(t.OfTool.InputSchema); err == nil {
			n += len(raw)
		}
	}
	return n
}

// systemBlocks renders the system prompt, marking it as a cache breakpoint when
// the prefix it would cover is worth caching.
//
// toolsChars is the size of the tool definitions rendered ahead of it, which is
// why every call site must assemble its tools BEFORE asking for these blocks.
func (p CachePolicy) systemBlocks(systemPrompt string, toolsChars int) []anthropic.TextBlockParam {
	if systemPrompt == "" {
		return nil
	}
	if w := volatilePrefixWarning(systemPrompt); w != "" && !p.Disabled {
		if p.OnWarning != nil {
			p.OnWarning(w)
		} else {
			warnOnce(w)
		}
	}
	block := anthropic.TextBlockParam{Text: systemPrompt}
	if p.cacheable(toolsChars + len(systemPrompt)) {
		block.CacheControl = anthropic.NewCacheControlEphemeralParam()
	}
	return []anthropic.TextBlockParam{block}
}

// systemBlocks is the client's own policy applied to its prompt.
//
// The model is filled in from the client so a caller never has to know which
// floor applies to what they are calling.
func (cc *ClaudeClient) systemBlocks(toolsChars int) []anthropic.TextBlockParam {
	return cc.cachePolicy().systemBlocks(cc.SystemPrompt, toolsChars)
}

// cachePolicy is the caller's policy with the model filled in.
func (cc *ClaudeClient) cachePolicy() CachePolicy {
	p := cc.Cache
	if p.Model == "" {
		p.Model = string(cc.Model)
	}
	return p
}

// cacheHistory marks the conversation prefix so history is not re-billed in
// full every turn.
//
// A system breakpoint covers tools and system, which for a long conversation is
// the smaller half: with a median of twenty-nine messages a turn, history was
// 69% of the input tokens and none of it was cached.
//
// boundary is the last message whose rendered bytes will not change on the next
// turn. That is not simply the last message: processMessages injects the
// caller's per-turn Context — the clock, freshly retrieved memory — into the
// last USER message, so that message renders differently once the next turn
// makes it history. A breakpoint on it would be invalidated by its own contents
// every single turn. Everything before it is append-only and stable.
func cacheHistory(msgs []anthropic.MessageParam, boundary int, prefixChars int, p CachePolicy) {
	if p.Disabled || boundary < 0 || boundary >= len(msgs) {
		return
	}
	for i := 0; i <= boundary; i++ {
		prefixChars += messageChars(msgs[i])
	}
	if !p.cacheable(prefixChars) {
		return
	}
	blocks := msgs[boundary].Content
	if len(blocks) == 0 {
		return
	}
	// The breakpoint goes on the LAST block of the message: it marks the end of
	// the cached prefix, not the start.
	if last := blocks[len(blocks)-1]; last.OfText != nil {
		last.OfText.CacheControl = anthropic.NewCacheControlEphemeralParam()
	}
}

// historyBoundary is the index of the last message that is stable across turns.
//
// withContext reports whether the caller supplied per-turn Context, which
// processMessages injects into the last user message.
func historyBoundary(msgs []anthropic.MessageParam, withContext bool) int {
	if len(msgs) < 2 {
		return -1
	}
	if !withContext {
		return len(msgs) - 1
	}
	for i := len(msgs) - 1; i >= 0; i-- {
		if msgs[i].Role == anthropic.MessageParamRoleUser {
			return i - 1
		}
	}
	return len(msgs) - 1
}

func messageChars(m anthropic.MessageParam) int {
	n := 0
	for _, b := range m.Content {
		if b.OfText != nil {
			n += len(b.OfText.Text)
		}
	}
	return n
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
// silently defeat caching by changing its bytes every call.
// warnOnce logs each distinct warning a single time. The condition it reports
// is a property of the prompt, so repeating it on every call of a long-running
// agent would bury it in its own noise.
func warnOnce(w string) {
	if _, seen := warned.LoadOrStore(w, true); !seen {
		log.Printf("karma: prompt caching will not hit — %s", w)
	}
}

var warned sync.Map

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
