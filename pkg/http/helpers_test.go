package http

import (
	"bufio"
	"bytes"
	"fmt"
	"net"
	"net/http/httptest"
	"sync"

	"github.com/shuldan/framework/pkg/contracts"
)

type mockFlushableResponseWriter struct {
	*httptest.ResponseRecorder
	flushed bool
}

func (m *mockFlushableResponseWriter) Flush() {
	m.flushed = true
}

type mockHijackableResponseWriter struct {
	*httptest.ResponseRecorder
	conn  net.Conn
	bufrw *bufio.ReadWriter
}

func (m *mockHijackableResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if m.conn == nil || m.bufrw == nil {
		return nil, nil, fmt.Errorf("hijacking not supported")
	}
	return m.conn, m.bufrw, nil
}

func createMultipartForm(body *bytes.Buffer, fields map[string]string) *multipartWriter {
	writer := &multipartWriter{
		boundary: "----formdata-test-boundary",
		body:     body,
	}

	filename := fields["filename"]
	content := fields["content"]

	writer.writeField("field", fields["field"])
	writer.writeFile("file", filename, content)
	writer.close()

	return writer
}

type multipartWriter struct {
	boundary string
	body     *bytes.Buffer
}

func (w *multipartWriter) writeField(fieldname, value string) {
	_, _ = fmt.Fprintf(w.body, "--%s\r\n", w.boundary)
	_, _ = fmt.Fprintf(w.body, "Content-Disposition: form-data; name=\"%s\"\r\n", fieldname)
	w.body.WriteString("\r\n")
	w.body.WriteString(value)
	w.body.WriteString("\r\n")
}

func (w *multipartWriter) writeFile(fieldname, filename, content string) {
	_, _ = fmt.Fprintf(w.body, "--%s\r\n", w.boundary)
	_, _ = fmt.Fprintf(w.body, "Content-Disposition: form-data; name=\"%s\"; filename=\"%s\"\r\n", fieldname, filename)
	w.body.WriteString("Content-Type: text/plain\r\n")
	w.body.WriteString("\r\n")
	w.body.WriteString(content)
	w.body.WriteString("\r\n")
}

func (w *multipartWriter) close() {
	_, _ = fmt.Fprintf(w.body, "--%s\r\n", w.boundary)
}

func (w *multipartWriter) FormDataContentType() string {
	return fmt.Sprintf("multipart/form-data; boundary=%s", w.boundary)
}

type mockLogger struct {
	messages []string
	mu       sync.Mutex
}

func (m *mockLogger) log(level, msg string, args ...any) {
	m.mu.Lock()
	defer m.mu.Unlock()
	formatted := fmt.Sprintf("[%s] %s", level, msg)
	for i := 0; i < len(args); i += 2 {
		if i+1 < len(args) {
			formatted += fmt.Sprintf(" %v=%v", args[i], args[i+1])
		}
	}
	m.messages = append(m.messages, formatted)
}

func (m *mockLogger) Trace(msg string, args ...any) { m.log("TRACE", msg, args...) }
func (m *mockLogger) Debug(msg string, args ...any) { m.log("DEBUG", msg, args...) }
func (m *mockLogger) Info(msg string, args ...any)  { m.log("INFO", msg, args...) }
func (m *mockLogger) Warn(msg string, args ...any)  { m.log("WARN", msg, args...) }
func (m *mockLogger) Error(msg string, args ...any) { m.log("ERROR", msg, args...) }
func (m *mockLogger) Critical(msg string, args ...any) {
	m.log("CRITICAL", msg, args...)
}
func (m *mockLogger) With(args ...any) contracts.Logger { return m }

func (m *mockLogger) getMessages() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]string, len(m.messages))
	copy(result, m.messages)
	return result
}
