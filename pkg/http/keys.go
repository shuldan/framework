package http

type contextKey struct{}

var ContextKey = contextKey{}

type requestIDKey struct{}

var RequestID = requestIDKey{}

type requestStartKey struct{}

var RequestStart = requestStartKey{}
