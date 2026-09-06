package stream

import (
	"bytes"
	"io"
	"testing"
)

type progressRecorder struct {
	updates []Progress
}

func (r *progressRecorder) WriteProgress(progress Progress) error {
	r.updates = append(r.updates, progress)
	return nil
}

func (r *progressRecorder) Close() error { return nil }

func TestProgressReaderReportsCompletion(t *testing.T) {
	recorder := &progressRecorder{}
	reader := NewProgressReader(io.NopCloser(bytes.NewReader([]byte("hello"))), recorder, 5, "job-1", "upload")

	data, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}
	if string(data) != "hello" {
		t.Fatalf("read data = %q, want %q", data, "hello")
	}
	if len(recorder.updates) == 0 {
		t.Fatal("no progress updates recorded")
	}
	last := recorder.updates[len(recorder.updates)-1]
	if last.ID != "job-1" || last.Action != "upload" || last.Current != 5 || last.Total != 5 || !last.LastUpdate {
		t.Fatalf("last progress = %+v, want completed final update", last)
	}
}

func TestProgressReaderCloseReportsFinalProgress(t *testing.T) {
	recorder := &progressRecorder{}
	reader := NewProgressReader(io.NopCloser(bytes.NewReader([]byte("hello"))), recorder, 10, "job-1", "upload")

	buf := make([]byte, 2)
	if _, err := reader.Read(buf); err != nil {
		t.Fatalf("Read() error = %v", err)
	}
	if err := reader.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	if len(recorder.updates) == 0 || recorder.updates[len(recorder.updates)-1].Current != 10 {
		t.Fatalf("final progress = %+v, want current 10", recorder.updates)
	}
}
