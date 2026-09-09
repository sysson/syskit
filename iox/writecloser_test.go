package iox

import (
	"bytes"
	"errors"
	"testing"
)

func TestWriteCloserWrapperClosesOnce(t *testing.T) {
	closeErr := errors.New("cleanup failed")
	closeCalls := 0
	writer := NewWriteCloserWrapper(&bytes.Buffer{}, func() error {
		closeCalls++
		return closeErr
	})

	if err := writer.Close(); !errors.Is(err, closeErr) {
		t.Fatalf("first Close() error = %v, want %v", err, closeErr)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("second Close() error = %v, want nil", err)
	}
	if closeCalls != 1 {
		t.Fatalf("close callback calls = %d, want 1", closeCalls)
	}
}
