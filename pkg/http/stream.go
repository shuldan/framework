package http

import (
	"context"
	"errors"
	"net/http"
	"sync"

	"github.com/shuldan/framework/pkg/contracts"
)

type httpStreamingContext struct {
	ctx     *httpContext
	flusher http.Flusher
	closed  chan struct{}
	once    sync.Once
}

func (s *httpStreamingContext) CloseNotify() <-chan struct{} {
	s.once.Do(func() {
		s.closed = make(chan struct{})

		go func() {
			<-s.ctx.req.Context().Done()
			if errors.Is(s.ctx.req.Context().Err(), context.Canceled) {
				close(s.closed)
			}
		}()
	})
	return s.closed
}

func (s *httpStreamingContext) IsClientClosed() bool {
	select {
	case <-s.CloseNotify():
		return true
	default:
		return false
	}
}

func (s *httpStreamingContext) SetHeader(key, value string) contracts.HTTPStreamingContext {
	s.ctx.SetHeader(key, value)
	return s
}

func (s *httpStreamingContext) SetContentType(contentType string) contracts.HTTPStreamingContext {
	s.ctx.SetHeader("Content-Type", contentType)
	return s
}

func (s *httpStreamingContext) WriteChunk(data []byte) error {
	if s.ctx.statusCode == 0 {
		if _, exists := s.ctx.resp.Header()["Content-Type"]; !exists {
			s.ctx.SetHeader("Content-Type", "text/plain")
		}
		s.ctx.statusCode = http.StatusOK
		s.ctx.resp.WriteHeader(s.ctx.statusCode)
	}
	_, err := s.ctx.resp.Write(data)
	s.Flush()
	return err
}

func (s *httpStreamingContext) WriteStringChunk(str string) error {
	return s.WriteChunk([]byte(str))
}

func (s *httpStreamingContext) Flush() {
	if s.flusher == nil {
		s.flusher, _ = s.ctx.resp.(http.Flusher)
	}
	if s.flusher != nil {
		s.flusher.Flush()
	}
}
