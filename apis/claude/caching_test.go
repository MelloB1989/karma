package claude

import (
	"encoding/json"

	"github.com/MelloB1989/karma/models"
	"github.com/anthropics/anthropic-sdk-go"

	"strings"
	"testing"
)

func longPrompt() string { return strings.Repeat("system instructions. ", 300) } // ~6000 chars

// The default must cache without the caller doing anything — that is the whole
// design goal.
func TestCachesByDefault(t *testing.T) {
	blocks := (CachePolicy{}).systemBlocks(longPrompt())
	if len(blocks) != 1 {
		t.Fatalf("got %d blocks", len(blocks))
	}
	if !hasCacheControl(t, blocks[0]) {
		t.Error("a long system prompt should be marked cacheable by default")
	}
}

// Below the provider minimum a breakpoint buys nothing and still costs a write
// premium, so it must be left off.
func TestShortPromptsAreNotCached(t *testing.T) {
	blocks := (CachePolicy{}).systemBlocks("short")
	if len(blocks) != 1 {
		t.Fatalf("got %d blocks", len(blocks))
	}
	if hasCacheControl(t, blocks[0]) {
		t.Error("a short prompt should not be marked cacheable")
	}
}

func TestDisabledMeansDisabled(t *testing.T) {
	blocks := (CachePolicy{Disabled: true}).systemBlocks(longPrompt())
	if hasCacheControl(t, blocks[0]) {
		t.Error("caching was disabled but a breakpoint was still set")
	}
}

func TestEmptyPromptProducesNoBlocks(t *testing.T) {
	b := (CachePolicy{}).systemBlocks("")
	if b != nil {
		t.Errorf("expected no blocks for an empty prompt, got %d", len(b))
	}
}

// The text is preserved exactly — caching must not alter the prompt.
func TestPromptTextIsUnchanged(t *testing.T) {
	p := longPrompt()
	got := (CachePolicy{}).systemBlocks(p)[0].Text
	if got != p {
		t.Error("the system prompt text was modified")
	}
}

// A timestamp in the system prompt silently defeats caching; the library says
// so rather than letting the bill explain it.
func TestVolatilePrefixIsFlagged(t *testing.T) {
	if volatilePrefixWarning("You are an agent.\n## Current date & time\nRight now it is 3pm.") == "" {
		t.Error("a timestamped system prompt should be flagged as cache-hostile")
	}
	if volatilePrefixWarning("You are a helpful agent with tools.") != "" {
		t.Error("a stable prompt should not be flagged")
	}
}

func TestCacheStatsReportHits(t *testing.T) {
	if (CacheStats{Read: 0}).Hit() {
		t.Error("zero reads is not a hit")
	}
	if !(CacheStats{Read: 1200}).Hit() {
		t.Error("non-zero reads is a hit")
	}
}

// hasCacheControl reports whether the block actually serializes a breakpoint.
// cache_control is `omitzero`, so the wire format is the only real answer.
func hasCacheControl(t *testing.T, b anthropic.TextBlockParam) bool {
	t.Helper()
	raw, err := json.Marshal(b)
	if err != nil {
		t.Fatalf("marshal block: %v", err)
	}
	return strings.Contains(string(raw), `"cache_control"`)
}

// Per-turn context must actually reach the model. It used to be written by
// callers and read by nothing, so every scrap of it was silently dropped.
func TestContextIsDeliveredOnTheLastUserMessage(t *testing.T) {
	h := models.AIChatHistory{
		Context: "## Right now it is 3pm",
		Messages: []models.AIMessage{
			{Role: models.User, Message: "first"},
			{Role: models.Assistant, Message: "reply"},
			{Role: models.User, Message: "second"},
		},
	}
	got := processMessages(h)
	if len(got) != 3 {
		t.Fatalf("got %d messages", len(got))
	}
	last := blockText(got[2])
	if !strings.Contains(last, "Right now it is 3pm") {
		t.Errorf("context did not reach the last user message: %q", last)
	}
	if !strings.Contains(last, "second") {
		t.Errorf("the user's own message was lost: %q", last)
	}
	// Not on the earlier user turn — that would duplicate it every call.
	if strings.Contains(blockText(got[0]), "Right now") {
		t.Error("context leaked onto an earlier message")
	}
}

// Volatile context must stay OUT of the system block, or it changes the cached
// prefix on every call and prompt caching never hits.
func TestContextDoesNotTouchTheSystemBlock(t *testing.T) {
	blocks := (CachePolicy{}).systemBlocks(strings.Repeat("stable system prompt. ", 300))
	before := blocks[0].Text
	h := models.AIChatHistory{Context: "volatile timestamp", Messages: []models.AIMessage{{Role: models.User, Message: "hi"}}}
	_ = processMessages(h)
	after := (CachePolicy{}).systemBlocks(strings.Repeat("stable system prompt. ", 300))[0].Text
	if before != after {
		t.Error("the system block changed between calls; the cached prefix would never hit")
	}
}

func TestNoContextLeavesMessagesAlone(t *testing.T) {
	h := models.AIChatHistory{Messages: []models.AIMessage{{Role: models.User, Message: "only"}}}
	if got := blockText(processMessages(h)[0]); got != "only" {
		t.Errorf("message was altered with no context set: %q", got)
	}
}

func blockText(m anthropic.MessageParam) string {
	var b strings.Builder
	for _, c := range m.Content {
		if c.OfText != nil {
			b.WriteString(c.OfText.Text)
		}
	}
	return b.String()
}

// Tools are part of the cached prefix, so their order must not vary between
// calls. Ranging a Go map does vary, which is what defeated caching entirely.
func TestToolOrderIsStableAcrossCalls(t *testing.T) {
	cc := &ClaudeClient{FunctionTools: map[string]GoFunctionTool{}}
	for _, n := range []string{"zeta", "alpha", "mid", "beta", "omega", "delta", "gamma"} {
		cc.FunctionTools[n] = GoFunctionTool{Name: n, Description: "d", Parameters: map[string]any{}}
	}
	first := names(cc.convertGoFunctionToolsToAnthropic())
	// Many rounds: map order is random per range, so one repeat proves little.
	for i := 0; i < 50; i++ {
		if got := names(cc.convertGoFunctionToolsToAnthropic()); got != first {
			t.Fatalf("tool order changed between calls:\n  %s\n  %s", first, got)
		}
	}
	if first != "alpha,beta,delta,gamma,mid,omega,zeta" {
		t.Errorf("tools are not sorted by name: %s", first)
	}
}

func names(ts []anthropic.ToolUnionParam) string {
	var out []string
	for _, t := range ts {
		if t.OfTool != nil {
			out = append(out, t.OfTool.Name)
		}
	}
	return strings.Join(out, ",")
}
