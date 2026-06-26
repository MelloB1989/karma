package ai

import (
	"testing"
	"time"

	"github.com/MelloB1989/karma/internal/codex"
)

func TestSanitizeToolName(t *testing.T) {
	cases := map[string]string{
		"calendar.add":     "calendar_add",
		"app.push":         "app_push",
		"claude_code.call": "claude_code_call", // existing underscores preserved
		"whatsapp.read":    "whatsapp_read",
		"memory.ingest":    "memory_ingest",
		"valid_name-1":     "valid_name-1", // already valid -> unchanged
		"with space":       "with_space",
	}
	for in, want := range cases {
		if got := sanitizeToolName(in); got != want {
			t.Errorf("sanitizeToolName(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestRestoreToolName(t *testing.T) {
	m := map[string]string{"calendar_add": "calendar.add"}
	if got := restoreToolName(m, "calendar_add"); got != "calendar.add" {
		t.Errorf("restore mapped = %q, want calendar.add", got)
	}
	// Names never rewritten fall back to themselves (no reverse over-translation
	// of legitimate underscores like claude_code_call).
	if got := restoreToolName(m, "claude_code_call"); got != "claude_code_call" {
		t.Errorf("restore unmapped = %q, want claude_code_call", got)
	}
}

func TestCodexResultRestoresNames(t *testing.T) {
	r := &codex.Result{
		Text:      "done",
		ToolCalls: []codex.ToolCall{{ID: "c1", Name: "calendar_add", Arguments: `{"x":1}`}},
	}
	res := codexResult(r, map[string]string{"calendar_add": "calendar.add"}, time.Now())
	if len(res.ToolCalls) != 1 || res.ToolCalls[0].Function.Name != "calendar.add" {
		t.Fatalf("tool name not restored: %+v", res.ToolCalls)
	}
}
