package stream

import (
	"bytes"
	"errors"
	"testing"
)

func TestFormatStatus(t *testing.T) {
	got := string(FormatStatus("job-1", "processed %d items", 3))
	want := `{"status":"processed 3 items","id":"job-1"}` + streamNewline
	if got != want {
		t.Fatalf("FormatStatus() = %q, want %q", got, want)
	}
}

func TestFormatError(t *testing.T) {
	got := string(FormatError(&StreamError{Code: 7, Message: "failed"}))
	want := `{"errorDetail":{"code":7,"message":"failed"}}` + streamNewline
	if got != want {
		t.Fatalf("FormatError() = %q, want %q", got, want)
	}
}

func TestJSONProgressOutputWritesLastUpdateMarker(t *testing.T) {
	var output bytes.Buffer
	writer := NewJSONProgressOutput(&output, true)

	err := writer.WriteProgress(Progress{ID: "job-1", Action: "done", Current: 2, Total: 2, LastUpdate: true})
	if err != nil {
		t.Fatalf("WriteProgress() error = %v", err)
	}
	want := `{"status":"done","progressDetail":{"current":2,"total":2},"id":"job-1"}` + streamNewline + `{}` + streamNewline
	if output.String() != want {
		t.Fatalf("output = %q, want %q", output.String(), want)
	}
}

func TestJSONProgressOutputReturnsWriterError(t *testing.T) {
	writeErr := errors.New("write failed")
	writer := NewJSONProgressOutput(errorWriter{err: writeErr}, false)

	if err := writer.WriteProgress(Progress{Action: "working"}); !errors.Is(err, writeErr) {
		t.Fatalf("WriteProgress() error = %v, want %v", err, writeErr)
	}
}

type errorWriter struct {
	err error
}

func (w errorWriter) Write([]byte) (int, error) {
	return 0, w.err
}
