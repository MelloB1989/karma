package openai

import "testing"

func TestSanitizeToolName(t *testing.T) {
	cases := map[string]string{
		"calendar.add":     "calendar_add",
		"claude_code.call": "claude_code_call",
		"plain_name-1":     "plain_name-1",
	}
	for in, want := range cases {
		if got := sanitizeToolName(in); got != want {
			t.Errorf("sanitizeToolName(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestMapAndRestoreToolName(t *testing.T) {
	o := &OpenAI{}
	if got := o.mapToolName("calendar.add"); got != "calendar_add" {
		t.Fatalf("mapToolName = %q, want calendar_add", got)
	}
	if got := o.RestoreToolName("calendar_add"); got != "calendar.add" {
		t.Errorf("RestoreToolName = %q, want calendar.add", got)
	}
	// Valid names are not recorded and restore to themselves.
	if got := o.mapToolName("plain"); got != "plain" {
		t.Errorf("mapToolName(plain) = %q", got)
	}
	if got := o.RestoreToolName("plain"); got != "plain" {
		t.Errorf("RestoreToolName(plain) = %q", got)
	}
}
