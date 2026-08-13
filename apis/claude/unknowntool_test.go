package claude

import "testing"

func toolSet(names ...string) map[string]GoFunctionTool {
	m := make(map[string]GoFunctionTool, len(names))
	for _, n := range names {
		m[n] = GoFunctionTool{Name: n}
	}
	return m
}

func TestUnknownToolErrorSuggestsTheFamily(t *testing.T) {
	have := toolSet("whatsapp_list_chats", "whatsapp_search_messages", "comms_send", "shell_exec")

	err := unknownToolError("whatsapp_get_chat", have)
	if err == nil {
		t.Fatal("expected an error")
	}
	msg := err.Error()
	for _, want := range []string{"whatsapp_list_chats", "whatsapp_search_messages", "does not exist"} {
		if !contains(msg, want) {
			t.Errorf("error should mention %q, got: %s", want, msg)
		}
	}
	if contains(msg, "shell_exec") {
		t.Errorf("an unrelated family should not be suggested, got: %s", msg)
	}
	if contains(msg, "MCP") {
		t.Errorf("must not blame MCP when none is configured, got: %s", msg)
	}
}

func TestUnknownToolErrorFallsBackToTheFullList(t *testing.T) {
	err := unknownToolError("frobnicate", toolSet("comms_send", "shell_exec"))
	msg := err.Error()
	if !contains(msg, "comms_send") || !contains(msg, "shell_exec") {
		t.Errorf("with no near match, list what is available: %s", msg)
	}
}

func TestUnknownToolErrorMatchesDottedNames(t *testing.T) {
	err := unknownToolError("whatsapp_read", toolSet("whatsapp.read"))
	if !contains(err.Error(), "whatsapp.read") {
		t.Errorf("dotted and canonical names should match, got: %s", err.Error())
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (func() bool {
		for i := 0; i+len(sub) <= len(s); i++ {
			if s[i:i+len(sub)] == sub {
				return true
			}
		}
		return false
	})()
}
