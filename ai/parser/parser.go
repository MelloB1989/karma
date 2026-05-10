package parser

import (
	"encoding/json"
	"fmt"
	"log"
	"reflect"
	"regexp"
	"slices"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/MelloB1989/karma/ai"
	"github.com/MelloB1989/karma/models"
)

type Parser struct {
	client     *ai.KarmaAI
	maxRetries int
	debug      bool
}

type ParserOption func(*Parser)

func WithMaxRetries(retries int) ParserOption {
	return func(p *Parser) { p.maxRetries = retries }
}

func WithAIClient(client *ai.KarmaAI) ParserOption {
	return func(p *Parser) { p.client = client }
}

func WithDebug(debug bool) ParserOption {
	return func(p *Parser) { p.debug = debug }
}

func NewParser(opts ...ParserOption) *Parser {
	p := &Parser{maxRetries: 3}
	for _, opt := range opts {
		opt(p)
	}
	if p.client == nil {
		p.client = ai.NewKarmaAI(ai.GPT4oMini, ai.OpenAI)
	}
	return p
}

func (p *Parser) Parse(prompt, context string, output any) (time.Duration, int, error) {
	start := time.Now()
	t := reflect.TypeOf(output)

	if t.Kind() != reflect.Ptr || t.Elem().Kind() != reflect.Struct {
		return 0, 0, fmt.Errorf("output must be pointer to struct")
	}

	fullPrompt := buildPrompt(t.Elem(), prompt, context)

	var lastErr error
	retries := p.maxRetries

	for retries > 0 {
		resp, err := p.client.GenerateFromSinglePrompt(fullPrompt)
		if err != nil {
			retries--
			lastErr = err
			continue
		}

		if p.debug {
			log.Printf("AI Response: %s", resp.AIResponse)
		}

		cleaned, cleanErr := cleanJSON(resp.AIResponse)
		if cleanErr != nil {
			lastErr = fmt.Errorf("clean error: %w", cleanErr)
			fullPrompt = fmt.Sprintf("Fix this JSON (error: %v):\n%s", cleanErr, resp.AIResponse)
			retries--
			continue
		}

		if err = json.Unmarshal([]byte(cleaned), output); err == nil {
			return time.Since(start), resp.Tokens, nil
		}

		lastErr = fmt.Errorf("parse error: %w", err)
		fullPrompt = fmt.Sprintf("Fix this JSON (error: %v):\n%s", err, resp.AIResponse)
		retries--
	}

	return time.Since(start), 0, lastErr
}

func (p *Parser) ParseChat(messages []models.AIMessage, output any) error {
	t := reflect.TypeOf(output)
	if t.Kind() != reflect.Ptr || t.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("output must be pointer to struct")
	}

	schema := buildSchema(t.Elem(), 0)
	lastIdx := -1
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == models.User {
			lastIdx = i
			break
		}
	}
	if lastIdx == -1 {
		return fmt.Errorf("no user message found")
	}

	messages[lastIdx].Message = fmt.Sprintf("%s\n\nRespond with valid JSON:\n%s",
		messages[lastIdx].Message, schema)

	var lastErr error
	for range p.maxRetries {
		resp, err := p.client.ChatCompletion(models.AIChatHistory{Messages: messages})
		if err != nil {
			lastErr = err
			continue
		}

		cleaned, cleanErr := cleanJSON(resp.AIResponse)
		if cleanErr != nil {
			lastErr = fmt.Errorf("clean error: %w", cleanErr)
			messages = append(messages, models.AIMessage{
				Role:    models.User,
				Message: fmt.Sprintf("Could not extract JSON (error: %v). Reply with valid JSON only, no explanation.", cleanErr),
			})
			continue
		}

		if err = json.Unmarshal([]byte(cleaned), output); err == nil {
			return nil
		}

		lastErr = fmt.Errorf("parse error: %w", err)
		messages = append(messages, models.AIMessage{
			Role:    models.User,
			Message: fmt.Sprintf("Invalid JSON (error: %v). Retry with valid JSON only.", err),
		})
	}
	return lastErr
}

func buildPrompt(t reflect.Type, prompt, context string) string {
	var sb strings.Builder
	if context != "" {
		sb.WriteString("CONTEXT:\n")
		sb.WriteString(context)
		sb.WriteString("\n\n")
	}
	sb.WriteString("PROMPT:\n")
	sb.WriteString(prompt)
	sb.WriteString("\n\nJSON SCHEMA:\n")
	sb.WriteString(buildSchema(t, 0))
	sb.WriteString("\n\nReturn ONLY valid JSON. No markdown, no explanation.")
	return sb.String()
}

