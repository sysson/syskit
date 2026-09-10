package iox

import (
	"bufio"
	"errors"
	"net"
	"net/http"
	"testing"
)

type testResponseWriter struct {
	header http.Header
}

func newTestResponseWriter() *testResponseWriter {
	return &testResponseWriter{header: make(http.Header)}
}

func (w *testResponseWriter) Header() http.Header {
	return w.header
}

func (w *testResponseWriter) Write(b []byte) (int, error) {
	return len(b), nil
}

func (w *testResponseWriter) WriteHeader(statusCode int) {}

type flushingResponseWriter struct {
	*testResponseWriter
	flushes int
}

func (w *flushingResponseWriter) Flush() {
	w.flushes++
}

type hijackingResponseWriter struct {
	*testResponseWriter
	hijacks int
}

var errHijacked = errors.New("hijacked")

func (w *hijackingResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	w.hijacks++
	return nil, nil, errHijacked
}

type flushingHijackingResponseWriter struct {
	*testResponseWriter
	flushes int
	hijacks int
}

func (w *flushingHijackingResponseWriter) Flush() {
	w.flushes++
}

func (w *flushingHijackingResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	w.hijacks++
	return nil, nil, errHijacked
}

func TestResponseWrapperOptionalInterfaces(t *testing.T) {
	t.Run("without optional interfaces", func(t *testing.T) {
		wrapper := NewResponseWrapper(newTestResponseWriter())

		if _, ok := wrapper.(http.Flusher); ok {
			t.Fatal("NewResponseWrapper() implements http.Flusher without underlying support")
		}
		if _, ok := wrapper.(http.Hijacker); ok {
			t.Fatal("NewResponseWrapper() implements http.Hijacker without underlying support")
		}
	})

	t.Run("with flusher", func(t *testing.T) {
		underlying := &flushingResponseWriter{testResponseWriter: newTestResponseWriter()}
		wrapper := NewResponseWrapper(underlying)

		flusher, ok := wrapper.(http.Flusher)
		if !ok {
			t.Fatal("NewResponseWrapper() does not implement http.Flusher with underlying support")
		}
		flusher.Flush()
		if underlying.flushes != 1 {
			t.Fatalf("Flush() calls = %d, want 1", underlying.flushes)
		}
		if _, ok := wrapper.(http.Hijacker); ok {
			t.Fatal("NewResponseWrapper() implements http.Hijacker without underlying support")
		}
	})

	t.Run("with hijacker", func(t *testing.T) {
		underlying := &hijackingResponseWriter{testResponseWriter: newTestResponseWriter()}
		wrapper := NewResponseWrapper(underlying)

		hijacker, ok := wrapper.(http.Hijacker)
		if !ok {
			t.Fatal("NewResponseWrapper() does not implement http.Hijacker with underlying support")
		}
		if _, _, err := hijacker.Hijack(); !errors.Is(err, errHijacked) {
			t.Fatalf("Hijack() error = %v, want %v", err, errHijacked)
		}
		if underlying.hijacks != 1 {
			t.Fatalf("Hijack() calls = %d, want 1", underlying.hijacks)
		}
		if _, ok := wrapper.(http.Flusher); ok {
			t.Fatal("NewResponseWrapper() implements http.Flusher without underlying support")
		}
	})

	t.Run("with flusher and hijacker", func(t *testing.T) {
		underlying := &flushingHijackingResponseWriter{testResponseWriter: newTestResponseWriter()}
		wrapper := NewResponseWrapper(underlying)

		flusher, ok := wrapper.(http.Flusher)
		if !ok {
			t.Fatal("NewResponseWrapper() does not implement http.Flusher with underlying support")
		}
		flusher.Flush()
		if underlying.flushes != 1 {
			t.Fatalf("Flush() calls = %d, want 1", underlying.flushes)
		}

		hijacker, ok := wrapper.(http.Hijacker)
		if !ok {
			t.Fatal("NewResponseWrapper() does not implement http.Hijacker with underlying support")
		}
		if _, _, err := hijacker.Hijack(); !errors.Is(err, errHijacked) {
			t.Fatalf("Hijack() error = %v, want %v", err, errHijacked)
		}
		if underlying.hijacks != 1 {
			t.Fatalf("Hijack() calls = %d, want 1", underlying.hijacks)
		}
	})
}
