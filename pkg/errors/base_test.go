package errors

import (
	"strings"
	"testing"
)

func TestBaseErrors_ErrorDefinitions(t *testing.T) {
	testCases := []struct {
		name string
		err  *Error
	}{
		{"ErrValidation", ErrValidation},
		{"ErrAuth", ErrAuth},
		{"ErrPermission", ErrPermission},
		{"ErrNotFound", ErrNotFound},
		{"ErrConflict", ErrConflict},
		{"ErrBusiness", ErrBusiness},
		{"ErrInternal", ErrInternal},
		{"ErrTimeout", ErrTimeout},
		{"ErrUnavailable", ErrUnavailable},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.err == nil {
				t.Error("error should not be nil")
			}
			if string(tc.err.Code) == "" {
				t.Error("error code should not be empty")
			}
			if tc.err.Message == "" {
				t.Error("error message should not be empty")
			}
			if !strings.HasPrefix(string(tc.err.Code), "CORE") {
				t.Error("error code should have CORE prefix")
			}
		})
	}
}

func TestBaseErrors_UniqueCodesGenerated(t *testing.T) {
	codes := map[string]bool{}

	baseErrors := []*Error{
		ErrValidation, ErrAuth, ErrPermission, ErrNotFound,
		ErrConflict, ErrBusiness, ErrInternal, ErrTimeout, ErrUnavailable,
	}

	for _, err := range baseErrors {
		code := string(err.Code)
		if codes[code] {
			t.Errorf("duplicate code found: %s", code)
		}
		codes[code] = true
	}
}

func TestBaseErrors_ErrorMessages(t *testing.T) {
	expectedMessages := map[string]string{
		string(ErrValidation.Code):  "validation failed",
		string(ErrAuth.Code):        "authentication required",
		string(ErrPermission.Code):  "access denied",
		string(ErrNotFound.Code):    "resource not found",
		string(ErrConflict.Code):    "resource conflict",
		string(ErrBusiness.Code):    "business rule violated",
		string(ErrInternal.Code):    "internal error",
		string(ErrTimeout.Code):     "operation timeout",
		string(ErrUnavailable.Code): "service unavailable",
	}

	baseErrors := []*Error{
		ErrValidation, ErrAuth, ErrPermission, ErrNotFound,
		ErrConflict, ErrBusiness, ErrInternal, ErrTimeout, ErrUnavailable,
	}

	for _, err := range baseErrors {
		code := string(err.Code)
		expected, exists := expectedMessages[code]
		if !exists {
			t.Errorf("no expected message for code: %s", code)
			continue
		}
		if err.Message != expected {
			t.Errorf("expected message %q for code %s, got %q", expected, code, err.Message)
		}
	}
}
