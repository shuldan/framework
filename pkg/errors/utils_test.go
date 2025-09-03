package errors

import (
	"errors"
	"strings"
	"testing"
)

func TestIs(t *testing.T) {
	err1 := errors.New("test error")
	err2 := errors.New("another error")

	if !Is(err1, err1) {
		t.Error("should return true for same error")
	}

	if Is(err1, err2) {
		t.Error("should return false for different errors")
	}

	frameworkErr := ErrValidation
	if !Is(frameworkErr, frameworkErr) {
		t.Error("should return true for same framework error")
	}
}

func TestAs(t *testing.T) {
	frameworkErr := ErrValidation
	var target *Error
	if !As(frameworkErr, &target) {
		t.Error("should return true when error matches target type")
	}

	genericErr := errors.New("generic error")
	var target2 *Error
	if As(genericErr, &target2) {
		t.Error("should return false when error doesn't match target type")
	}
}

func TestUnwrap(t *testing.T) {
	cause := errors.New("cause error")
	wrappedErr := ErrInternal.WithCause(cause)

	unwrapped := Unwrap(wrappedErr)
	if !errors.Is(unwrapped, cause) {
		t.Error("should unwrap to cause error")
	}

	simpleErr := errors.New("simple error")
	unwrapped = Unwrap(simpleErr)
	if unwrapped != nil {
		t.Error("should return nil for non-wrapped error")
	}
}

func TestJoin(t *testing.T) {
	err1 := errors.New("error 1")
	err2 := errors.New("error 2")
	err3 := errors.New("error 3")

	joined := Join(err1, err2, err3)
	if joined == nil {
		t.Fatal("joined error should not be nil")
	}

	joinedStr := joined.Error()
	if !strings.Contains(joinedStr, "error 1") {
		t.Error("joined error should contain first error")
	}
	if !strings.Contains(joinedStr, "error 2") {
		t.Error("joined error should contain second error")
	}
	if !strings.Contains(joinedStr, "error 3") {
		t.Error("joined error should contain third error")
	}
}

func TestJoin_WithNils(t *testing.T) {
	err1 := errors.New("error 1")

	joined := Join(nil, err1, nil)
	if joined == nil {
		t.Fatal("joined error should not be nil")
	}

	joinedStr := joined.Error()
	if !strings.Contains(joinedStr, "error 1") {
		t.Error("joined error should contain non-nil error")
	}
}

func TestJoin_AllNils(t *testing.T) {
	joined := Join(nil, nil, nil)
	if joined != nil {
		t.Error("joined error should be nil when all inputs are nil")
	}
}

func TestIs_WithWrappedErrors(t *testing.T) {
	cause := errors.New("cause")
	wrapped := ErrInternal.WithCause(cause)

	if !Is(wrapped, cause) {
		t.Error("should find cause in wrapped error")
	}

	if Is(wrapped, ErrValidation) {
		t.Error("should not match different error types")
	}
}

func TestAs_WithWrappedErrors(t *testing.T) {
	cause := ErrValidation
	wrapped := ErrInternal.WithCause(cause)

	var validationErr *Error
	if !As(wrapped, &validationErr) {
		t.Error("should find validation error in wrapped chain")
	}
}

func TestErrorChaining_Complex(t *testing.T) {
	originalErr := errors.New("original error")

	level1 := ErrInternal.WithCause(originalErr)
	level2 := ErrBusiness.WithCause(level1)
	level3 := ErrValidation.WithCause(level2)

	if !Is(level3, originalErr) {
		t.Error("should find original error through multiple levels")
	}

	var businessErr *Error
	if !As(level3, &businessErr) {
		t.Error("should find business error in chain")
	}

	unwrapped := Unwrap(level3)
	if !errors.Is(unwrapped, level2) {
		t.Error("should unwrap to immediate cause")
	}
}

func TestUtils_NilHandling(t *testing.T) {
	if Is(nil, nil) {
		t.Error("Is should return false for nil errors")
	}

	var target *Error
	if As(nil, &target) {
		t.Error("As should handle nil errors correctly")
	}

	if Unwrap(nil) != nil {
		t.Error("Unwrap should return nil for nil input")
	}

	if Join() != nil {
		t.Error("Join with no arguments should return nil")
	}
}
