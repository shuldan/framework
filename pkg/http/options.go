package http

import (
	"context"
	"encoding/base64"
	"time"

	"github.com/shuldan/framework/pkg/contracts"
)

func WithHeader(key, value string) contracts.HTTPRequestOption {
	return func(req contracts.HTTPRequest) {
		if r, ok := req.(*httpRequest); ok {
			r.AddHeader(key, value)
		}
	}
}

func WithHeaders(headers map[string]string) contracts.HTTPRequestOption {
	return func(req contracts.HTTPRequest) {
		if r, ok := req.(*httpRequest); ok {
			for key, value := range headers {
				r.SetHeader(key, value)
			}
		}
	}
}

func WithTimeout(timeout time.Duration) contracts.HTTPRequestOption {
	return func(req contracts.HTTPRequest) {
		if r, ok := req.(*httpRequest); ok {
			ctx, cancel := context.WithTimeout(r.Context(), timeout)
			_ = cancel // We can't call cancel here as the context will be used later
			r.SetContext(ctx)
		}
	}
}

func WithContext(ctx context.Context) contracts.HTTPRequestOption {
	return func(req contracts.HTTPRequest) {
		if r, ok := req.(*httpRequest); ok {
			r.SetContext(ctx)
		}
	}
}

func WithBasicAuth(username, password string) contracts.HTTPRequestOption {
	return func(req contracts.HTTPRequest) {
		if r, ok := req.(*httpRequest); ok {
			auth := base64.StdEncoding.EncodeToString([]byte(username + ":" + password))
			r.SetHeader("Authorization", "Basic "+auth)
		}
	}
}

func WithBearerToken(token string) contracts.HTTPRequestOption {
	return func(req contracts.HTTPRequest) {
		if r, ok := req.(*httpRequest); ok {
			r.SetHeader("Authorization", "Bearer "+token)
		}
	}
}

func WithUserAgent(userAgent string) contracts.HTTPRequestOption {
	return func(req contracts.HTTPRequest) {
		if r, ok := req.(*httpRequest); ok {
			r.SetHeader("User-Agent", userAgent)
		}
	}
}
