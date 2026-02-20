package voice

import (
	"regexp"
	"strings"
)

var thinkBlockRegex = regexp.MustCompile(`(?is)<think\b[^>]*>.*?</think>`)

func stripThinkingTokens(text string) string {
	if strings.TrimSpace(text) == "" {
		return ""
	}

	clean := thinkBlockRegex.ReplaceAllString(text, "")
	clean = strings.ReplaceAll(clean, "<think>", "")
	clean = strings.ReplaceAll(clean, "</think>", "")

	return strings.TrimSpace(clean)
}