func buildSchema(t reflect.Type, indent int) string {
	if t.Kind() == reflect.Ptr {
		return buildSchema(t.Elem(), indent)
	}
	if t.Kind() != reflect.Struct {
		return typeDesc(t)
	}

	var sb strings.Builder
	sb.WriteString("{\n")

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if !field.IsExported() {
			continue
		}

		jsonTag := field.Tag.Get("json")
		parts := strings.Split(jsonTag, ",")
		name := parts[0]
		if name == "" || name == "-" {
			name = field.Name
		}

		required := !slices.Contains(parts[1:], "omitempty")
		sb.WriteString(strings.Repeat(" ", indent+2))
		sb.WriteString(fmt.Sprintf(`"%s": `, name))

		ft := field.Type
		if ft.Kind() == reflect.Struct {
			sb.WriteString(buildSchema(ft, indent+2))
		} else if ft.Kind() == reflect.Slice {
			sb.WriteString("[")
			if ft.Elem().Kind() == reflect.Struct {
				sb.WriteString(buildSchema(ft.Elem(), indent+4))
			} else {
				sb.WriteString(typeDesc(ft.Elem()))
			}
			sb.WriteString("]")
		} else {
			sb.WriteString(typeDesc(ft))
		}

		if required {
			sb.WriteString(" (required)")
		}
		if desc := field.Tag.Get("description"); desc != "" {
			sb.WriteString(fmt.Sprintf(" // %s", desc))
		}
		if i < t.NumField()-1 {
			sb.WriteString(",")
		}
		sb.WriteString("\n")
	}
	sb.WriteString(strings.Repeat(" ", indent))
	sb.WriteString("}")
	return sb.String()
}

func typeDesc(t reflect.Type) string {
	switch t.Kind() {
	case reflect.String:
		return "string"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return "number"
	case reflect.Float32, reflect.Float64:
		return "number"
	case reflect.Bool:
		return "boolean"
	case reflect.Ptr:
		return typeDesc(t.Elem())
	default:
		return "any"
	}
}

// cleanJSON robustly extracts and repairs a JSON object or array from an LLM
// response that may contain markdown fences, prose, comments, trailing commas,
// single-quoted strings, or other common model-output artifacts.
//
// It returns an error only when no JSON-like structure can be found at all.
// Repair steps are applied even when extraction succeeds, so the returned
// string is always passed through json.Valid before being returned; if it
// still fails validation after all repairs the raw extracted bytes are
// returned so the caller can surface a meaningful unmarshal error.
func cleanJSON(s string) (string, error) {
	if s == "" {
		return "", fmt.Errorf("empty response")
	}

	// ── Step 1: strip markdown code fences ──────────────────────────────────
	// Handles ```json, ```JSON, ``` on its own, and edge cases like ``` json
	s = stripCodeFences(s)

	// ── Step 2: extract the outermost balanced JSON value ───────────────────
	// This is the most important step: scan for the first { or [ and walk
	// forward tracking depth so we find the *true* closing delimiter even
	// when the model appended prose after the JSON.
	extracted, err := extractBalanced(s)
	if err != nil {
		// No balanced structure found — maybe the model returned something
		// like a bare string or number. Return as-is so the caller gets a
		// meaningful unmarshal error rather than a silent empty string.
		return strings.TrimSpace(s), fmt.Errorf("no JSON object or array found: %w", err)
	}

	// ── Step 3: repair common model-output issues ───────────────────────────
	repaired := repairJSON(extracted)

	// ── Step 4: validate; if still broken return repaired anyway ────────────
	// The caller's json.Unmarshal will produce the exact error location.
	if !json.Valid([]byte(repaired)) {
		// Try one more time with a more aggressive comment stripper in case
		// repairJSON missed something (e.g. inline // after a value).
		aggressive := stripInlineComments(repaired)
		aggressive = removeTrailingCommas(aggressive)
		if json.Valid([]byte(aggressive)) {
			return aggressive, nil
		}
		// Return best effort — caller will get the unmarshal error.
		return repaired, nil
	}

	return repaired, nil
}

// ── Extraction helpers ───────────────────────────────────────────────────────

var codeBlockRe = regexp.MustCompile("(?si)```[ \t]*(?:json)?[ \t]*\r?\n?(.*?)```")

func stripCodeFences(s string) string {
	// Replace every code block with its inner content.
	// Use the *last* match so that if the model wrapped its answer in a block
	// we get the innermost actual JSON, not outer commentary.
	matches := codeBlockRe.FindAllStringSubmatch(s, -1)
	if len(matches) == 0 {
		// No fences — just strip any stray backtick sequences.
		s = regexp.MustCompile("```(?:json)?").ReplaceAllString(s, "")
		s = strings.ReplaceAll(s, "```", "")
		return strings.TrimSpace(s)
	}

	// Collect all inner contents and join them; often there is only one.
	var parts []string
	for _, m := range matches {
		inner := strings.TrimSpace(m[1])
		if inner != "" {
			parts = append(parts, inner)
		}
	}

	if len(parts) == 0 {
		return strings.TrimSpace(s)
	}

	// Prefer the last block (model sometimes emits explanation then JSON).
	return strings.TrimSpace(parts[len(parts)-1])
}

