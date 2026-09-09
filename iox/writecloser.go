package iox

import (
	"io"
	"sync/atomic"
)

type writeCloserWrapper struct {
	io.Writer
	closer func() error
	closed atomic.Bool
}

func (r *writeCloserWrapper) Close() error {
	if !r.closed.CompareAndSwap(false, true) {
		return nil
	}
	return r.closer()
}

// NewWriteCloserWrapper combines an io.Writer with a custom close function.
// Close invokes closer at most once; subsequent calls return nil.
func NewWriteCloserWrapper(r io.Writer, closer func() error) io.WriteCloser {
	return &writeCloserWrapper{
		Writer: r,
		closer: closer,
	}
}
