package iox

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"net"
	"net/http"
)

// ResponseModifier wraps an http.ResponseWriter to capture, buffer, and modify HTTP response data
// before it is sent to the client. It allows inspecting and overriding status codes, headers, and
// response body without immediately flushing to the underlying writer. This is useful for
// middleware that needs to transform responses or implement response filtering.
// The maximum buffer size is 64KB; exceeding this will auto-flush to the underlying writer.
type ResponseModifier interface {
	http.ResponseWriter
	http.Flusher

	// RawBody returns the buffered response body.
	RawBody() []byte

	// RawHeaders returns the buffered headers as raw bytes.
	RawHeaders() ([]byte, error)

	// StatusCode returns the HTTP status code that will be sent.
	StatusCode() int

	// OverrideBody replaces the buffered response body with new content.
	OverrideBody(b []byte)

	// OverrideHeader replaces all headers from JSON-encoded header data.
	OverrideHeader(b []byte) error

	// OverrideStatusCode replaces the HTTP status code.
	OverrideStatusCode(statusCode int)

	// FlushAll writes all buffered data (headers and body) to the underlying ResponseWriter.
	FlushAll() error

	// Hijacked returns true if the connection has been hijacked.
	Hijacked() bool
}

// NewResponseModifier wraps an http.ResponseWriter to enable response interception and modification.
func NewResponseModifier(rw http.ResponseWriter) ResponseModifier {
	return &responseModifier{rw: rw, header: make(http.Header)}
}

// maxBufferSize is the maximum size of buffered response body before auto-flushing.
const maxBufferSize = 64 * 1024

// responseModifier is the concrete implementation of ResponseModifier.
type responseModifier struct {
	rw         http.ResponseWriter
	body       []byte
	header     http.Header
	statusCode int
	hijacked   bool
}

// Hijacked returns whether the connection has been hijacked.
func (rm *responseModifier) Hijacked() bool {
	return rm.hijacked
}

// WriteHeader sets the status code. If hijacked, delegates to the underlying ResponseWriter.
func (rm *responseModifier) WriteHeader(s int) {
	if rm.hijacked {
		rm.rw.WriteHeader(s)
		return
	}

	rm.statusCode = s
}

// Header returns the buffered headers. If hijacked, returns headers from the underlying ResponseWriter.
func (rm *responseModifier) Header() http.Header {
	if rm.hijacked {
		return rm.rw.Header()
	}

	return rm.header
}

// StatusCode returns the buffered status code.
func (rm *responseModifier) StatusCode() int {
	return rm.statusCode
}

// OverrideBody replaces the buffered body content.
func (rm *responseModifier) OverrideBody(b []byte) {
	rm.body = b
}

// OverrideStatusCode replaces the buffered status code.
func (rm *responseModifier) OverrideStatusCode(statusCode int) {
	rm.statusCode = statusCode
}

// OverrideHeader replaces buffered headers from JSON-encoded header data.
func (rm *responseModifier) OverrideHeader(b []byte) error {
	header := http.Header{}
	if err := json.Unmarshal(b, &header); err != nil {
		return err
	}
	rm.header = header
	return nil
}

// Write appends data to the buffered body. If the buffer exceeds maxBufferSize, it auto-flushes.
// If hijacked, delegates to the underlying ResponseWriter.
func (rm *responseModifier) Write(b []byte) (int, error) {
	if rm.hijacked {
		return rm.rw.Write(b)
	}

	if len(rm.body)+len(b) > maxBufferSize {
		rm.Flush()
	}
	rm.body = append(rm.body, b...)
	return len(b), nil
}

// RawBody returns the buffered response body.
func (rm *responseModifier) RawBody() []byte {
	return rm.body
}

// RawHeaders returns the buffered headers as raw HTTP header bytes.
func (rm *responseModifier) RawHeaders() ([]byte, error) {
	var b bytes.Buffer
	if err := rm.header.Write(&b); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

// Hijack attempts to hijack the underlying connection, flushing all buffered data first.
// After hijacking, the ResponseModifier cannot be used for normal writes.
func (rm *responseModifier) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	rm.hijacked = true
	_ = rm.FlushAll()

	hijacker, ok := rm.rw.(http.Hijacker)
	if !ok {
		return nil, nil, errors.New("internal response writer doesn't support the Hijacker interface")
	}
	return hijacker.Hijack()
}

// Flush flushes all buffered data and calls Flush on the underlying ResponseWriter if it supports it.
func (rm *responseModifier) Flush() {
	flusher, ok := rm.rw.(http.Flusher)
	if !ok {
		return
	}

	_ = rm.FlushAll()
	flusher.Flush()
}

// FlushAll writes all buffered headers and body to the underlying ResponseWriter.
// It writes headers first, then the status code, then the body.
func (rm *responseModifier) FlushAll() error {
	for k, vv := range rm.header {
		for _, v := range vv {
			rm.rw.Header().Add(k, v)
		}
	}

	if rm.statusCode > 0 {
		rm.rw.WriteHeader(rm.statusCode)
	}

	var err error
	if len(rm.body) > 0 {
		var n int
		n, err = rm.rw.Write(rm.body)
		rm.body = rm.body[n:]
	}

	rm.statusCode = 0
	rm.header = http.Header{}
	return err
}
