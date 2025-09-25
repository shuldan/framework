package database

import (
	"testing"

	"github.com/google/uuid"
)

func TestUUID(t *testing.T) {
	t.Run("NewUUID", func(t *testing.T) {
		id := NewUUID()
		if !id.IsValid() {
			t.Error("new UUID should be valid")
		}
		if id.String() == "" {
			t.Error("UUID string should not be empty")
		}
	})

	t.Run("ParseUUID", func(t *testing.T) {
		validUUID := "550e8400-e29b-41d4-a716-446655440000"
		id, err := ParseUUID(validUUID)
		if err != nil {
			t.Errorf("failed to parse valid UUID: %v", err)
		}
		if id.String() != validUUID {
			t.Errorf("expected %s, got %s", validUUID, id.String())
		}

		_, err = ParseUUID("invalid-uuid")
		if err == nil {
			t.Error("expected error for invalid UUID")
		}
	})

	t.Run("MustParseUUID", func(t *testing.T) {
		validUUID := "550e8400-e29b-41d4-a716-446655440000"
		id := MustParseUUID(validUUID)
		if id.String() != validUUID {
			t.Errorf("expected %s, got %s", validUUID, id.String())
		}

		defer func() {
			if r := recover(); r == nil {
				t.Error("expected panic for invalid UUID")
			}
		}()
		MustParseUUID("invalid-uuid")
	})

	t.Run("UUID methods", func(t *testing.T) {
		originalUUID := uuid.New()
		id := UUID{value: originalUUID}

		if !id.IsValid() {
			t.Error("UUID should be valid")
		}
		if id.String() != originalUUID.String() {
			t.Error("String() method mismatch")
		}
		if id.UUID() != originalUUID {
			t.Error("UUID() method mismatch")
		}

		zeroID := UUID{value: uuid.Nil}
		if zeroID.IsValid() {
			t.Error("zero UUID should not be valid")
		}
	})
}

func TestIntID(t *testing.T) {
	t.Run("NewIntID", func(t *testing.T) {
		id := NewIntID(123).(IntID)
		if !id.IsValid() {
			t.Error("positive IntID should be valid")
		}
		if id.Int64() != 123 {
			t.Errorf("expected 123, got %d", id.Int64())
		}
		if id.String() != "123" {
			t.Errorf("expected '123', got '%s'", id.String())
		}

		zeroID := NewIntID(0)
		if zeroID.IsValid() {
			t.Error("zero IntID should not be valid")
		}

		negativeID := NewIntID(-1)
		if negativeID.IsValid() {
			t.Error("negative IntID should not be valid")
		}
	})

	t.Run("ParseIntID", func(t *testing.T) {
		contractId, err := ParseIntID("456")
		if err != nil {
			t.Errorf("failed to parse valid IntID: %v", err)
		}

		id := contractId.(IntID)
		if id.Int64() != 456 {
			t.Errorf("expected 456, got %d", id.Int64())
		}

		_, err = ParseIntID("not-a-number")
		if err == nil {
			t.Error("expected error for invalid IntID string")
		}
	})

	t.Run("MustParseIntID", func(t *testing.T) {
		id := MustParseIntID("789").(IntID)
		if id.Int64() != 789 {
			t.Errorf("expected 789, got %d", id.Int64())
		}

		defer func() {
			if r := recover(); r == nil {
				t.Error("expected panic for invalid IntID string")
			}
		}()
		MustParseIntID("not-a-number")
	})
}

func TestStringID(t *testing.T) {
	t.Run("NewStringID", func(t *testing.T) {
		id := NewStringID("test-id")
		if !id.IsValid() {
			t.Error("non-empty StringID should be valid")
		}
		if id.String() != "test-id" {
			t.Errorf("expected 'test-id', got '%s'", id.String())
		}

		emptyID := NewStringID("")
		if emptyID.IsValid() {
			t.Error("empty StringID should not be valid")
		}
	})

	t.Run("StringID validation", func(t *testing.T) {
		tests := []struct {
			value string
			valid bool
		}{
			{"test", true},
			{"test-123", true},
			{"", false},
			{"   ", true},
		}

		for _, tt := range tests {
			id := NewStringID(tt.value)
			if id.IsValid() != tt.valid {
				t.Errorf("StringID(%q).IsValid() = %v, want %v", tt.value, id.IsValid(), tt.valid)
			}
		}
	})
}

func TestIDInterface(t *testing.T) {
	var ids []interface {
		String() string
		IsValid() bool
	}

	ids = append(ids, NewUUID())
	ids = append(ids, NewIntID(1))
	ids = append(ids, NewStringID("test"))

	for i, id := range ids {
		if id.String() == "" && id.IsValid() {
			t.Errorf("ID %d: valid ID should have non-empty string representation", i)
		}
	}
}
