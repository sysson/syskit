package iox

import (
	"io"
	"net/http"
	"sync/atomic"
)

// WriteFlusher is an io.WriteCloser that flushes its underlying writer after
// each write and exposes whether it has been flushed after closing.
type WriteFlusher interface {
	io.WriteCloser
	http.Flusher

	// HasWritten reports whether the writer has successfully written any data.
	HasWritten() bool
}

type writeFlusher struct {
	w       io.Writer
	flusher http.Flusher
	wrote   atomic.Bool
	closed  atomic.Bool
}

func (wf *writeFlusher) Write(b []byte) (int, error) {
	if wf.closed.Load() {
		return 0, io.EOF
	}

	wf.wrote.Store(true)
	n, err := wf.w.Write(b)
	wf.Flush()
	return n, err
}

func (wf *writeFlusher) Flush() {
	if wf.closed.Load() {
		return
	}
	wf.flusher.Flush()
}

// HasWritten reports whether the writer has successfully written any data. It
// can be called at any time, even after the writer is closed.
// If using a http.ResponseWriter, it's no longer possible to modify the headers, when true.
func (wf *writeFlusher) HasWritten() bool {
	return wf.wrote.Load()
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
