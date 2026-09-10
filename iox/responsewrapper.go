package iox

import (
	"bufio"
	"net"
	"net/http"
	"sync/atomic"
)

// ResponseWrapper is an interface that wraps http.ResponseWriter and provides
// additional methods to track the status code and number of bytes written.
type ResponseWrapper interface {
	http.ResponseWriter
	StatusCode() int
	BytesWritten() int
}

// responseWrapper is the concrete implementation of ResponseWrapper.
type responseWrapper struct {
	w            http.ResponseWriter
	statusCode   int
	wroteHeader  atomic.Bool
	bytesWritten atomic.Int64
}

type responseWrapperFlusher struct {
	*responseWrapper
	flusher http.Flusher
}

type responseWrapperHijacker struct {
	*responseWrapper
	hijacker http.Hijacker
}

type responseWrapperFlusherHijacker struct {
	*responseWrapper
	flusher  http.Flusher
	hijacker http.Hijacker
}

// StatusCode returns the HTTP status code of the response. If WriteHeader has
// not been called yet, it returns 0.
func (rw *responseWrapper) StatusCode() int {
	return rw.statusCode
}

// Header returns the header map that will be sent by WriteHeader.
func (rw *responseWrapper) Header() http.Header {
	return rw.w.Header()
}

// Write writes the data to the underlying http.ResponseWriter. If WriteHeader
// has not been called yet, it calls WriteHeader with http.StatusOK.
func (rw *responseWrapper) Write(b []byte) (int, error) {
	if !rw.wroteHeader.Load() {
		rw.WriteHeader(http.StatusOK)
	}
	n, err := rw.w.Write(b)
	rw.bytesWritten.Add(int64(n))
	return n, err
}

// WriteHeader sends an HTTP response header with the provided status code.
// If WriteHeader has already been called, it does nothing.
func (rw *responseWrapper) WriteHeader(statusCode int) {
	if rw.wroteHeader.Load() {
		return
	}
	rw.statusCode = statusCode
	rw.wroteHeader.Store(true)
	rw.w.WriteHeader(statusCode)
}

// BytesWritten returns the number of bytes written to the response.
func (rw *responseWrapper) BytesWritten() int {
	return int(rw.bytesWritten.Load())
}

// Flush sends any buffered response data to the client if the underlying
// ResponseWriter supports flushing.
func (rw *responseWrapperFlusher) Flush() {
	rw.flusher.Flush()
}

// Hijack lets the caller take over the connection if the underlying
// ResponseWriter supports hijacking.
func (rw *responseWrapperHijacker) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return rw.hijacker.Hijack()
}

// Flush sends any buffered response data to the client.
func (rw *responseWrapperFlusherHijacker) Flush() {
	rw.flusher.Flush()
}

// Hijack lets the caller take over the connection.
func (rw *responseWrapperFlusherHijacker) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return rw.hijacker.Hijack()
}

// NewResponseWrapper wraps an http.ResponseWriter and returns a ResponseWrapper
// that tracks the status code and number of bytes written.
func NewResponseWrapper(w http.ResponseWriter) ResponseWrapper {
	rw := &responseWrapper{w: w}
	flusher, canFlush := w.(http.Flusher)
	hijacker, canHijack := w.(http.Hijacker)

	switch {
	case canFlush && canHijack:
		return &responseWrapperFlusherHijacker{responseWrapper: rw, flusher: flusher, hijacker: hijacker}
	case canFlush:
		return &responseWrapperFlusher{responseWrapper: rw, flusher: flusher}
	case canHijack:
		return &responseWrapperHijacker{responseWrapper: rw, hijacker: hijacker}
	default:
		return rw
	}
}
