// Package httpx provides utilities for handling HTTP requests and responses, including JSON parsing and writing, context value retrieval, and standardized error handling.

package httpx

import (
	"context"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"net/http"
)

// HTTPErrorFunc defines a function type for HTTP handlers that return an error.
type HTTPErrorFunc func(w http.ResponseWriter, r *http.Request) error

// KeyFromContext retrieves a value of type V from the context using the specified key of type T.
// It returns the value and a boolean indicating whether the value was found and successfully cast to type V.
func KeyFromContext[T, V any](ctx context.Context, key T) (V, bool) {
	if v := ctx.Value(key); v != nil {
		if value, ok := v.(V); ok {
			return value, true
		}
	}
	var zero V
	return zero, false
}

// WriteJSON writes the given value as a JSON response with the specified HTTP status code.
func WriteJSON(w http.ResponseWriter, code int, v any) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	return json.MarshalWrite(w, v,
		jsontext.EscapeForHTML(false),
		json.Deterministic(true),
	)
}

// ParseJSON parses the JSON request body into the specified value.
// It closes the request body after reading.
func ParseJSON(r *http.Request, v any) error {
	defer func() {
		_ = r.Body.Close()
	}()
	return json.UnmarshalRead(r.Body, v)
}

// Message creates a simple JSON object with a "message" field containing the specified string.
// This is useful for sending standardized error or status messages in JSON responses.
func Message(s string) map[string]string {
	return map[string]string{"message": s}
}
