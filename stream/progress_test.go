package stream

import (
	"io"
	"testing"
	"time"
)

func TestChanOutputCloseDoesNotWaitForBlockedConsumer(t *testing.T) {
	progressChan := make(chan Progress)
	output := ChanOutput(progressChan)

	writeReturned := make(chan error)
	go func() {
		writeReturned <- output.WriteProgress(Progress{Action: "blocked"})
	}()

	closed := make(chan struct{})
	go func() {
		_ = output.Close()
		close(closed)
	}()

	select {
	case <-closed:
	case <-time.After(time.Second):
		t.Fatal("Close() waited for an inactive consumer")
	}
	if err := <-writeReturned; err != nil && err != io.EOF {
		t.Fatalf("WriteProgress() error = %v, want nil or %v", err, io.EOF)
	}
}

func TestChanOutputCloseIsIdempotent(t *testing.T) {
	output := ChanOutput(make(chan Progress))

	if err := output.Close(); err != nil {
		t.Fatalf("first Close() error = %v", err)
	}
	if err := output.Close(); err != nil {
		t.Fatalf("second Close() error = %v", err)
	}
}
