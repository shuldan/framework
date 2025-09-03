package http

import (
	"bufio"
	"net"
	"net/http/httptest"
	"testing"

	"github.com/shuldan/framework/pkg/contracts"
)

func TestWebsocketContext(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest("GET", "/ws", nil)
	req.Header.Set("Connection", "upgrade")
	req.Header.Set("Upgrade", "websocket")
	req.Header.Set("Sec-WebSocket-Key", "dGhlIHNhbXBsZSBub25jZQ==")
	req.Header.Set("Sec-WebSocket-Version", "13")

	logger := &mockLogger{}
	w := &mockHijackableResponseWriter{ResponseRecorder: httptest.NewRecorder()}

	ctx := NewHTTPContext(w, req, logger)
	ws := ctx.Websocket()

	if !ws.IsWebsocket() {
		t.Error("Expected IsWebsocket to return true")
	}

	if protocols := ws.Subprotocols(); len(protocols) != 0 {
		t.Errorf("Expected no subprotocols, got %v", protocols)
	}
}

func TestWebsocketUpgrade(t *testing.T) {
	t.Parallel()

	t.Run("Successful upgrade", func(t *testing.T) {
		testSuccessfulWebsocketUpgrade(t)
	})

	t.Run("Invalid upgrade", func(t *testing.T) {
		testInvalidWebsocketUpgrade(t)
	})
}

func testSuccessfulWebsocketUpgrade(t *testing.T) {
	req := httptest.NewRequest("GET", "/ws", nil)
	req.Header.Set("Connection", "upgrade")
	req.Header.Set("Upgrade", "websocket")
	req.Header.Set("Sec-WebSocket-Key", "dGhlIHNhbXBsZSBub25jZQ==")
	req.Header.Set("Sec-WebSocket-Version", "13")
	req.Header.Set("Sec-WebSocket-Protocol", "chat, echo")

	logger := &mockLogger{}
	conn, clientConn := net.Pipe()
	defer closeConn(conn)
	defer closeConn(clientConn)

	w := &mockHijackableResponseWriter{
		ResponseRecorder: httptest.NewRecorder(),
		conn:             conn,
		bufrw:            bufio.NewReadWriter(bufio.NewReader(clientConn), bufio.NewWriter(clientConn)),
	}

	ctx := NewHTTPContext(w, req, logger)
	ws := ctx.Websocket()
	wsConn, err := ws.Upgrade()

	if err != nil {
		t.Fatalf("Websocket upgrade failed: %v", err)
	}
	defer closeWs(wsConn)

	if wsConn.IsClosed() {
		t.Error("Expected websocket not to be closed initially")
	}
}

func testInvalidWebsocketUpgrade(t *testing.T) {
	tests := []struct {
		name    string
		headers map[string]string
		wantErr bool
	}{
		{
			name: "Missing Connection header",
			headers: map[string]string{
				"Upgrade":               "websocket",
				"Sec-WebSocket-Key":     "dGhlIHNhbXBsZSBub25jZQ==",
				"Sec-WebSocket-Version": "13",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/ws", nil)
			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}

			logger := &mockLogger{}
			w := &mockHijackableResponseWriter{ResponseRecorder: httptest.NewRecorder()}
			ctx := NewHTTPContext(w, req, logger)
			ws := ctx.Websocket()
			_, err := ws.Upgrade()

			if (err != nil) != tt.wantErr {
				t.Errorf("Expected error=%v, got %v", tt.wantErr, err)
			}
		})
	}
}

func closeConn(c net.Conn) {
	_ = c.Close()
}

func closeWs(ws contracts.HTTPWebsocketConnection) {
	_ = ws.Close()
}

func TestWebsocketInvalidUpgrade(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		headers map[string]string
		wantErr bool
	}{
		{
			name: "Missing Connection header",
			headers: map[string]string{
				"Upgrade":               "websocket",
				"Sec-WebSocket-Key":     "dGhlIHNhbXBsZSBub25jZQ==",
				"Sec-WebSocket-Version": "13",
			},
			wantErr: true,
		},
		{
			name: "Missing WebSocket key",
			headers: map[string]string{
				"Connection":            "upgrade",
				"Upgrade":               "websocket",
				"Sec-WebSocket-Version": "13",
			},
			wantErr: true,
		},
		{
			name: "Invalid WebSocket version",
			headers: map[string]string{
				"Connection":            "upgrade",
				"Upgrade":               "websocket",
				"Sec-WebSocket-Key":     "dGhlIHNhbXBsZSBub25jZQ==",
				"Sec-WebSocket-Version": "12",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/ws", nil)
			for key, value := range tt.headers {
				req.Header.Set(key, value)
			}

			logger := &mockLogger{}
			w := &mockHijackableResponseWriter{ResponseRecorder: httptest.NewRecorder()}

			ctx := NewHTTPContext(w, req, logger)
			ws := ctx.Websocket()

			_, err := ws.Upgrade()
			if tt.wantErr && err == nil {
				t.Error("Expected error for invalid upgrade")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}
