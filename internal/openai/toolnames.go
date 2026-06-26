package openai

import "strings"

// sanitizeToolName rewrites a tool name to satisfy the OpenAI function-name
// constraint ^[a-zA-Z0-9_-]{1,64}$. Disallowed characters — e.g. the dots in
// names like "calendar.add" — become "_". It is a no-op for already-valid
// names, so non-dotted tools are unaffected. Deterministic, so a given original
// maps to the same sanitized name in tool definitions and replayed history.
func sanitizeToolName(name string) string {
	var b strings.Builder
	b.Grow(len(name))
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '_', r == '-':
			b.WriteRune(r)
		default:
			b.WriteByte('_')
		}
	}
	s := b.String()
	if len(s) > 64 {
		s = s[:64]
	}
	return s
}

// mapToolName sanitizes name for upstream use and records the reverse mapping
// (sanitized -> original) when it changed, so RestoreToolName can recover it.
func (o *OpenAI) mapToolName(name string) string {
	s := sanitizeToolName(name)
	if s != name {
		if o.toolNameMap == nil {
			o.toolNameMap = map[string]string{}
		}
		o.toolNameMap[s] = name
	}
	return s
}

// RestoreToolName maps a sanitized tool name back to the original, or returns
// it unchanged when it was never rewritten.
func (o *OpenAI) RestoreToolName(name string) string {
	if o.toolNameMap != nil {
		if orig, ok := o.toolNameMap[name]; ok {
			return orig
		}
	}
	return name
}
