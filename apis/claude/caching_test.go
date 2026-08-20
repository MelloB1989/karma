package claude

import (
	"bytes"
	"encoding/json"
	"log"
	"os"

	"github.com/MelloB1989/karma/models"
	"github.com/anthropics/anthropic-sdk-go"

	"strings"
	"testing"
)

func longPrompt() string { return strings.Repeat("system instructions. ", 300) } // ~6000 chars

// The default must cache without the caller doing anything — that is the whole
// design goal.
func TestCachesByDefault(t *testing.T) {
	blocks := (CachePolicy{}).systemBlocks(longPrompt(), 0)
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
	blocks := (CachePolicy{}).systemBlocks("short", 0)
	if len(blocks) != 1 {
		t.Fatalf("got %d blocks", len(blocks))
	}
	if hasCacheControl(t, blocks[0]) {
		t.Error("a short prompt should not be marked cacheable")
	}
}

func TestDisabledMeansDisabled(t *testing.T) {
	blocks := (CachePolicy{Disabled: true}).systemBlocks(longPrompt(), 0)
	if hasCacheControl(t, blocks[0]) {
		t.Error("caching was disabled but a breakpoint was still set")
	}
}

func TestEmptyPromptProducesNoBlocks(t *testing.T) {
	b := (CachePolicy{}).systemBlocks("", 0)
	if b != nil {
		t.Errorf("expected no blocks for an empty prompt, got %d", len(b))
	}
}

// The text is preserved exactly — caching must not alter the prompt.
func TestPromptTextIsUnchanged(t *testing.T) {
	p := longPrompt()
	got := (CachePolicy{}).systemBlocks(p, 0)[0].Text
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
	blocks := (CachePolicy{}).systemBlocks(strings.Repeat("stable system prompt. ", 300), 0)
	before := blocks[0].Text
	h := models.AIChatHistory{Context: "volatile timestamp", Messages: []models.AIMessage{{Role: models.User, Message: "hi"}}}
	_ = processMessages(h)
	after := (CachePolicy{}).systemBlocks(strings.Repeat("stable system prompt. ", 300), 0)[0].Text
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

// The prefix a breakpoint covers is tools AND system, so a short system prompt
// riding in front of large tool schemas must still cache.
//
// This is the case that was being missed: a 2,400-character system prompt with
// seven tools was ~11,800 input tokens a call — more than ten times the
// provider minimum — and was never cached, because only the system string was
// measured.
func TestToolsCountTowardTheCachedPrefix(t *testing.T) {
	short := strings.Repeat("retrieve memory. ", 140) // ~2400 chars, under the minimum alone
	if (CachePolicy{}).cacheable(len(short)) {
		t.Fatalf("precondition: %d chars should be under the minimum on its own", len(short))
	}
	blocks := (CachePolicy{}).systemBlocks(short, 9000)
	if !hasCacheControl(t, blocks[0]) {
		t.Error("a short system prompt in front of large tool schemas must still be cached")
	}
}

func TestToolCharsMeasuresNameDescriptionAndSchema(t *testing.T) {
	tool := anthropic.ToolUnionParam{OfTool: &anthropic.ToolParam{
		Name:        "memory_search",
		Description: anthropic.String("Search long-term memory for a fact."),
		InputSchema: anthropic.ToolInputSchemaParam{
			Properties: map[string]any{
				"query": map[string]any{"type": "string", "description": "what to look for"},
			},
		},
	}}
	n := toolChars([]anthropic.ToolUnionParam{tool})
	if n < len("memory_search")+len("Search long-term memory for a fact.") {
		t.Errorf("toolChars = %d; must count at least the name and description", n)
	}
	if toolChars(nil) != 0 {
		t.Error("no tools must measure zero")
	}
}

// The history breakpoint must land on the last message that will still render
// identically next turn. With per-turn Context injected into the last user
// message, that message is volatile and the boundary is the one before it.
func TestHistoryBoundarySkipsTheMessageContextIsInjectedInto(t *testing.T) {
	msgs := []anthropic.MessageParam{
		{Role: anthropic.MessageParamRoleUser},
		{Role: anthropic.MessageParamRoleAssistant},
		{Role: anthropic.MessageParamRoleUser}, // Context lands here
	}
	if got := historyBoundary(msgs, true); got != 1 {
		t.Errorf("historyBoundary with context = %d; want 1 (the assistant turn before the volatile user message)", got)
	}
	if got := historyBoundary(msgs, false); got != 2 {
		t.Errorf("historyBoundary without context = %d; want 2 (the last message is stable)", got)
	}
	if got := historyBoundary(msgs[:1], true); got != -1 {
		t.Errorf("historyBoundary on a single message = %d; want -1 (nothing stable to cache)", got)
	}
}

func TestHistoryIsMarkedWhenTheConversationIsWorthIt(t *testing.T) {
	long := strings.Repeat("a previous turn. ", 400) // ~6800 chars
	msgs := []anthropic.MessageParam{
		{Role: anthropic.MessageParamRoleUser, Content: []anthropic.ContentBlockParamUnion{
			{OfText: &anthropic.TextBlockParam{Text: long}}}},
		{Role: anthropic.MessageParamRoleAssistant, Content: []anthropic.ContentBlockParamUnion{
			{OfText: &anthropic.TextBlockParam{Text: long}}}},
		{Role: anthropic.MessageParamRoleUser, Content: []anthropic.ContentBlockParamUnion{
			{OfText: &anthropic.TextBlockParam{Text: "and now this"}}}},
	}
	cacheHistory(msgs, historyBoundary(msgs, true), 0, CachePolicy{})

	marked, err := json.Marshal(msgs[1].Content[len(msgs[1].Content)-1].OfText)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(marked), `"cache_control"`) {
		t.Error("the stable end of the history should carry a breakpoint")
	}
	volatile, err := json.Marshal(msgs[2].Content[0].OfText)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(volatile), `"cache_control"`) {
		t.Error("the message per-turn context is injected into must NOT be a breakpoint — it changes every turn")
	}
}

