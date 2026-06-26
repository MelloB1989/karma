package codex

import (
	"bufio"
	"encoding/json"
	"io"
	"strings"
)

// maxSSEBufferBytes caps a single event block (between two blank lines). Large
// enough for a base64 image_generation event with headroom.
const maxSSEBufferBytes = 64 * 1024 * 1024

// parseSSE reads a Codex Responses SSE stream and invokes fn for each parsed
// event. It returns when the stream ends or fn returns an error. A "[DONE]"
// data payload terminates an event (skipped, not delivered).
func parseSSE(r io.Reader, fn func(SSEEvent) error) error {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), maxSSEBufferBytes)
	// Split on SSE record boundary (blank line). We use a custom split that
	// accumulates lines until a blank line, tolerating both \n and \r\n.
	var block []string
	flush := func() error {
		if len(block) == 0 {
			return nil
		}
		evt, ok := parseSSEBlock(block)
		block = block[:0]
		if !ok {
			return nil
		}
		return fn(evt)
	}

	for scanner.Scan() {
		line := strings.TrimSuffix(scanner.Text(), "\r")
		if line == "" {
			if err := flush(); err != nil {
				return err
			}
			continue
		}
		block = append(block, line)
	}
	if err := flush(); err != nil {
		return err
	}
	return scanner.Err()
}

// parseSSEBlock parses a single SSE record (its non-empty lines) into an event.
// Mirrors codex-sse.ts: collects the `event:` name and concatenates `data:`
// lines, tolerating pretty-printed multi-line JSON whose continuation lines
// omit the `data:` prefix.
func parseSSEBlock(lines []string) (SSEEvent, bool) {
	var event string
	var dataLines []string
	dataStarted := false

	for _, line := range lines {
		switch {
		case strings.HasPrefix(line, "event:"):
			event = strings.TrimSpace(line[len("event:"):])
		case strings.HasPrefix(line, "data:"):
			dataStarted = true
			dataLines = append(dataLines, strings.TrimLeft(line[len("data:"):], " "))
		case dataStarted &&
			!strings.HasPrefix(line, "id:") &&
			!strings.HasPrefix(line, "retry:") &&
			!strings.HasPrefix(line, ":"):
			dataLines = append(dataLines, line)
		}
	}

	if event == "" && len(dataLines) == 0 {
		return SSEEvent{}, false
	}
	raw := strings.Join(dataLines, "\n")
	if raw == "[DONE]" {
		return SSEEvent{}, false
	}
	return SSEEvent{Event: event, Data: json.RawMessage(raw)}, true
}
