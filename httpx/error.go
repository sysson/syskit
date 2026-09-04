// Package httpx provides utilities for handling HTTP errors and responses in a consistent manner.
//
// #Basic Usage
//
// Example: Handling a not found error.
// httpx.NotFound(errors.New("resource not found")).Write(w)
//

package httpx

import (
	"encoding/json/v2"
	"fmt"
	"net/http"
)

// HTTPError represents an error that occurred while handling an HTTP request. It includes the HTTP status code and an error message.
type HTTPError struct {
	StatusCode int
	Message    error
}

// NewHTTPError creates a new HTTPError with the given status code and error message.
func NewHTTPError(statusCode int, err error) *HTTPError {
	return &HTTPError{
		StatusCode: statusCode,
		Message:    err,
	}
}

// Write writes the HTTP error as a JSON response with the appropriate status code.
func (e *HTTPError) WriteJSON(w http.ResponseWriter) error {
	return WriteJSON(w, e.StatusCode, Message(e.Message.Error()))
}

// Unwrap returns the underlying error message of the HTTPError.
func (e *HTTPError) Unwrap() error {
	return e.Message
}

// Error returns a string representation of the HTTPError, including the status code and error message.
func (e *HTTPError) Error() string {
	return fmt.Sprintf("HTTP error %d: %s", e.StatusCode, e.Message)
}

// MarshalJSON customizes the JSON representation of the HTTPError.
func (e *HTTPError) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		StatusCode int    `json:"status_code"`
		Message    string `json:"message"`
	}{
		StatusCode: e.StatusCode,
		Message:    e.Message.Error(),
	})
}

// InternalServerError creates a new HTTPError representing an internal server error.
func InternalServerError(err error) *HTTPError {
	return NewHTTPError(http.StatusInternalServerError, err)
}

// NotFound creates a new HTTPError representing a not found error.
func NotFound(err error) *HTTPError {
	return NewHTTPError(http.StatusNotFound, err)
}

// Forbidden creates a new HTTPError representing a forbidden error.
func Forbidden(err error) *HTTPError {
	return NewHTTPError(http.StatusForbidden, err)
}

// Unauthorized creates a new HTTPError representing an unauthorized error.
func Unauthorized(err error) *HTTPError {
	return NewHTTPError(http.StatusUnauthorized, err)
}

// RequestTimeout creates a new HTTPError representing a request timeout error.
func RequestTimeout(err error) *HTTPError {
	return NewHTTPError(http.StatusRequestTimeout, err)
}

// BadRequest creates a new HTTPError representing a bad request error.
func BadRequest(err error) *HTTPError {
	return NewHTTPError(http.StatusBadRequest, err)
}

// Conflict creates a new HTTPError representing a conflict error.
func Conflict(err error) *HTTPError {
	return NewHTTPError(http.StatusConflict, err)
}

// Gone creates a new HTTPError representing a gone error.
func Gone(err error) *HTTPError {
	return NewHTTPError(http.StatusGone, err)
}
