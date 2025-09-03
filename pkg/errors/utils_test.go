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

func TestIs_NilComparisons(t *testing.T) {
	err := errors.New("test error")

	if Is(nil, err) {
		t.Error("nil should not match non-nil error")
	}

	if Is(err, nil) {
		t.Error("non-nil error should not match nil")
	}

	if Is(nil, nil) {
		t.Error("both nil should return false according to implementation")
	}
}

type customError struct {
	inner error
}

func (c customError) Error() string {
	return "custom: " + c.inner.Error()
}

func (c customError) Is(target error) bool {
	return errors.Is(c.inner, target)
}

func TestJoin_SingleError(t *testing.T) {
	err := errors.New("single error")

	joined := Join(err)
	if !errors.Is(joined, err) {
		t.Error("joining single error should return the same error")
	}
}

func TestJoin_ErrorOrdering(t *testing.T) {
	err1 := errors.New("first")
	err2 := errors.New("second")
	err3 := errors.New("third")

	var joinedStr string
	joined := Join(err1, err2, err3)
	if joined != nil {
		joinedStr = joined.Error()
	}
	pos1 := strings.Index(joinedStr, "first")
	pos2 := strings.Index(joinedStr, "second")
	pos3 := strings.Index(joinedStr, "third")

	if pos1 > pos2 || pos2 > pos3 {
		t.Error("errors should appear in the same order as they were joined")
	}
}

func TestIs_WithCustomError(t *testing.T) {
	innerErr := errors.New("inner")
	customErr := customError{inner: innerErr}

	if !Is(customErr, innerErr) {
		t.Error("should find inner error through custom Is method")
	}
}

func TestUnwrap_MultipleLevel(t *testing.T) {
	level0 := errors.New("level 0")
	level1 := ErrInternal.WithCause(level0)
	level2 := ErrBusiness.WithCause(level1)
	level3 := ErrValidation.WithCause(level2)

	unwrapped1 := Unwrap(level3)
	if !errors.Is(unwrapped1, level2) {
		t.Error("first unwrap should return level2")
	}

	unwrapped2 := Unwrap(unwrapped1)
	if !errors.Is(unwrapped2, level1) {
		t.Error("second unwrap should return level1")
	}

	unwrapped3 := Unwrap(unwrapped2)
	if !errors.Is(unwrapped3, level0) {
		t.Error("third unwrap should return level0")
	}

	unwrapped4 := Unwrap(unwrapped3)
	if unwrapped4 != nil {
		t.Error("final unwrap should return nil")
	}
}

func TestJoin_WithFrameworkErrors(t *testing.T) {
	err1 := ErrValidation.WithDetail("field", "username")
	err2 := ErrPermission.WithDetail("resource", "user")
	genericErr := errors.New("generic error")

	joined := Join(err1, err2, genericErr)

	if !Is(joined, err1) {
		t.Error("should find validation error in joined error")
	}
	if !Is(joined, err2) {
		t.Error("should find permission error in joined error")
	}
	if !Is(joined, genericErr) {
		t.Error("should find generic error in joined error")
	}
}

func TestAs_WithJoinedErrors(t *testing.T) {
	validationErr := ErrValidation.WithDetail("field", "email")
	businessErr := ErrBusiness.WithDetail("rule", "unique_email")

	joined := Join(validationErr, businessErr)

	var target *Error
	if !As(joined, &target) {
		t.Error("should be able to extract Error from joined errors")
	}
}
