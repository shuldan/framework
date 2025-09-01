package http

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"time"

	"github.com/shuldan/framework/pkg/contracts"
)

type Client struct {
	client *http.Client
	logger contracts.Logger
}

func NewClient(logger contracts.Logger) *Client {
	return &Client{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger: logger,
	}
}

func (c *Client) Get(ctx context.Context, url string, opts ...contracts.HTTPRequestOption) (contracts.HTTPResponse, error) {
	req := NewHTTPRequest("GET", url, nil)
	for _, opt := range opts {
		opt(req)
	}
	return c.Do(ctx, req)
}

func (c *Client) Post(ctx context.Context, url string, body interface{}, opts ...contracts.HTTPRequestOption) (contracts.HTTPResponse, error) {
	req := NewHTTPRequest("POST", url, body)
	for _, opt := range opts {
		opt(req)
	}
	return c.Do(ctx, req)
}

func (c *Client) Put(ctx context.Context, url string, body interface{}, opts ...contracts.HTTPRequestOption) (contracts.HTTPResponse, error) {
	req := NewHTTPRequest("PUT", url, body)
	for _, opt := range opts {
		opt(req)
	}
	return c.Do(ctx, req)
}

func (c *Client) Delete(ctx context.Context, url string, opts ...contracts.HTTPRequestOption) (contracts.HTTPResponse, error) {
	req := NewHTTPRequest("DELETE", url, nil)
	for _, opt := range opts {
		opt(req)
	}
	return c.Do(ctx, req)
}

func (c *Client) Patch(ctx context.Context, url string, body interface{}, opts ...contracts.HTTPRequestOption) (contracts.HTTPResponse, error) {
	req := NewHTTPRequest("PATCH", url, body)
	for _, opt := range opts {
		opt(req)
	}
	return c.Do(ctx, req)
}

func (c *Client) Do(ctx context.Context, req contracts.HTTPRequest) (contracts.HTTPResponse, error) {
	httpReq, err := http.NewRequestWithContext(ctx, req.Method(), req.URL(), bytes.NewReader(req.Body()))
	if err != nil {
		return nil, ErrHTTPRequest.WithCause(err)
	}

	for key, values := range req.Headers() {
		for _, value := range values {
			httpReq.Header.Add(key, value)
		}
	}

	resp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, ErrHTTPRequest.WithCause(err)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		if err = resp.Body.Close(); err != nil && c.logger != nil {
			c.logger.Error("Failed to close response body", "error", err)
			return nil, ErrHTTPRequest.WithCause(err)
		}
		return nil, ErrHTTPRequest.WithCause(err)
	}
	if err = resp.Body.Close(); err != nil && c.logger != nil {
		c.logger.Error("Failed to close response body", "error", err)
		return nil, ErrHTTPRequest.WithCause(err)
	}

	return &HTTPResponseImpl{
		statusCode: resp.StatusCode,
		headers:    resp.Header,
		body:       body,
		request:    req,
	}, nil
}
