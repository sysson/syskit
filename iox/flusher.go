package iox

import (
	"errors"
	"io"
	"net/http"
	"sync/atomic"
)

// WriteFlusher is an io.WriteCloser that flushes its underlying writer after
// each write and exposes whether it has been flushed after closing.
type WriteFlusher interface {
	io.WriteCloser
	http.Flusher

	// Flushed reports whether the writer was flushed before it was closed. It
	// returns an error while the writer is still open because another flush may
	// occur before the method returns.
	Flushed() (bool, error)
}

type writeFlusher struct {
	w           io.Writer
	flusher     http.Flusher
	flushedOnce atomic.Bool
	closed      atomic.Bool
}

func (wf *writeFlusher) Write(b []byte) (int, error) {
	if wf.closed.Load() {
		return 0, io.EOF
	}

	n, err := wf.w.Write(b)
	wf.Flush()
	return n, err
}

func (wf *writeFlusher) Flush() {
	if wf.closed.Load() {
		return
	}

	if !wf.flushedOnce.Load() {
		wf.flushedOnce.Store(true)
	}
	wf.flusher.Flush()
}

func (wf *writeFlusher) Flushed() (bool, error) {
	if wf.closed.Load() {
		return wf.flushedOnce.Load(), nil
	}
	// If the writeFlusher is not closed, we cannot guarantee that it won't get flushed upon return.
	return false, errors.New("writeFlusher is not closed yet")
}

func (wf *writeFlusher) Close() error {
	if wf.closed.Load() {
		return io.EOF
	}
	wf.closed.Store(true)
	return nil
}

type nopFlusher struct{}

func (f *nopFlusher) Flush() {}

// NewWriteFlusher wraps w and flushes it after every successful or partial
// write. If w does not implement http.Flusher, Flush is a no-op.
func NewWriteFlusher(w io.Writer) WriteFlusher {
	var fl http.Flusher
	if f, ok := w.(http.Flusher); ok {
		fl = f
	} else {
		fl = &nopFlusher{}
	}
	return &writeFlusher{w: w, flusher: fl}
}