// A short conversation is not worth a write premium.
func TestShortHistoryIsNotMarked(t *testing.T) {
	msgs := []anthropic.MessageParam{
		{Role: anthropic.MessageParamRoleUser, Content: []anthropic.ContentBlockParamUnion{
			{OfText: &anthropic.TextBlockParam{Text: "hi"}}}},
		{Role: anthropic.MessageParamRoleAssistant, Content: []anthropic.ContentBlockParamUnion{
			{OfText: &anthropic.TextBlockParam{Text: "hello"}}}},
	}
	cacheHistory(msgs, historyBoundary(msgs, false), 0, CachePolicy{})
	raw, _ := json.Marshal(msgs[1].Content[0].OfText)
	if strings.Contains(string(raw), `"cache_control"`) {
		t.Error("a two-line conversation should not be cached")
	}
}

func TestDisabledSkipsHistoryToo(t *testing.T) {
	long := strings.Repeat("a previous turn. ", 400)
	msgs := []anthropic.MessageParam{
		{Role: anthropic.MessageParamRoleUser, Content: []anthropic.ContentBlockParamUnion{
			{OfText: &anthropic.TextBlockParam{Text: long}}}},
		{Role: anthropic.MessageParamRoleAssistant, Content: []anthropic.ContentBlockParamUnion{
			{OfText: &anthropic.TextBlockParam{Text: long}}}},
	}
	cacheHistory(msgs, historyBoundary(msgs, false), 0, CachePolicy{Disabled: true})
	raw, _ := json.Marshal(msgs[1].Content[0].OfText)
	if strings.Contains(string(raw), `"cache_control"`) {
		t.Error("Disabled must mean disabled for history as well as system")
	}
}

// The warning existed but nothing called it, so the one documented way to
// destroy caching was detected and never reported.
func TestAVolatileSystemPromptIsReportedToTheCaller(t *testing.T) {
	var got string
	p := CachePolicy{OnWarning: func(w string) { got = w }}
	p.systemBlocks("You are an agent.\n## Current date & time\n"+longPrompt(), 0)
	if got == "" {
		t.Error("a timestamp in the system prompt must be reported — it silently defeats caching")
	}

	got = ""
	p.systemBlocks(longPrompt(), 0)
	if got != "" {
		t.Errorf("a stable prompt must not warn, got %q", got)
	}
}

// With no callback wired the warning must still go somewhere. It previously
// went nowhere at all: nothing in the package called the function that
// produced it.
func TestAVolatilePromptWarnsEvenWithNoCallback(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(os.Stderr)
	warned.Delete("a timestamp in the system prompt changes its bytes every call")

	(CachePolicy{}).systemBlocks("You are an agent.\n## Current date & time\n"+longPrompt(), 0)
	if !strings.Contains(buf.String(), "prompt caching will not hit") {
		t.Errorf("expected a logged warning, got %q", buf.String())
	}

	buf.Reset()
	(CachePolicy{}).systemBlocks("You are an agent.\n## Current date & time\n"+longPrompt(), 0)
	if buf.Len() != 0 {
		t.Errorf("the same warning must not repeat on every call, got %q", buf.String())
	}
}

// Haiku will not cache a prefix a Sonnet would. Marking one anyway is not an
// error the provider reports — it simply ignores the breakpoint, so the only
// evidence is a cache_write of zero that nobody is looking at.
func TestHaikuHasAHigherFloorThanSonnet(t *testing.T) {
	const prefix = 6000 // ~1,500 tokens: over Sonnet's floor, under Haiku's

	if !(CachePolicy{Model: "global.anthropic.claude-sonnet-4-6"}).cacheable(prefix) {
		t.Error("Sonnet caches from 1024 tokens; 6000 chars should qualify")
	}
	if (CachePolicy{Model: "global.anthropic.claude-haiku-4-5-20251001-v1:0"}).cacheable(prefix) {
		t.Error("Haiku needs 2048 tokens; marking 6000 chars buys nothing and the provider ignores it")
	}
	if !(CachePolicy{Model: "global.anthropic.claude-haiku-4-5-20251001-v1:0"}).cacheable(12000) {
		t.Error("a large enough prefix must still cache on Haiku")
	}
	// An unknown or empty model must not silently adopt the stricter floor.
	if !(CachePolicy{}).cacheable(prefix) {
		t.Error("with no model named, assume the common floor")
	}
}

// The client knows its own model, so a caller never has to.
func TestTheClientFillsInItsOwnModel(t *testing.T) {
	cc := &ClaudeClient{Model: "global.anthropic.claude-haiku-4-5-20251001-v1:0"}
	if cc.cachePolicy().Model == "" {
		t.Fatal("the policy should inherit the client's model")
	}
	blocks := cc.systemBlocks(2605) // the real memory sub-agent prefix
	cc.SystemPrompt = strings.Repeat("retrieve. ", 240)
	blocks = cc.systemBlocks(2605)
	if len(blocks) == 1 && hasCacheControl(t, blocks[0]) {
		t.Error("a 5,000-character prefix on Haiku must not be marked: the provider ignores it")
	}
	// An explicit policy model wins over the client's.
	cc.Cache = CachePolicy{Model: "claude-sonnet-4-6"}
	if cc.cachePolicy().Model != "claude-sonnet-4-6" {
		t.Error("an explicitly set model must not be overwritten")
	}
}
