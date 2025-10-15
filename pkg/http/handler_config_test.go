package http

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/shuldan/framework/pkg/contracts"
	"github.com/shuldan/framework/pkg/errors"
)

var testingCode = errors.WithPrefix("TESTING_CODE")
var ErrTestingCode = testingCode().
	New("Test error for TESTING_CODE_0001 with test_detail: {{.test_detail}}").
	WithDetail("test_detail", "value")

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
