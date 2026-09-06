package iox

import (
	"bytes"
	"io"
	"testing"
)

type recordingFlusher struct {
	bytes.Buffer
	flushes int
}

func (w *recordingFlusher) Flush() {
	w.flushes++
}

func TestWriteFlusherFlushesAfterWrite(t *testing.T) {
	underlying := &recordingFlusher{}
	writer := NewWriteFlusher(underlying)

	n, err := writer.Write([]byte("hello"))
	if err != nil {
		t.Fatalf("Write() error = %v", err)
	}
	if n != 5 {
		t.Fatalf("Write() bytes = %d, want 5", n)
	}
	if got := underlying.flushes; got != 1 {
		t.Fatalf("flush count = %d, want 1", got)
	}
	if flushed, err := writer.Flushed(); err == nil || flushed {
		t.Fatalf("Flushed() = (%v, %v), want (false, error) while open", flushed, err)
	}

	if err := writer.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	if flushed, err := writer.Flushed(); err != nil || !flushed {
		t.Fatalf("Flushed() = (%v, %v), want (true, nil) after close", flushed, err)
	}
	if _, err := writer.Write([]byte("after close")); err != io.EOF {
		t.Fatalf("Write() after Close() error = %v, want %v", err, io.EOF)
	}
}

func TestWriteFlusherWithoutFlusherStillWrites(t *testing.T) {
	var underlying bytes.Buffer
	writer := NewWriteFlusher(&underlying)

	if _, err := writer.Write([]byte("hello")); err != nil {
		t.Fatalf("Write() error = %v", err)
	}
	if got := underlying.String(); got != "hello" {
		t.Fatalf("underlying output = %q, want %q", got, "hello")
	}
}
