package iox

import (
	"io"
	"sync/atomic"
)

type flusher interface {
	Flush()
}

// WriteFlusher is an io.WriteCloser that flushes its underlying writer after
// each write.
type WriteFlusher interface {
	io.WriteCloser
	flusher
}

type writeFlusher struct {
	w       io.Writer
	flusher flusher

	closed atomic.Bool
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
	wf.flusher.Flush()
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
// write. If w does not implement Flush(), Flush is a no-op.
func NewWriteFlusher(w io.Writer) WriteFlusher {
	var fl flusher
	if f, ok := w.(flusher); ok {
		fl = f
	} else {
		fl = &nopFlusher{}
	}
	return &writeFlusher{w: w, flusher: fl}
}
