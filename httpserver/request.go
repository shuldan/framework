package httpserver

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
)

const DefaultMaxBodySize = 1 << 20

func PathParam(r *http.Request, name string) string {
	return r.PathValue(name)
}

func QueryParam(r *http.Request, name string) string {
	return r.URL.Query().Get(name)
}

func Bind(r *http.Request, target any) error {
	return BindWithLimit(r, target, DefaultMaxBodySize)
}

func BindWithLimit(
	r *http.Request, target any, maxBytes int64,
) error {
	if r.Body == nil || r.Body == http.NoBody {
		return ErrEmptyBody
	}

	limited := http.MaxBytesReader(nil, r.Body, maxBytes)
	defer func() { _ = limited.Close() }()

	dec := json.NewDecoder(limited)

	if err := dec.Decode(target); err != nil {
		return categorizeDecodeError(err)
	}

	return nil
}

func categorizeDecodeError(err error) error {
	var maxBytesErr *http.MaxBytesError
	if errors.As(err, &maxBytesErr) {
		return ErrBodyTooLarge
	}

	return fmt.Errorf("%w: %v", ErrInvalidJSON, err)
}
