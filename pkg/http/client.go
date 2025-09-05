package http

import (
	"bytes"
	"context"
	"crypto/rand"
	"io"
	"math"
	"net/http"
	"time"

	"github.com/shuldan/framework/pkg/contracts"
)

type ClientConfig struct {
	Timeout        time.Duration
	MaxRetries     int
	RetryWaitMin   time.Duration
	RetryWaitMax   time.Duration
	RetryCondition func(contracts.HTTPResponse, error) bool
}

func NewClientWithConfig(logger contracts.Logger, config ClientConfig) contracts.HTTPClient {
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}
	if config.MaxRetries == 0 {
		config.MaxRetries = 3
	}
	if config.RetryWaitMin == 0 {
		config.RetryWaitMin = time.Second
	}
	if config.RetryWaitMax == 0 {
		config.RetryWaitMax = 10 * time.Second
	}
	if config.RetryCondition == nil {
		config.RetryCondition = func(resp contracts.HTTPResponse, err error) bool {
			if err != nil {
				return true
			}
			return resp.StatusCode() >= 500 || resp.StatusCode() == 429
		}
	}

	return &httpClient{
		client: &http.Client{
			Timeout: config.Timeout,
		},
		logger: logger,
		config: config,
	}
}

type httpClient struct {
	client *http.Client
	logger contracts.Logger
	config ClientConfig
}

func NewClient(logger contracts.Logger) contracts.HTTPClient {
	return &httpClient{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger: logger,
	}
}

func (c *httpClient) Get(ctx context.Context, url string, opts ...contracts.HTTPRequestOption) (contracts.HTTPResponse, error) {
	req := NewHTTPRequest("GET", url, nil)
	for _, opt := range opts {
		opt(req)
	}
	return c.Do(ctx, req)
}

func (c *httpClient) Post(ctx context.Context, url string, body interface{}, opts ...contracts.HTTPRequestOption) (contracts.HTTPResponse, error) {
	req := NewHTTPRequest("POST", url, body)
	for _, opt := range opts {
		opt(req)
	}
	return c.Do(ctx, req)
}

func (c *httpClient) Put(ctx context.Context, url string, body interface{}, opts ...contracts.HTTPRequestOption) (contracts.HTTPResponse, error) {
	req := NewHTTPRequest("PUT", url, body)
	for _, opt := range opts {
		opt(req)
	}
	return c.Do(ctx, req)
}

func (c *httpClient) Delete(ctx context.Context, url string, opts ...contracts.HTTPRequestOption) (contracts.HTTPResponse, error) {
	req := NewHTTPRequest("DELETE", url, nil)
	for _, opt := range opts {
		opt(req)
	}
	return c.Do(ctx, req)
}

func (c *httpClient) Patch(ctx context.Context, url string, body interface{}, opts ...contracts.HTTPRequestOption) (contracts.HTTPResponse, error) {
	req := NewHTTPRequest("PATCH", url, body)
	for _, opt := range opts {
		opt(req)
	}
	return c.Do(ctx, req)
}

func (c *httpClient) Do(ctx context.Context, req contracts.HTTPRequest) (contracts.HTTPResponse, error) {
	if c.config.MaxRetries == 0 {
		response, err := c.doSingleRequest(ctx, req)
		if err != nil {
			return nil, ErrHTTPRequest.WithCause(err)
		}
		return response, nil
	}
	response, err := c.doWithRetry(ctx, req)
	if err != nil {
		return nil, ErrHTTPRequest.WithCause(err)
	}
	if response == nil {
		return nil, ErrHTTPRequest.WithDetail("reason", "max retries exceeded")
	}
	return response, nil
}

func (c *httpClient) doWithRetry(ctx context.Context, req contracts.HTTPRequest) (contracts.HTTPResponse, error) {
	var lastErr error
	for attempt := 0; attempt <= c.config.MaxRetries; attempt++ {
		if attempt > 0 {
			if err := c.waitForRetry(ctx, attempt, req.URL()); err != nil {
				return nil, err
			}
		}
		resp, err := c.doSingleRequest(ctx, req)
		if err != nil || resp != nil && !resp.IsSuccess() {
			lastErr = err
			if c.config.RetryCondition != nil && c.config.RetryCondition(resp, lastErr) {
				continue
			}
			break
		}
		return resp, nil
	}
	return nil, lastErr
}

func (c *httpClient) doSingleRequest(ctx context.Context, req contracts.HTTPRequest) (contracts.HTTPResponse, error) {
	httpReq, err := c.buildHTTPRequest(ctx, req)
	if err != nil {
		return nil, err
	}
	resp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	return c.processResponse(resp, req)
}

func (c *httpClient) buildHTTPRequest(ctx context.Context, req contracts.HTTPRequest) (*http.Request, error) {
	body := req.Body()
	httpReq, err := http.NewRequestWithContext(ctx, req.Method(), req.URL(), bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	for key, values := range req.Headers() {
		for _, value := range values {
			httpReq.Header.Add(key, value)
		}
	}
	return httpReq, nil
}

func (c *httpClient) processResponse(resp *http.Response, req contracts.HTTPRequest) (contracts.HTTPResponse, error) {
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil && c.logger != nil {
			c.logger.Error("Failed to close response body", "error", closeErr)
		}
	}()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return &httpResponse{
		statusCode: resp.StatusCode,
		headers:    resp.Header,
		body:       body,
		request:    req,
	}, nil
}

func (c *httpClient) waitForRetry(ctx context.Context, attempt int, url string) error {
	waitTime := c.calculateRetryWait(attempt)
	c.logger.Debug("Retrying HTTP request",
		"attempt", attempt,
		"wait_time", waitTime,
		"url", url)
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(waitTime):
		return nil
	}
}

func (c *httpClient) calculateRetryWait(attempt int) time.Duration {
	waitTime := time.Duration(float64(c.config.RetryWaitMin) * math.Pow(2, float64(attempt-1)))
	if waitTime > c.config.RetryWaitMax {
		waitTime = c.config.RetryWaitMax
	}
	jitter := c.generateSecureJitter(waitTime)
	calculated := waitTime + jitter - time.Duration(float64(waitTime)*0.25)
	if calculated < c.config.RetryWaitMin {
		return c.config.RetryWaitMin
	}
	return calculated
}

func (c *httpClient) generateSecureJitter(waitTime time.Duration) time.Duration {
	maxJitter := float64(waitTime) * 0.5
	maxJitterInt := int64(maxJitter)
	if maxJitterInt <= 0 {
		return 0
	}
	randomBytes := make([]byte, 8)
	if _, err := rand.Read(randomBytes); err != nil {
		if c.logger != nil {
			c.logger.Warn("Failed to generate secure random jitter, using time-based fallback", "error", err)
		}
		return time.Duration(time.Now().UnixNano() % maxJitterInt)
	}
	randomInt64 := int64(0)
	for i, b := range randomBytes {
		randomInt64 |= int64(b) << (i * 8)
	}
	if randomInt64 < 0 {
		randomInt64 = -randomInt64
	}
	return time.Duration(randomInt64 % maxJitterInt)
}
