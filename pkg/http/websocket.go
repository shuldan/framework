package http

import (
	"bufio"
	"context"
	"crypto/sha1" // #nosec G505
	"encoding/base64"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/shuldan/framework/pkg/contracts"
)

const websocketMagicString = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"

type httpWebsocketContext struct {
	ctx    *httpContext
	logger contracts.Logger
}

func (w *httpWebsocketContext) IsWebsocket() bool {
	return strings.ToLower(w.ctx.RequestHeader("Connection")) == "upgrade" &&
		strings.ToLower(w.ctx.RequestHeader("Upgrade")) == "websocket"
}

func (w *httpWebsocketContext) Origin() string {
	return w.ctx.RequestHeader("Origin")
}

func (w *httpWebsocketContext) Subprotocols() []string {
	protocols := w.ctx.RequestHeader("Sec-WebSocket-Protocol")
	if protocols == "" {
		return []string{}
	}
	parts := strings.Split(protocols, ",")
	result := make([]string, len(parts))
	for i, part := range parts {
		result[i] = strings.TrimSpace(part)
	}

	return result
}

func (w *httpWebsocketContext) Upgrade() (contracts.HTTPWebsocketConnection, error) {
	if !w.IsWebsocket() {
		return nil, ErrWebsocketUpgrade.WithDetail("reason", "not a websocket request")
	}
	key := w.ctx.RequestHeader("Sec-WebSocket-Key")
	if key == "" {
		return nil, ErrWebsocketUpgrade.WithDetail("reason", "missing Sec-WebSocket-Key")
	}
	version := w.ctx.RequestHeader("Sec-WebSocket-Version")
	if version != "13" {
		return nil, ErrWebsocketUpgrade.WithDetail("reason", "unsupported WebSocket version").WithDetail("version", version)
	}
	origin := w.Origin()
	if origin != "" && !w.isOriginAllowed(origin) {
		return nil, ErrWebsocketUpgrade.WithDetail("reason", "origin not allowed").WithDetail("origin", origin)
	}
	h := sha1.New() // #nosec G401
	h.Write([]byte(key + websocketMagicString))
	accept := base64.StdEncoding.EncodeToString(h.Sum(nil))
	hijacker, ok := w.ctx.resp.(http.Hijacker)
	if !ok {
		return nil, ErrWebsocketUpgrade.WithDetail("reason", "response writer doesn't support hijacking")
	}
	selectedProtocol := w.selectSubprotocol()
	w.ctx.resp.Header().Set("Upgrade", "websocket")
	w.ctx.resp.Header().Set("Connection", "Upgrade")
	w.ctx.resp.Header().Set("Sec-WebSocket-Accept", accept)
	if selectedProtocol != "" {
		w.ctx.resp.Header().Set("Sec-WebSocket-Protocol", selectedProtocol)
	}
	w.ctx.resp.WriteHeader(http.StatusSwitchingProtocols)
	conn, bufrw, err := hijacker.Hijack()
	if err != nil {
		return nil, ErrWebsocketUpgrade.WithCause(err)
	}
	if tcpConn, ok := conn.(*net.TCPConn); ok {
		if err := tcpConn.SetKeepAlive(true); err != nil && w.logger != nil {
			w.logger.Warn("Failed to set keep alive", "error", err)
		}
		if err := tcpConn.SetKeepAlivePeriod(30 * time.Second); err != nil && w.logger != nil {
			w.logger.Warn("Failed to set keep alive period", "error", err)
		}
	}
	return NewWebsocketConnection(conn, bufrw, w.logger), nil
}

func (w *httpWebsocketContext) isOriginAllowed(origin string) bool {
	return true
}

func (w *httpWebsocketContext) selectSubprotocol() string {
	clientProtocols := w.Subprotocols()
	if len(clientProtocols) == 0 {
		return ""
	}
	supportedProtocols := []string{"chat", "echo"}
	for _, clientProto := range clientProtocols {
		for _, supportedProto := range supportedProtocols {
			if clientProto == supportedProto {
				return clientProto
			}
		}
	}
	return ""
}

