package errors

import (
	"bytes"
	"errors"
	"fmt"
	"log/slog"
	"runtime"
	"sync/atomic"
	"text/template"
	"time"
)

type Code string

func (c Code) New(msg string) *Error {
	return &Error{
		Code:      c,
		Message:   msg,
		Details:   make(map[string]interface{}),
		Stack:     getStack(),
		Timestamp: time.Now(),
	}
}

func WithPrefix(prefix string) func() Code {
	counter := uint64(0)
	return func() Code {
		next := atomic.AddUint64(&counter, 1)
		return Code(fmt.Sprintf("%s_%04d", prefix, next))
	}
}

type Error struct {
	Code      Code                   `json:"code"`
	Message   string                 `json:"message"`
	Details   map[string]interface{} `json:"details,omitempty"`
	Cause     error                  `json:"-"`
	Stack     string                 `json:"-"`
	Timestamp time.Time              `json:"timestamp"`
}

func (e *Error) Error() string {
	defer func() {
		if r := recover(); r != nil {
			slog.Error("panic",
				"error", r,
				"stack", slog.String("stack", e.Stack),
			)
		}
	}()

	t, err := template.New("error").Parse(e.Message)
	if err != nil {
		return e.formatSimpleMessage()
	}

	var output bytes.Buffer
	err = t.Execute(&output, e.Details)
	if err != nil {
		return e.formatSimpleMessage()
	}

	o := output.String()
	r := []rune(o)

	if len(r) == 0 {
		return ""
	}

	msg := string(r)

	if e.Cause != nil {
		return fmt.Sprintf("%s: %s (caused by: %v)", e.Code, msg, e.Cause)
	}

	return fmt.Sprintf("%s: %s", e.Code, msg)
}

func (e *Error) formatSimpleMessage() string {
	r := []rune(e.Message)
	if len(r) == 0 {
		return ""
	}

	msg := string(r)

	if e.Cause != nil {
		return fmt.Sprintf("%s: %s (caused by: %s)", e.Code, msg, e.Cause.Error())
	}

	return fmt.Sprintf("%s: %s", e.Code, msg)
}

func (e *Error) WithCause(err error) *Error {
	clone := &Error{
		Code:      e.Code,
		Message:   e.Message,
		Details:   make(map[string]interface{}),
		Cause:     err,
		Stack:     e.Stack,
		Timestamp: e.Timestamp,
	}

	for k, v := range e.Details {
		clone.Details[k] = v
	}

	return clone
}

func (e *Error) WithDetail(key string, value interface{}) *Error {
	clone := &Error{
		Code:      e.Code,
		Message:   e.Message,
		Details:   make(map[string]interface{}),
		Cause:     e.Cause,
		Stack:     e.Stack,
		Timestamp: e.Timestamp,
	}

	for k, v := range e.Details {
		clone.Details[k] = v
	}
	clone.Details[key] = value

	return clone
}

func (e *Error) Unwrap() error {
	return e.Cause
}

func (e *Error) Is(target error) bool {
	var targetErr *Error
	if errors.As(target, &targetErr) {
		return e.Code == targetErr.Code
	}
	return false
}

func getStack() string {
	buf := make([]byte, 4096)
	n := runtime.Stack(buf, false)
	return string(buf[:n])
}