// extractBalanced scans s for the first occurrence of '{' or '[', then walks
// forward character-by-character — honouring string literals (including escape
// sequences) — to find the matching closing delimiter.  This avoids the
// classic bug of using strings.LastIndex('}') which breaks when the model
// appends prose after the closing brace.
func extractBalanced(s string) (string, error) {
	start := -1
	var open, close rune

	for i, r := range s {
		if r == '{' || r == '[' {
			start = i
			open = r
			if r == '{' {
				close = '}'
			} else {
				close = ']'
			}
			break
		}
	}

	if start == -1 {
		return "", fmt.Errorf("no opening brace or bracket found")
	}

	depth := 0
	inString := false
	escaped := false

	for i := start; i < len(s); {
		r, size := utf8.DecodeRuneInString(s[i:])
		i += size

		if escaped {
			escaped = false
			continue
		}

		if inString {
			switch r {
			case '\\':
				escaped = true
			case '"':
				inString = false
			}
			continue
		}

		switch r {
		case '"':
			inString = true
		case open:
			depth++
		case close:
			depth--
			if depth == 0 {
				return s[start:i], nil
			}
		}
	}

	// Depth never reached zero — the JSON is truncated.  Return what we have
	// and let the repair step try to close it.
	truncated := s[start:]
	return repairTruncated(truncated, open, close), nil
}

// repairTruncated attempts to close an unclosed JSON value by appending the
// missing closing delimiters.  It is intentionally conservative.
func repairTruncated(s string, open, close rune) string {
	depth := 0
	inString := false
	escaped := false

	for _, r := range s {
		if escaped {
			escaped = false
			continue
		}
		if inString {
			if r == '\\' {
				escaped = true
			} else if r == '"' {
				inString = false
			}
			continue
		}
		switch r {
		case '"':
			inString = true
		case open:
			depth++
		case close:
			depth--
		}
	}

	if inString {
		s += `"`
	}
	for i := 0; i < depth; i++ {
		s += string(close)
	}
	return s
}

// ── Repair helpers ───────────────────────────────────────────────────────────

func repairJSON(s string) string {
	// Order matters: do cheaper/safer transforms first.
	s = stripInlineComments(s)    // remove // … and /* … */ comments
	s = removeTrailingCommas(s)   // remove ,} and ,]
	s = normaliseBoolsAndNulls(s) // true/false/null casing
	s = fixSingleQuotes(s)        // 'value' → "value" (last resort)
	return strings.TrimSpace(s)
}

// stripInlineComments removes JavaScript-style comments that LLMs sometimes
// emit inside JSON.  It is string-aware so it won't accidentally strip URLs.
func stripInlineComments(s string) string {
	var out strings.Builder
	out.Grow(len(s))
	i := 0
	inString := false
	escaped := false

	for i < len(s) {
		r, size := utf8.DecodeRuneInString(s[i:])

		if escaped {
			escaped = false
			out.WriteRune(r)
			i += size
			continue
		}

		if inString {
			if r == '\\' {
				escaped = true
			} else if r == '"' {
				inString = false
			}
			out.WriteRune(r)
			i += size
			continue
		}

		// Not in a string — check for comment starters.
		if r == '/' && i+1 < len(s) {
			next := s[i+1]
			if next == '/' {
				// Skip to end of line.
				for i < len(s) && s[i] != '\n' {
					i++
				}
				continue
			}
			if next == '*' {
				// Skip to closing */.
				i += 2
				for i+1 < len(s) {
					if s[i] == '*' && s[i+1] == '/' {
						i += 2
						break
					}
					i++
				}
				continue
			}
		}

		if r == '"' {
			inString = true
		}
		out.WriteRune(r)
		i += size
	}
	return out.String()
}

// removeTrailingCommas removes commas immediately before } or ] which are
// invalid in JSON but common in model output.
var trailingCommaRe = regexp.MustCompile(`,\s*([}\]])`)

func removeTrailingCommas(s string) string {
	// Loop because there can be nested trailing commas.
	for {
		next := trailingCommaRe.ReplaceAllString(s, "$1")
		if next == s {
			break
		}
		s = next
	}
	return s
}