type WebsocketConnection struct {
	conn   net.Conn
	reader *bufio.Reader
	writer *bufio.Writer
	mu     sync.RWMutex

	closed bool

	readChan chan contracts.HTTPWebsocketMessage
	done     chan struct{}
	logger   contracts.Logger
	wg       sync.WaitGroup
}

func NewWebsocketConnection(conn net.Conn, bufrw *bufio.ReadWriter, logger contracts.Logger) contracts.HTTPWebsocketConnection {
	ws := &WebsocketConnection{
		conn:     conn,
		reader:   bufrw.Reader,
		writer:   bufrw.Writer,
		readChan: make(chan contracts.HTTPWebsocketMessage, 10),
		done:     make(chan struct{}),
		logger:   logger,
	}

	ws.wg.Add(1)
	go ws.readLoop()

	return ws
}

func (w *WebsocketConnection) Read() <-chan contracts.HTTPWebsocketMessage {
	return w.readChan
}

func (w *WebsocketConnection) Write(_ context.Context, msg contracts.HTTPWebsocketMessage) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.closed {
		return ErrWebsocketClosed
	}

	var opcode byte
	switch msg.Type {
	case contracts.WebsocketText:
		opcode = 0x81
	case contracts.WebsocketBinary:
		opcode = 0x82
	default:
		return ErrWebsocketUpgrade.WithDetail("reason", "invalid message type")
	}

	dataLen := len(msg.Data)
	var frame []byte

	switch {
	case dataLen < 126:
		frame = make([]byte, 2+dataLen)
		frame[0] = opcode
		frame[1] = byte(dataLen)
		copy(frame[2:], msg.Data)

	case dataLen < 65536:
		frame = make([]byte, 4+dataLen)
		frame[0] = opcode
		frame[1] = 126
		frame[2] = byte(dataLen >> 8)
		frame[3] = byte(dataLen)
		copy(frame[4:], msg.Data)

	default:
		frame = make([]byte, 10+dataLen)
		frame[0] = opcode
		frame[1] = 127
		for i := 0; i < 8; i++ {
			frame[2+i] = byte(dataLen >> (56 - i*8))
		}
		copy(frame[10:], msg.Data)
	}

	_, err := w.writer.Write(frame)
	if err != nil {
		return err
	}

	return w.writer.Flush()
}

func (w *WebsocketConnection) Close() error {
	w.mu.Lock()
	if w.closed {
		w.mu.Unlock()
		return nil
	}
	w.closed = true
	close(w.done)
	w.mu.Unlock()

	w.wg.Wait()

	close(w.readChan)
	return w.conn.Close()
}

func (w *WebsocketConnection) Ping(_ context.Context) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.closed {
		return ErrWebsocketClosed
	}

	frame := []byte{0x89, 0x00}
	_, err := w.writer.Write(frame)
	if err != nil {
		return err
	}

	return w.writer.Flush()
}

func (w *WebsocketConnection) IsClosed() bool {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.closed
}

func (w *WebsocketConnection) readLoop() {
	defer w.wg.Done()
	defer func() {
		if r := recover(); r != nil {
			if w.logger != nil {
				w.logger.Error("websocket read panic", "panic", r)
			}
			w.readChan <- contracts.HTTPWebsocketMessage{
				Error: fmt.Errorf("websocket panic: %v", r),
			}
		}
	}()

	for {
		select {
		case <-w.done:
			return
		default:
		}

		if err := w.conn.SetReadDeadline(time.Now().Add(30 * time.Second)); err != nil {
			w.readChan <- contracts.HTTPWebsocketMessage{Error: err}
			return
		}

		frame, err := w.readFrame()
		if err != nil {
			if !w.IsClosed() {
				w.readChan <- contracts.HTTPWebsocketMessage{Error: err}
			}
			return
		}

		if err := w.handleFrame(frame); err != nil {
			if !w.IsClosed() {
				w.readChan <- contracts.HTTPWebsocketMessage{Error: err}
			}
			return
		}
	}
}

