package httpserver

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"

	domainerrors "github.com/shuldan/errors"
)

func JSON(w http.ResponseWriter, status int, data any) {
	buf, err := json.Marshal(data)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(status)
	_, _ = io.Copy(w, bytes.NewReader(buf))
}

func OK(w http.ResponseWriter, data any) {
	JSON(w, http.StatusOK, data)
}

func Created(w http.ResponseWriter, data any) {
	JSON(w, http.StatusCreated, data)
}

func NoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}

func Error(w http.ResponseWriter, err error) {
	status := domainerrors.ToHTTPStatus(err)
	body := domainerrors.ToPublicError(err)
	JSON(w, status, body)
}

func Wrap(
	fn func(http.ResponseWriter, *http.Request) error,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := fn(w, r); err != nil {
			Error(w, err)
		}
	}
}