// normaliseBoolsAndNulls fixes capitalisation variants that some models emit
// (True, False, None, NULL, …) while staying string-aware.
func normaliseBoolsAndNulls(s string) string {
	// Only replace outside of quoted strings using a simple word-boundary
	// regex; this is fast and good enough for the common cases.
	replacements := []struct{ bad, good string }{
		{"True", "true"},
		{"False", "false"},
		{"None", "null"},
		{"NULL", "null"},
		{"Null", "null"},
		{"TRUE", "true"},
		{"FALSE", "false"},
		{"undefined", "null"},
	}

	for _, r := range replacements {
		// Use word boundaries so we don't mangle string values that happen
		// to contain these words.
		re := regexp.MustCompile(`(?i)\b` + regexp.QuoteMeta(r.bad) + `\b`)
		s = replaceOutsideStrings(s, re, r.good)
	}
	return s
}

// replaceOutsideStrings applies re.ReplaceAllString only to the portions of s
// that are not inside double-quoted JSON strings.
func replaceOutsideStrings(s string, re *regexp.Regexp, repl string) string {
	var out strings.Builder
	i := 0
	inString := false
	escaped := false

	segStart := 0 // start of current outside-string segment

	flush := func(end int) {
		if segStart < end {
			chunk := s[segStart:end]
			out.WriteString(re.ReplaceAllString(chunk, repl))
		}
	}

	for i < len(s) {
		r, size := utf8.DecodeRuneInString(s[i:])

		if escaped {
			escaped = false
			i += size
			continue
		}

		if inString {
			if r == '\\' {
				escaped = true
			} else if r == '"' {
				inString = false
				segStart = i + size // resume outside segment after closing quote
			}
			i += size
			continue
		}

		if r == '"' {
			// Flush the outside-string segment accumulated so far.
			flush(i)
			inString = true
			// Write the opening quote verbatim.
			out.WriteRune(r)
			i += size
			// Write everything until closing quote verbatim.
			for i < len(s) {
				sr, ss := utf8.DecodeRuneInString(s[i:])
				out.WriteRune(sr)
				i += ss
				if escaped {
					escaped = false
					continue
				}
				if sr == '\\' {
					escaped = true
					continue
				}
				if sr == '"' {
					break
				}
			}
			inString = false
			segStart = i
			continue
		}

		i += size
	}

	// Flush remaining outside-string segment.
	flush(len(s))
	return out.String()
}

// fixSingleQuotes converts single-quoted keys and string values to
// double-quoted ones.  This is a last-resort transform applied only when the
// input looks like it uses single quotes pervasively.
func fixSingleQuotes(s string) string {
	// Only attempt conversion if the string contains single quotes and does
	// not already look like valid JSON (avoid mangling apostrophes in values).
	if !strings.Contains(s, "'") {
		return s
	}

	// Quick heuristic: if it already parses, don't touch it.
	if json.Valid([]byte(s)) {
		return s
	}

	var out strings.Builder
	out.Grow(len(s))
	i := 0
	inDouble := false
	inSingle := false
	escaped := false

	for i < len(s) {
		r, size := utf8.DecodeRuneInString(s[i:])
		i += size

		if escaped {
			escaped = false
			// Inside a now-double-quoted string, pass through the char.
			out.WriteRune(r)
			continue
		}

		if inDouble {
			if r == '\\' {
				escaped = true
			} else if r == '"' {
				inDouble = false
			}
			out.WriteRune(r)
			continue
		}

		if inSingle {
			if r == '\\' {
				escaped = true
				out.WriteRune(r)
				continue
			}
			if r == '\'' {
				inSingle = false
				out.WriteRune('"') // close with double quote
				continue
			}
			// Escape any bare double quotes that appear inside a single-quoted string.
			if r == '"' {
				out.WriteString(`\"`)
				continue
			}
			out.WriteRune(r)
			continue
		}

		// Not in any string.
		if r == '"' {
			inDouble = true
			out.WriteRune(r)
			continue
		}
		if r == '\'' {
			// Only treat as a string delimiter if preceded by :, ,, [, or {
			// (possibly with whitespace).  This avoids mangling contractions.
			prev := strings.TrimRightFunc(out.String(), unicode.IsSpace)
			last := rune(0)
			if len(prev) > 0 {
				last, _ = utf8.DecodeLastRuneInString(prev)
			}
			if last == ':' || last == ',' || last == '[' || last == '{' || last == 0 {
				inSingle = true
				out.WriteRune('"') // open with double quote
				continue
			}
		}
		out.WriteRune(r)
	}

	result := out.String()
	// Only keep the conversion if it actually produces valid JSON; otherwise
	// the original (which the caller will unmarshal and get a proper error on)
	// is safer.
	if json.Valid([]byte(result)) {
		return result
	}
	return s
}