func (w *WebsocketConnection) sendPong(data []byte) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.closed {
		return
	}

	frame := make([]byte, 2+len(data))
	frame[0] = 0x8A
	frame[1] = byte(len(data))
	copy(frame[2:], data)

	if _, err := w.writer.Write(frame); err != nil && w.logger != nil {
		w.logger.Error("error sending pong", "error", err)
	}
	if err := w.writer.Flush(); err != nil && w.logger != nil {
		w.logger.Error("error sending pong", "error", err)
	}
}

type websocketFrame struct {
	fin     bool
	opcode  byte
	payload []byte
}

func (w *WebsocketConnection) readFrame() (*websocketFrame, error) {
	header, err := w.readHeader()
	if err != nil {
		return nil, err
	}

	fin := (header[0] & 0x80) != 0
	opcode := header[0] & 0x0F
	masked := (header[1] & 0x80) != 0

	payloadLen, err := w.readPayloadLength(header)
	if err != nil {
		return nil, err
	}

	var maskKey []byte
	if masked {
		maskKey, err = w.readMaskKey()
		if err != nil {
			return nil, err
		}
	}

	payload, err := w.readAndUnmaskPayload(payloadLen, maskKey)
	if err != nil {
		return nil, err
	}

	return &websocketFrame{
		fin:     fin,
		opcode:  opcode,
		payload: payload,
	}, nil
}

func (w *WebsocketConnection) readHeader() ([]byte, error) {
	header := make([]byte, 2)
	_, err := io.ReadFull(w.reader, header)
	return header, err
}

func (w *WebsocketConnection) readPayloadLength(header []byte) (int, error) {
	payloadLen := int(header[1] & 0x7F)

	switch payloadLen {
	case 126:
		extLen := make([]byte, 2)
		_, err := io.ReadFull(w.reader, extLen)
		if err != nil {
			return 0, err
		}
		return int(extLen[0])<<8 | int(extLen[1]), nil

	case 127:
		extLen := make([]byte, 8)
		_, err := io.ReadFull(w.reader, extLen)
		if err != nil {
			return 0, err
		}
		var length int
		for i := 0; i < 8; i++ {
			length = length<<8 | int(extLen[i])
		}
		return length, nil

	default:
		return payloadLen, nil
	}
}

func (w *WebsocketConnection) readMaskKey() ([]byte, error) {
	maskKey := make([]byte, 4)
	_, err := io.ReadFull(w.reader, maskKey)
	return maskKey, err
}

func (w *WebsocketConnection) readAndUnmaskPayload(payloadLen int, maskKey []byte) ([]byte, error) {
	if payloadLen == 0 {
		return []byte{}, nil
	}

	payload := make([]byte, payloadLen)
	_, err := io.ReadFull(w.reader, payload)
	if err != nil {
		return nil, err
	}

	if maskKey != nil {
		for i := range payload {
			payload[i] ^= maskKey[i%4]
		}
	}
	return payload, nil
}

func (w *WebsocketConnection) handleFrame(frame *websocketFrame) error {
	switch frame.opcode {
	case 0x01:
		if frame.fin {
			w.readChan <- contracts.HTTPWebsocketMessage{
				Type: contracts.WebsocketText,
				Data: frame.payload,
			}
		}
	case 0x02:
		if frame.fin {
			w.readChan <- contracts.HTTPWebsocketMessage{
				Type: contracts.WebsocketBinary,
				Data: frame.payload,
			}
		}
	case 0x08:
		if err := w.Close(); err != nil && w.logger != nil {
			w.logger.Error("error closing websocket connection", "error", err)
		}
		return io.EOF
	case 0x09:
		w.sendPong(frame.payload)
	case 0x0A:
	default:
		return ErrUnsupportedOpcode.WithDetail("code", string(frame.opcode))
	}
	return nil
}
