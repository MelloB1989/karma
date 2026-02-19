package voice

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/gorilla/websocket"
)

const (
	defaultWSReadTimeout  = 60 * time.Second
	defaultWSWriteTimeout = 30 * time.Second
)

// WSMessage holds an incoming websocket message.
type WSMessage struct {
	Type int
	Data []byte
	JSON map[string]any
}

// WSMessageHandler handles incoming websocket messages.
type WSMessageHandler func(ctx context.Context, message WSMessage) error

// WSCloseHandler handles websocket close frames.
type WSCloseHandler func(code int, text string) error

// WebSocketHandler provides a reusable read/write utility around a websocket connection.
type WebSocketHandler struct {
	conn         *websocket.Conn
	readTimeout  time.Duration
	writeTimeout time.Duration
	onMessage    WSMessageHandler
	onClose      WSCloseHandler
}

// WebSocketHandlerOption configures a websocket handler.
type WebSocketHandlerOption func(*WebSocketHandler)

// NewWebSocketHandler creates a new websocket helper for a connection.
func NewWebSocketHandler(conn *websocket.Conn, options ...WebSocketHandlerOption) (*WebSocketHandler, error) {
	if conn == nil {
		return nil, errors.New("websocket connection is nil")
	}

	handler := &WebSocketHandler{
		conn:         conn,
		readTimeout:  defaultWSReadTimeout,
		writeTimeout: defaultWSWriteTimeout,
	}

	for _, option := range options {
		if option != nil {
			option(handler)
		}
	}

	return handler, nil
}

// WithWSReadTimeout sets read deadline timeout.
func WithWSReadTimeout(timeout time.Duration) WebSocketHandlerOption {
	return func(handler *WebSocketHandler) {
		if timeout > 0 {
			handler.readTimeout = timeout
		}
	}
}

// WithWSWriteTimeout sets write deadline timeout.
func WithWSWriteTimeout(timeout time.Duration) WebSocketHandlerOption {
	return func(handler *WebSocketHandler) {
		if timeout > 0 {
			handler.writeTimeout = timeout
		}
	}
}

// WithWSMessageHandler sets callback for every message received by Run.
func WithWSMessageHandler(messageHandler WSMessageHandler) WebSocketHandlerOption {
	return func(handler *WebSocketHandler) {
		handler.onMessage = messageHandler
	}
}

// WithWSCloseHandler sets callback for websocket close frames.
func WithWSCloseHandler(closeHandler WSCloseHandler) WebSocketHandlerOption {
	return func(handler *WebSocketHandler) {
		handler.onClose = closeHandler
	}
}

// Run starts a read loop until context cancellation, close frame, or error.
func (h *WebSocketHandler) Run(ctx context.Context) error {
	if h == nil || h.conn == nil {
		return errors.New("websocket handler is not initialized")
	}

	for {
		if ctx != nil {
			if err := ctx.Err(); err != nil {
				return err
			}
		}

		if err := h.setReadDeadline(ctx); err != nil {
			return err
		}

		messageType, data, err := h.conn.ReadMessage()
		if err != nil {
			var closeErr *websocket.CloseError
			if errors.As(err, &closeErr) {
				if h.onClose != nil {
					if closeHandlerErr := h.onClose(closeErr.Code, closeErr.Text); closeHandlerErr != nil {
						return closeHandlerErr
					}
				}
				return nil
			}

			if ctx != nil {
				if ctxErr := ctx.Err(); ctxErr != nil {
					return ctxErr
				}
			}
			return fmt.Errorf("websocket read failed: %w", err)
		}

		if h.onMessage == nil {
			continue
		}

		message := WSMessage{
			Type: messageType,
			Data: data,
		}

		if messageType == websocket.TextMessage {
			var payload map[string]any
			if err := json.Unmarshal(data, &payload); err == nil {
				message.JSON = payload
			}
		}

		if err := h.onMessage(ctx, message); err != nil {
			return err
		}
	}
}

// SendJSON writes JSON payload to the websocket connection.
func (h *WebSocketHandler) SendJSON(ctx context.Context, payload any) error {
	if h == nil || h.conn == nil {
		return errors.New("websocket handler is not initialized")
	}
	if err := h.setWriteDeadline(ctx); err != nil {
		return err
	}
	if err := h.conn.WriteJSON(payload); err != nil {
		return fmt.Errorf("websocket write json failed: %w", err)
	}
	return nil
}

// SendText writes text payload to the websocket connection.
func (h *WebSocketHandler) SendText(ctx context.Context, text string) error {
	return h.sendMessage(ctx, websocket.TextMessage, []byte(text))
}

// SendBinary writes binary payload to the websocket connection.
func (h *WebSocketHandler) SendBinary(ctx context.Context, data []byte) error {
	return h.sendMessage(ctx, websocket.BinaryMessage, data)
}

func (h *WebSocketHandler) sendMessage(ctx context.Context, messageType int, payload []byte) error {
	if h == nil || h.conn == nil {
		return errors.New("websocket handler is not initialized")
	}
	if err := h.setWriteDeadline(ctx); err != nil {
		return err
	}
	if err := h.conn.WriteMessage(messageType, payload); err != nil {
		return fmt.Errorf("websocket write failed: %w", err)
	}
	return nil
}

// Close closes the websocket connection.
func (h *WebSocketHandler) Close() error {
	if h == nil || h.conn == nil {
		return nil
	}
	return h.conn.Close()
}

func (h *WebSocketHandler) setReadDeadline(ctx context.Context) error {
	deadline := time.Now().Add(h.readTimeout)
	if ctx != nil {
		if ctxDeadline, ok := ctx.Deadline(); ok && ctxDeadline.Before(deadline) {
			deadline = ctxDeadline
		}
	}
	if err := h.conn.SetReadDeadline(deadline); err != nil {
		return fmt.Errorf("failed to set websocket read deadline: %w", err)
	}
	return nil
}

func (h *WebSocketHandler) setWriteDeadline(ctx context.Context) error {
	deadline := time.Now().Add(h.writeTimeout)
	if ctx != nil {
		if ctxDeadline, ok := ctx.Deadline(); ok && ctxDeadline.Before(deadline) {
			deadline = ctxDeadline
		}
	}
	if err := h.conn.SetWriteDeadline(deadline); err != nil {
		return fmt.Errorf("failed to set websocket write deadline: %w", err)
	}
	return nil
}
