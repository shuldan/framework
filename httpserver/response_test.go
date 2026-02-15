package httpserver

import (
	"encoding/json"
	"math"
	"net/http"
	"net/http/httptest"
	"testing"

	domainerrors "github.com/shuldan/errors"
)

func TestJSON(t *testing.T) {
	t.Parallel()
	rr := httptest.NewRecorder()
	JSON(rr, http.StatusOK, map[string]string{"key": "val"})
	assertStatus(t, http.StatusOK, rr)
	assertHeader(t, "Content-Type", "application/json", rr)
	var result map[string]string
	_ = json.Unmarshal(rr.Body.Bytes(), &result)
	if result["key"] != "val" {
		t.Fatalf("expected 'val', got %q", result["key"])
	}
}

func TestJSON_MarshalError(t *testing.T) {
	t.Parallel()
	rr := httptest.NewRecorder()
	JSON(rr, http.StatusOK, math.Inf(1))
	assertStatus(t, http.StatusInternalServerError, rr)
}

func TestOK(t *testing.T) {
	t.Parallel()
	rr := httptest.NewRecorder()
	OK(rr, map[string]int{"count": 5})
	assertStatus(t, http.StatusOK, rr)
}

func TestCreated(t *testing.T) {
	t.Parallel()
	rr := httptest.NewRecorder()
	Created(rr, map[string]string{"id": "1"})
	assertStatus(t, http.StatusCreated, rr)
}

func TestNoContent(t *testing.T) {
	t.Parallel()
	rr := httptest.NewRecorder()
	NoContent(rr)
	assertStatus(t, http.StatusNoContent, rr)
	if rr.Body.Len() != 0 {
		t.Fatal("expected empty body")
	}
}

func TestError_DomainError(t *testing.T) {
	t.Parallel()
	err := domainerrors.NewCode("USER_NOT_FOUND").
		Kind(domainerrors.NotFound).
		New("user not found")
	rr := httptest.NewRecorder()
	Error(rr, err)
	assertStatus(t, http.StatusNotFound, rr)
	assertHeader(t, "Content-Type", "application/json", rr)
	var body domainerrors.PublicError
	_ = json.Unmarshal(rr.Body.Bytes(), &body)
	if body.Code != "USER_NOT_FOUND" {
		t.Fatalf("expected code 'USER_NOT_FOUND', got %q", body.Code)
	}
}

func TestError_GenericError(t *testing.T) {
	t.Parallel()
	rr := httptest.NewRecorder()
	Error(rr, http.ErrNoCookie)
	assertStatus(t, http.StatusInternalServerError, rr)
}

func TestWrap_Success(t *testing.T) {
	t.Parallel()
	handler := Wrap(func(w http.ResponseWriter, _ *http.Request) error {
		OK(w, "ok")
		return nil
	})
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, httptest.NewRequest("GET", "/", nil))
	assertStatus(t, http.StatusOK, rr)
}

func TestWrap_Error(t *testing.T) {
	t.Parallel()
	domainErr := domainerrors.NewCode("VALIDATION").
		Kind(domainerrors.Validation).
		New("invalid input")
	handler := Wrap(func(_ http.ResponseWriter, _ *http.Request) error {
		return domainErr
	})
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, httptest.NewRequest("GET", "/", nil))
	assertStatus(t, http.StatusBadRequest, rr)
}
