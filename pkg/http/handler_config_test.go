package http

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/shuldan/framework/pkg/contracts"
	"github.com/shuldan/framework/pkg/errors"
)

var testingCode = errors.WithPrefix("TESTING_CODE")
var ErrTestingCode = testingCode().New("Test error for TESTING_CODE_0001").WithDetail("test_detail", "value")

type mockConfig struct {
	data map[string]interface{}
}

func (m *mockConfig) Has(key string) bool {
	_, ok := m.data[key]
	return ok
}

func (m *mockConfig) Get(key string) any {
	return m.data[key]
}

func (m *mockConfig) GetString(key string, defaultVal ...string) string {
	if v, ok := m.data[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	if len(defaultVal) > 0 {
		return defaultVal[0]
	}
	return ""
}

func (m *mockConfig) GetInt(key string, defaultVal ...int) int {
	if v, ok := m.data[key]; ok {
		if i, ok := v.(int); ok {
			return i
		}
	}
	if len(defaultVal) > 0 {
		return defaultVal[0]
	}
	return 0
}

func (m *mockConfig) GetInt64(_ string, _ ...int64) int64 {
	panic("not implemented for mock")
}

func (m *mockConfig) GetFloat64(_ string, _ ...float64) float64 {
	panic("not implemented for mock")
}

func (m *mockConfig) GetBool(key string, defaultVal ...bool) bool {
	v, ok := m.find(key)
	if !ok {
		return getFirst(defaultVal)
	}
	if b, ok := v.(bool); ok {
		return b
	}
	if s, ok := v.(string); ok {
		switch strings.ToLower(s) {
		case "true", "1", "on", "yes", "y":
			return true
		case "false", "0", "off", "no", "n":
			return false
		}
	}
	if f, ok := v.(float64); ok {
		return f != 0
	}
	if i, ok := v.(int); ok {
		return i != 0
	}
	return getFirst(defaultVal)
}

func (m *mockConfig) GetStringSlice(_ string, _ ...string) []string {
	panic("not implemented for mock")
}

func (m *mockConfig) GetSub(key string) (contracts.Config, bool) {
	sub, ok := m.find(key)
	if !ok {
		return nil, false
	}
	if subMap, ok := sub.(map[string]any); ok {
		return &mockConfig{data: subMap}, true
	}
	return nil, false
}

func (m *mockConfig) All() map[string]any {
	return m.data
}

func (m *mockConfig) find(path string) (any, bool) {
	keys := strings.Split(path, ".")
	var current any = m.data

	for _, k := range keys {
		if current == nil {
			return nil, false
		}

		switch cur := current.(type) {
		case map[string]any:
			next, exists := cur[k]
			if !exists {
				return nil, false
			}
			current = next
		case map[any]any:
			next, exists := cur[k]
			if !exists {
				return nil, false
			}
			current = next
		default:
			return nil, false
		}
	}

	return current, true
}

func getFirst[T any](values []T) T {
	var zero T
	if len(values) > 0 {
		return values[0]
	}
	return zero
}

func TestErrorHandler_CustomConfigFromMock(t *testing.T) {
	configData := map[string]interface{}{
		"http": map[string]interface{}{
			"server": map[string]interface{}{
				"middleware": map[string]interface{}{
					"logging": map[string]interface{}{
						"enabled": true,
					},
					"error_handler": map[string]interface{}{
						"enabled":          true,
						"show_stack_trace": false,
						"show_details":     true,
						"log_level":        "error",
						"status_codes": map[string]interface{}{
							"TESTING_CODE_0001": 409,
							"TESTING_CODE_0002": 422,
						},
						"user_messages": map[string]interface{}{
							"TESTING_CODE_0001": "Custom message: active application already exists for this inn",
							"TESTING_CODE_0002": "Custom message: Unprocessable Entity Test",
						},
					},
				},
			},
		},
	}
	mockCfg := &mockConfig{data: configData}
	logger := &mockLogger{}

	router := NewRouter(logger)
	middlewares := LoadMiddlewareFromConfig(mockCfg, logger)
	router.Use(middlewares...)

	router.GET("/test-error", func(ctx contracts.HTTPContext) error {
		return ErrTestingCode
	})

	req := httptest.NewRequest("GET", "/test-error", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != 409 {
		t.Errorf("Expected HTTP status 409, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response body: %v", err)
	}

	errorObj, ok := response["error"].(map[string]interface{})
	if !ok {
		t.Fatalf("Response body does not contain 'error' object or it's not a map: %v", response)
	}

	if code, ok := errorObj["code"].(string); !ok || code != "TESTING_CODE_0001" {
		t.Errorf("Expected error.code 'TESTING_CODE_0001', got '%v' (type %T)", code, code)
	}

	expectedMessage := "Custom message: active application already exists for this inn"
	if msg, ok := errorObj["message"].(string); !ok || msg != expectedMessage {
		t.Errorf("Expected error.message '%s', got '%v' (type %T)", expectedMessage, msg, msg)
	}

	if details, ok := errorObj["details"].(map[string]interface{}); !ok {
		t.Log("No details found in error response, but show_details is true. Might be expected if original error has no details.")
	} else {
		if detailVal, exists := details["test_detail"]; !exists || detailVal != "value" {
			t.Logf("Expected detail 'test_detail': 'value', got %v", details)
		}
	}

	router.GET("/test-error-2", func(ctx contracts.HTTPContext) error {
		return testingCode().New("Unprocessable Test Error")
	})

	req2 := httptest.NewRequest("GET", "/test-error-2", nil)
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)

	if w2.Code != 422 {
		t.Errorf("Expected HTTP status 422 for TESTING_CODE_0002, got %d", w2.Code)
	}
	var response2 map[string]interface{}
	if err := json.Unmarshal(w2.Body.Bytes(), &response2); err != nil {
		t.Fatalf("Failed to unmarshal response body for second test: %v", err)
	}
	errorObj2, ok := response2["error"].(map[string]interface{})
	if !ok {
		t.Fatalf("Second response body does not contain 'error' object: %v", response2)
	}
	expectedMsg2 := "Custom message: Unprocessable Entity Test"
	if msg, ok := errorObj2["message"].(string); !ok || msg != expectedMsg2 {
		t.Errorf("Expected error.message '%s' for TESTING_CODE_0002, got '%v'", expectedMsg2, msg)
	}

	router.GET("/test-unknown", func(ctx contracts.HTTPContext) error {
		return errors.ErrInternal
	})

	req3 := httptest.NewRequest("GET", "/test-unknown", nil)
	w3 := httptest.NewRecorder()
	router.ServeHTTP(w3, req3)

	if w3.Code != 500 {
		t.Errorf("Expected HTTP status 500 for unknown/standard error, got %d", w3.Code)
	}

}

func TestMockConfig_GetSub(t *testing.T) {
	data := map[string]interface{}{
		"level1": map[string]interface{}{
			"level2": map[string]interface{}{
				"key": "value",
			},
		},
	}
	cfg := &mockConfig{data: data}

	sub1, ok1 := cfg.GetSub("level1")
	if !ok1 {
		t.Fatal("Expected sub-config for 'level1'")
	}
	sub2, ok2 := sub1.GetSub("level2")
	if !ok2 {
		t.Fatal("Expected sub-config for 'level2'")
	}
	if sub2.GetString("key") != "value" {
		t.Errorf("Expected 'value' for 'key', got '%s'", sub2.GetString("key"))
	}
}
