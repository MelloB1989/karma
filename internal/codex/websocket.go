package codex

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

// wsCreateMessage is the single frame sent to open a Responses turn over the
// WebSocket transport: a "response.create" type plus the (inlined) request
// fields. Matches codex-proxy's WsCreateRequest.
type wsCreateMessage struct {
	Type string `json:"type"` // "response.create"
	*ResponsesRequest
}

// generateWS runs the request over a WebSocket to /codex/responses and consumes
// the streamed events. req must already be prepared (see prepareRequest).
func (c *Client) generateWS(ctx context.Context, req *ResponsesRequest, onText, onReasoning func(string) error) (*Result, error) {
	conn, err := c.dialWS(ctx)
	if err != nil {
		return nil, fmt.Errorf("codex: websocket dial: %w", err)
	}
	defer conn.Close()

	// Close the connection when ctx is done so a blocked ReadMessage unblocks.
	stop := make(chan struct{})
	defer close(stop)
	go func() {
		select {
		case <-ctx.Done():
			conn.Close()
		case <-stop:
		}
	}()

	if err := conn.WriteJSON(wsCreateMessage{Type: "response.create", ResponsesRequest: req}); err != nil {
		return nil, fmt.Errorf("codex: websocket send: %w", err)
	}

	// Keepalive pings — the only writer after the initial send, so this stays
	// within gorilla's one-reader/one-writer concurrency contract.
	go func() {
		t := time.NewTicker(25 * time.Second)
		defer t.Stop()
		for {
			select {
			case <-stop:
				return
			case <-t.C:
				_ = conn.WriteControl(websocket.PingMessage, nil, time.Now().Add(10*time.Second))
			}
		}
	}()

	return processEvents(onText, onReasoning, func(fn func(SSEEvent) error) error {
		return pumpWS(conn, fn)
	})
}

// dialWS opens the authenticated WebSocket to the Codex Responses endpoint.
func (c *Client) dialWS(ctx context.Context) (*websocket.Conn, error) {
	token, accountID, err := c.tokens.Token(ctx)
	if err != nil {
		return nil, err
	}
	header := http.Header{}
	c.applyHeaders(header, token, accountID, getInstallationID(), "", false)

	dialer := websocket.Dialer{
		HandshakeTimeout: 20 * time.Second,
		Jar:              c.httpClient.Jar,
	}
	conn, resp, err := dialer.DialContext(ctx, toWSURL(c.cfg.BaseURL)+"/codex/responses", header)
	if err != nil {
		if resp != nil && resp.Body != nil {
			resp.Body.Close()
		}
		return nil, err
	}
	return conn, nil
}

// pumpWS reads WebSocket frames and feeds each as an SSEEvent to fn until a
// terminal event or a read error. Codex rate-limit frames are skipped.
func pumpWS(conn *websocket.Conn, fn func(SSEEvent) error) error {
	for {
		_, data, err := conn.ReadMessage()
		if err != nil {
			return err
		}
		typ := wsEventType(data)
		if typ == "codex.rate_limits" {
			continue
		}
		if err := fn(SSEEvent{Event: typ, Data: json.RawMessage(data)}); err != nil {
			return err
		}
		switch typ {
		case "response.completed", "response.failed", "error":
			return nil
		}
	}
}

func wsEventType(data []byte) string {
	var head struct {
		Type string `json:"type"`
	}
	_ = json.Unmarshal(data, &head)
	return head.Type
}

// toWSURL converts an http(s) base URL to its ws(s) equivalent.
func toWSURL(base string) string {
	switch {
	case strings.HasPrefix(base, "https://"):
		return "wss://" + strings.TrimPrefix(base, "https://")
	case strings.HasPrefix(base, "http://"):
		return "ws://" + strings.TrimPrefix(base, "http://")
	default:
		return base
	}
}
