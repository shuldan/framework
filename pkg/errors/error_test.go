package errors

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestCode_New(t *testing.T) {
	code := Code("TEST_001")
	err := code.New("something went wrong")

	if err.Code != code {
		t.Errorf("expected code %s, got %s", code, err.Code)
	}
	if err.Message != "something went wrong" {
		t.Errorf("expected message 'something went wrong', got %s", err.Message)
	}
	if err.Details == nil {
		t.Error("expected Details to be initialized")
	}
	if err.Stack == "" {
		t.Error("expected Stack to be filled")
	}
	if err.Timestamp.IsZero() {
		t.Error("expected Timestamp to be set")
	}
}

func TestWithPrefix(t *testing.T) {
	gen := WithPrefix("API")
	c1 := gen()
	c2 := gen()
	c3 := gen()

	if c1 != "API_0001" {
		t.Errorf("expected API_0001, got %s", c1)
	}
	if c2 != "API_0002" {
		t.Errorf("expected API_0002, got %s", c2)
	}
	if c3 != "API_0003" {
		t.Errorf("expected API_0003, got %s", c3)
	}
}

func TestError_Error_Simple(t *testing.T) {
	code := Code("TEST_001")
	err := code.New("simple error")

	expected := "TEST_001: simple error"
	if err.Error() != expected {
		t.Errorf("expected %s, got %s", expected, err.Error())
	}
}

func TestError_Error_WithTemplate(t *testing.T) {
	code := Code("TEST_001")
	err := code.New("hello {{.name}}").
		WithDetail("name", "world")

	expected := "TEST_001: hello world"
	if err.Error() != expected {
		t.Errorf("expected %s, got %s", expected, err.Error())
	}
}

func TestError_Error_InvalidTemplate(t *testing.T) {
	code := Code("TEST_001")
	err := code.New("hello {{.name")

	expected := "TEST_001: hello {{.name"
	if err.Error() != expected {
		t.Errorf("expected %s, got %s", expected, err.Error())
	}
}

func TestError_Error_WithCause(t *testing.T) {
	cause := errors.New("cause error")
	code := Code("TEST_001")
	err := code.New("wrapped error").WithCause(cause)

	expected := "TEST_001: wrapped error (caused by: cause error)"
	if err.Error() != expected {
		t.Errorf("expected %s, got %s", expected, err.Error())
	}
}

func TestError_Error_WithCauseAndTemplate(t *testing.T) {
	cause := errors.New("cause error")
	code := Code("TEST_001")
	err := code.New("hello {{.name}}").
		WithDetail("name", "world").
		WithCause(cause)

	expected := "TEST_001: hello world (caused by: cause error)"
	if err.Error() != expected {
		t.Errorf("expected %s, got %s", expected, err.Error())
	}
}

func TestError_Error_EmptyMessage(t *testing.T) {
	code := Code("TEST_001")
	err := code.New("")

	if err.Error() != "" {
		t.Errorf("expected empty string, got %s", err.Error())
	}
}

func TestError_WithDetail(t *testing.T) {
	code := Code("TEST_001")
	err := code.New("test").
		WithDetail("key1", "value1").
		WithDetail("key2", 123)

	if err.Details["key1"] != "value1" {
		t.Errorf("expected value1, got %v", err.Details["key1"])
	}
	if err.Details["key2"] != 123 {
		t.Errorf("expected 123, got %v", err.Details["key2"])
	}
}

func TestError_Unwrap(t *testing.T) {
	cause := errors.New("cause error")
	code := Code("TEST_001")
	err := code.New("wrapped").WithCause(cause)

	unwrapped := err.Unwrap()
	if !errors.Is(unwrapped, cause) {
		t.Errorf("expected cause error, got %v", unwrapped)
	}
}

func TestError_Stack(t *testing.T) {
	code := Code("TEST_001")
	err := code.New("stack test")

	if err.Stack == "" {
		t.Error("expected Stack to be filled")
	}
	if !strings.Contains(err.Stack, "TestError_Stack") {
		t.Error("expected stack to contain TestError_Stack")
	}
}

func TestError_Error_PanicRecovery(t *testing.T) {
	code := Code("TEST_001")

	err := code.New("hello {{.Name.Func}}")
	err.Details["Name"] = "world"

	result := err.Error()
	if !strings.HasPrefix(result, "TEST_001:") {
		t.Errorf("expected error to start with TEST_001:, got %s", result)
	}
}

func TestError_WithDetail_Chaining(t *testing.T) {
	code := Code("TEST_001")
	err := code.New("test").
		WithDetail("key1", "value1").
		WithDetail("key2", "value2")

	if err.Details["key1"] != "value1" {
		t.Errorf("Expected key1=value1, got %v", err.Details["key1"])
	}

	if err.Details["key2"] != "value2" {
		t.Errorf("Expected key2=value2, got %v", err.Details["key2"])
	}
}

func TestError_Unwrap_Nil(t *testing.T) {
	code := Code("TEST_001")
	err := code.New("test without cause")

	unwrapped := err.Unwrap()
	if unwrapped != nil {
		t.Errorf("Expected nil unwrapped error, got %v", unwrapped)
	}
}

func TestError_Error_ComplexTemplate(t *testing.T) {
	code := Code("TEST_001")
	err := code.New("User {{.user}} (ID: {{.id}}) failed with reason: {{.reason}}").
		WithDetail("user", "john").
		WithDetail("id", 123).
		WithDetail("reason", "permission denied")

	expected := "TEST_001: User john (ID: 123) failed with reason: permission denied"
	if err.Error() != expected {
		t.Errorf("Expected %s, got %s", expected, err.Error())
	}
}

func TestError_Error_TemplateWithMissingKey(t *testing.T) {
	code := Code("TEST_001")
	err := code.New("Hello {{.name}}, welcome to {{.place}}").
		WithDetail("name", "John")

	result := err.Error()
	if !strings.HasPrefix(result, "TEST_001:") {
		t.Errorf("Expected error to start with TEST_001:, got %s", result)
	}
}

func TestError_Timestamp(t *testing.T) {
	code := Code("TEST_001")
	before := time.Now()
	err := code.New("test")
	after := time.Now()

	if err.Timestamp.Before(before) || err.Timestamp.After(after) {
		t.Error("Timestamp should be set during error creation")
	}
}
