// Package httpx provides middleware utilities for handling HTTP errors and logging in a consistent manner.

package httpx

import (
	"errors"
	"net/http"

	"github.com/sysson/syskit/logx"
)

// Status code used to indicate that the client closed the request.
const statusClientClosedRequest = 499

var (
	// ErrHTTPServerError indicates an internal server error.
	ErrHTTPServerError = errors.New("internal server error")
)

// ErrorLogger is a middleware that logs errors returned by an HTTP handler.
// It handles client cancellations, HTTP errors, and server errors appropriately.
// It should be used to wrap HTTP handlers that return errors, providing consistent error handling and logging.
// Use the logx package to set the appropriate logging context.
func ErrorLogger(handler HTTPErrorFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := handler(w, r)
		// If no error occurred, simply return.
		if err == nil {
			return
		}
		// If the request was cancelled by the client, handle it separately.
		if contextErr := r.Context().Err(); contextErr != nil {
			logx.G(r.Context()).Info("request cancelled by client", "method", r.Method, "url", r.URL.String(), "error", err)
			// Return a 499 status code to indicate that the client closed the request.
			w.WriteHeader(statusClientClosedRequest)
			return
		}
		// If the error is an HTTPError, handle it accordingly.
		if httpResp, ok := errors.AsType[*HTTPError](err); ok {
			// Log server errors (status code >= 500) and to prevent leaking internal server details to the client, always return a generic internal server error message for server errors.
			if httpResp.StatusCode >= 500 {
				logx.G(r.Context()).Error("handler returned error", "method", r.Method, "url", r.URL.String(), "error", httpResp.Error())
				_ = WriteJSON(w, httpResp.StatusCode, Message(ErrHTTPServerError.Error()))
				return
			} else if httpResp.StatusCode >= 400 {
				// Log errors (status code >= 400 and < 500) at the warning level.
				logx.G(r.Context()).Warn("handler returned client error", "method", r.Method, "url", r.URL.String(), "error", httpResp.Error())
			}
			// For non-server errors, write the specific HTTP error response.
			_ = httpResp.WriteJSON(w)
			return
		}
		// If the error is not an HTTPError, treat it as a server error.
		logx.G(r.Context()).Error("server error", "method", r.Method, "url", r.URL.String(), "error", err)
		_ = WriteJSON(w, http.StatusInternalServerError, Message(ErrHTTPServerError.Error()))
	}
}
