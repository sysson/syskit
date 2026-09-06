package stream

import (
	"bytes"
	"io"
	"testing"
)

type shortWriter struct{}

func (shortWriter) Write(p []byte) (int, error) {
	return len(p) - 1, nil
}

func TestStreamWriterReportsNoInputBytesOnShortFormattedWrite(t *testing.T) {
	writer := NewStdoutWriter(shortWriter{})

	n, err := writer.Write([]byte("output"))
	if n != 0 {
		t.Fatalf("Write() bytes = %d, want 0", n)
	}
	if err != io.ErrShortWrite {
		t.Fatalf("Write() error = %v, want %v", err, io.ErrShortWrite)
	}
}

func TestStdoutWriterFormatsOutput(t *testing.T) {
	var output bytes.Buffer

	if _, err := NewStdoutWriter(&output).Write([]byte("output")); err != nil {
		t.Fatalf("Write() error = %v", err)
	}
	if got, want := output.String(), `{"stream":"output"}`+streamNewline; got != want {
		t.Fatalf("formatted output = %q, want %q", got, want)
	}
}
