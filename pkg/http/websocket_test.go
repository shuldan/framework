package http

import (
	"bufio"
	"context"
	"errors"
	"net"
	"net/http/httptest"
	"testing"
	"time"

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

func TestWebsocketContextSubprotocols(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest("GET", "/ws", nil)
	req.Header.Set("Sec-WebSocket-Protocol", "chat, echo, binary")
	logger := &mockLogger{}
	w := httptest.NewRecorder()

	ctx := NewHTTPContext(w, req, logger)
	ws := ctx.Websocket()

	protocols := ws.Subprotocols()
	expected := []string{"chat", "echo", "binary"}

	if len(protocols) != len(expected) {
		t.Errorf("Expected %d protocols, got %d", len(expected), len(protocols))
	}

	for i, protocol := range protocols {
		if protocol != expected[i] {
			t.Errorf("Expected protocol %s, got %s", expected[i], protocol)
		}
	}
}

func TestWebsocketContextOrigin(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest("GET", "/ws", nil)
	req.Header.Set("Origin", "https://example.com")
	logger := &mockLogger{}
	w := httptest.NewRecorder()

	ctx := NewHTTPContext(w, req, logger)
	ws := ctx.Websocket()

	if origin := ws.Origin(); origin != "https://example.com" {
		t.Errorf("Expected origin https://example.com, got %s", origin)
	}
}

func TestWebsocketUpgradeProtocolSelection(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest("GET", "/ws", nil)
	req.Header.Set("Connection", "upgrade")
	req.Header.Set("Upgrade", "websocket")
	req.Header.Set("Sec-WebSocket-Key", "dGhlIHNhbXBsZSBub25jZQ==")
	req.Header.Set("Sec-WebSocket-Version", "13")
	req.Header.Set("Sec-WebSocket-Protocol", "chat, unsupported")

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

	selectedProtocol := w.Header().Get("Sec-WebSocket-Protocol")
	if selectedProtocol != "chat" {
		t.Errorf("Expected selected protocol 'chat', got '%s'", selectedProtocol)
	}
}

func TestWebsocketConnectionWrite(t *testing.T) {
	t.Parallel()

	conn, clientConn := net.Pipe()
	defer closeConn(conn)

	logger := &mockLogger{}
	bufrw := bufio.NewReadWriter(bufio.NewReader(clientConn), bufio.NewWriter(clientConn))
	wsConn := NewWebsocketConnection(conn, bufrw, logger)
	defer closeWs(wsConn)

	go func() {
		buf := make([]byte, 4096)
		for {
			_, err := conn.Read(buf)
			if err != nil {
				return
			}
		}
	}()

	ctx := context.Background()
	textMsg := contracts.HTTPWebsocketMessage{
		Type: contracts.WebsocketText,
		Data: []byte("Hello, WebSocket!"),
	}
	if err := wsConn.Write(ctx, textMsg); err != nil {
		t.Fatalf("Write text message failed: %v", err)
	}

	binaryMsg := contracts.HTTPWebsocketMessage{
		Type: contracts.WebsocketBinary,
		Data: []byte{0x01, 0x02, 0x03, 0x04},
	}
	if err := wsConn.Write(ctx, binaryMsg); err != nil {
		t.Fatalf("Write binary message failed: %v", err)
	}

	_ = clientConn.Close()
}

func TestWebsocketConnectionWriteInvalidType(t *testing.T) {
	t.Parallel()

	conn, clientConn := net.Pipe()
	defer closeConn(conn)

	logger := &mockLogger{}
	bufrw := bufio.NewReadWriter(bufio.NewReader(clientConn), bufio.NewWriter(clientConn))
	wsConn := NewWebsocketConnection(conn, bufrw, logger)
	defer closeWs(wsConn)

	go func() {
		buf := make([]byte, 4096)
		for {
			_, err := conn.Read(buf)
			if err != nil {
				return
			}
		}
	}()

	ctx := context.Background()

	invalidMsg := contracts.HTTPWebsocketMessage{
		Type: 999,
		Data: []byte("invalid"),
	}

	if err := wsConn.Write(ctx, invalidMsg); err == nil {
		t.Error("Expected error for invalid message type")
	}

	_ = clientConn.Close()
}

func TestWebsocketConnectionPing(t *testing.T) {
	t.Parallel()

	conn, clientConn := net.Pipe()
	defer closeConn(conn)

	logger := &mockLogger{}
	bufrw := bufio.NewReadWriter(bufio.NewReader(clientConn), bufio.NewWriter(clientConn))
	wsConn := NewWebsocketConnection(conn, bufrw, logger)
	defer closeWs(wsConn)

	go func() {
		buf := make([]byte, 4096)
		for {
			_, err := conn.Read(buf)
			if err != nil {
				return
			}
		}
	}()

	ctx := context.Background()

	if err := wsConn.Ping(ctx); err != nil {
		t.Fatalf("Ping failed: %v", err)
	}

	_ = clientConn.Close()
}

func TestWebsocketConnectionWriteAfterClose(t *testing.T) {
	t.Parallel()

	conn, clientConn := net.Pipe()
	defer closeConn(conn)

	logger := &mockLogger{}
	bufrw := bufio.NewReadWriter(bufio.NewReader(clientConn), bufio.NewWriter(clientConn))
	wsConn := NewWebsocketConnection(conn, bufrw, logger)
	defer closeWs(wsConn)

	go func() {
		buf := make([]byte, 4096)
		for {
			_, err := conn.Read(buf)
			if err != nil {
				return
			}
		}
	}()

	_ = clientConn.Close()
	if err := wsConn.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	ctx := context.Background()
	msg := contracts.HTTPWebsocketMessage{
		Type: contracts.WebsocketText,
		Data: []byte("test"),
	}

	if err := wsConn.Write(ctx, msg); !errors.Is(err, ErrWebsocketClosed) {
		t.Errorf("Expected ErrWebsocketClosed, got %v", err)
	}

	if err := wsConn.Ping(ctx); !errors.Is(err, ErrWebsocketClosed) {
		t.Errorf("Expected ErrWebsocketClosed for ping, got %v", err)
	}
}

func TestWebsocketConnectionLargePayload(t *testing.T) {
	t.Parallel()

	conn, clientConn := net.Pipe()
	defer closeConn(conn)

	logger := &mockLogger{}
	bufrw := bufio.NewReadWriter(bufio.NewReader(clientConn), bufio.NewWriter(clientConn))
	wsConn := NewWebsocketConnection(conn, bufrw, logger)
	defer closeWs(wsConn)

	go func() {
		buf := make([]byte, 4096)
		for {
			_, err := conn.Read(buf)
			if err != nil {
				return
			}
		}
	}()

	ctx := context.Background()

	largeData := make([]byte, 70000)
	for i := range largeData {
		largeData[i] = byte(i % 256)
	}

	largeMsg := contracts.HTTPWebsocketMessage{
		Type: contracts.WebsocketBinary,
		Data: largeData,
	}

	if err := wsConn.Write(ctx, largeMsg); err != nil {
		t.Fatalf("Write large message failed: %v", err)
	}
	_ = clientConn.Close()
}

func TestWebsocketConnectionDoubleClose(t *testing.T) {
	t.Parallel()

	conn, clientConn := net.Pipe()
	defer closeConn(conn)

	logger := &mockLogger{}
	bufrw := bufio.NewReadWriter(bufio.NewReader(clientConn), bufio.NewWriter(clientConn))
	wsConn := NewWebsocketConnection(conn, bufrw, logger)
	defer closeWs(wsConn)

	go func() {
		buf := make([]byte, 4096)
		for {
			_, err := conn.Read(buf)
			if err != nil {
				return
			}
		}
	}()

	if err := wsConn.Close(); err != nil {
		t.Fatalf("First close failed: %v", err)
	}

	if err := wsConn.Close(); err != nil {
		t.Errorf("Second close should not error: %v", err)
	}
	_ = clientConn.Close()
}

func TestHSTSMiddlewareImport(t *testing.T) {
	t.Parallel()
	middleware := HSTSMiddleware(time.Hour, false)
	if middleware == nil {
		t.Error("HSTSMiddleware should not return nil")
	}
}
