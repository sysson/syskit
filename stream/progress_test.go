package stream

import (
	"context"
	"io"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestChanOutputCloseDoesNotWaitForBlockedConsumer(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	progressChan := make(chan Progress)
	output := ChanOutputContext(ctx, progressChan)

	if err := output.WriteProgress(Progress{Action: "accepted"}); err != nil {
		t.Fatalf("WriteProgress() error = %v, want nil", err)
	}

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
		t.Fatal("Close() returned before context cancellation")
	case <-time.After(50 * time.Millisecond):
	}

	cancel()

	select {
	case <-closed:
	case <-time.After(time.Second):
		t.Fatal("Close() did not return after context cancellation")
	}
	if err := <-writeReturned; err != io.EOF {
		t.Fatalf("WriteProgress() error = %v, want %v", err, io.EOF)
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

func TestChanOutputCloseDeliversAcceptedWrites(t *testing.T) {
	const n = 200

	progressChan := make(chan Progress, 1)
	output := ChanOutput(progressChan)

	var accepted atomic.Int64
	var writers sync.WaitGroup
	writers.Add(n)
	for i := range n {
		go func(i int) {
			defer writers.Done()
			if err := output.WriteProgress(Progress{ID: strconv.Itoa(i)}); err == nil {
				accepted.Add(1)
			}
		}(i)
	}

	received := make(chan Progress, n)
	done := make(chan struct{})
	go func() {
		for {
			select {
			case <-done:
				return
			case p := <-progressChan:
				received <- p
			}
		}
	}()

	writers.Wait()
	if err := output.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	defer close(done)

	want := int(accepted.Load())
	for got := 0; got < want; got++ {
		select {
		case <-received:
		case <-time.After(time.Second):
			t.Fatalf("received %d messages, want %d", got, want)
		}
	}
}

func TestChanOutputWriteProgressAfterClose(t *testing.T) {
	output := ChanOutput(make(chan Progress))

	if err := output.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("WriteProgress() panicked after Close(): %v", r)
		}
	}()
	if err := output.WriteProgress(Progress{Action: "after-close"}); err != io.EOF {
		t.Fatalf("WriteProgress() error = %v, want %v", err, io.EOF)
	}
}
