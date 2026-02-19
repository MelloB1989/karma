package voice

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestWebSocketHandlerRun_ParsesJSONMessages(t *testing.T) {
	upgrader := websocket.Upgrader{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		_ = conn.WriteMessage(websocket.TextMessage, []byte(`{"event":"hello"}`))
		_ = conn.WriteControl(
			websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseNormalClosure, "bye"),
			time.Now().Add(time.Second),
		)
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("failed to dial websocket server: %v", err)
	}
	defer conn.Close()

	var got WSMessage
	handler, err := NewWebSocketHandler(conn, WithWSMessageHandler(func(_ context.Context, message WSMessage) error {
		got = message
		return nil
	}))
	if err != nil {
		t.Fatalf("failed to create websocket handler: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := handler.Run(ctx); err != nil {
		t.Fatalf("run returned unexpected error: %v", err)
	}

	if got.Type != websocket.TextMessage {
		t.Fatalf("expected text message, got type %d", got.Type)
	}
	if string(got.Data) != `{"event":"hello"}` {
		t.Fatalf("unexpected data: %s", string(got.Data))
	}
	if got.JSON["event"] != "hello" {
		t.Fatalf("expected parsed json event=hello, got: %+v", got.JSON)
	}
}

func TestWebSocketHandlerSendJSON(t *testing.T) {
	upgrader := websocket.Upgrader{}
	received := make(chan map[string]any, 1)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		_, data, err := conn.ReadMessage()
		if err != nil {
			return
		}

		payload := map[string]any{}
		if err := json.Unmarshal(data, &payload); err != nil {
			return
		}
		received <- payload
	}))
	defer server.Close()

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("failed to dial websocket server: %v", err)
	}
	defer conn.Close()

	handler, err := NewWebSocketHandler(conn)
	if err != nil {
		t.Fatalf("failed to create websocket handler: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := handler.SendJSON(ctx, map[string]any{"kind": "ping"}); err != nil {
		t.Fatalf("failed to send json: %v", err)
	}

	select {
	case payload := <-received:
		if payload["kind"] != "ping" {
			t.Fatalf("unexpected payload: %+v", payload)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("did not receive payload on server")
	}
}
