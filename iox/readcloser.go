// Package iox provides utilities for working with I/O operations, particularly for
// wrapping and modifying HTTP response streams and providing cancellable I/O operations.
package iox

import (
	"context"
	"io"
	"sync/atomic"
)

// readCloserWrapper wraps an io.Reader with a custom close function, allowing you to
// provide your own cleanup logic when the reader is closed. Close is idempotent and
// safe to call multiple times.
type readCloserWrapper struct {
	io.Reader
	closer func() error
	closed atomic.Bool
}

// Close calls the custom closer function, ensuring it is only called once.
func (r *readCloserWrapper) Close() error {
	if !r.closed.CompareAndSwap(false, true) {
		return nil
	}
	return r.closer()
}

// NewReadCloserWrapper wraps an io.Reader with a custom close function.
// The closer function is called exactly once when Close() is invoked.
// This is useful when you want to combine a reader with custom cleanup logic,
// such as releasing resources or triggering cleanup callbacks.
func NewReadCloserWrapper(r io.Reader, closer func() error) io.ReadCloser {
	return &readCloserWrapper{
		Reader: r,
		closer: closer,
	}
}

// cancelReadCloser wraps an io.ReadCloser and allows it to be cancelled via context.
// It uses an io.Pipe internally to bridge between the original reader and the caller,
// enabling graceful cancellation when the context is done.
type cancelReadCloser struct {
	cancel func()
	pR     *io.PipeReader
	pW     *io.PipeWriter
	closed atomic.Bool
}

// NewCancelReadCloser wraps an io.ReadCloser with context cancellation support.
// When the provided context is cancelled or done, the read operation will be interrupted
// with the context error (ctx.Err()). This is useful for implementing timeouts or
// explicit cancellation of long-running I/O operations. The original reader is closed
// when the context is done or when the returned ReadCloser is closed.
func NewCancelReadCloser(ctx context.Context, in io.ReadCloser) io.ReadCloser {
	pR, pW := io.Pipe()

	doneCtx, cancel := context.WithCancel(ctx)

	p := &cancelReadCloser{
		cancel: cancel,
		pR:     pR,
		pW:     pW,
	}

	go func() {
		_, err := io.Copy(pW, in)
		select {
		case <-ctx.Done():
		default:
			p.closeWithError(err)
		}
		_ = in.Close()
	}()
	go func() {
		for {
			select {
			case <-ctx.Done():
				p.closeWithError(ctx.Err())
			case <-doneCtx.Done():
				return
			}
		}
	}()

	return p
}

// Read reads from the underlying reader, respecting context cancellation.
func (p *cancelReadCloser) Read(buf []byte) (int, error) {
	return p.pR.Read(buf)
}

// closeWithError closes the pipe writer with the given error and cancels the internal context.
func (p *cancelReadCloser) closeWithError(err error) {
	_ = p.pW.CloseWithError(err)
	p.cancel()
}

// Close closes the reader gracefully. Close is idempotent and safe to call multiple times.
func (p *cancelReadCloser) Close() error {
	if !p.closed.CompareAndSwap(false, true) {
		return nil
	}
	p.closeWithError(io.EOF)
	return nil
}
