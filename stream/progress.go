package stream

import (
	"context"
	"fmt"
	"io"
	"sync"
)

// Progress represents the progress of a transfer.
type Progress struct {
	ID string

	// Progress contains a Message or...
	Message string

	// ...progress of an action
	Action  string
	Current int64
	Total   int64

	// If true, don't show xB/yB
	HideCounts bool
	// If not empty, use units instead of bytes for counts
	Units string

	// Aux contains extra information not presented to the user, such as
	// digests for push signing.
	Aux any

	LastUpdate bool
}

type ProgressWriter interface {
	io.Closer
	WriteProgress(Progress) error
}

type progressWriter struct {
	in     chan Progress
	out    chan<- Progress
	ctx    context.Context
	closed chan struct{}
	wg     sync.WaitGroup
	once   sync.Once
}

func (pw *progressWriter) WriteProgress(p Progress) error {
	select {
	case <-pw.closed:
		return io.EOF
	case <-pw.ctx.Done():
		return io.EOF
	case pw.in <- p:
		return nil
	}
}

// ChanOutput returns an Output that writes progress updates to the
// supplied channel.
func ChanOutput(progressChan chan<- Progress) ProgressWriter {
	return ChanOutputContext(context.Background(), progressChan)
}

// ChanOutputContext returns an Output that writes progress updates to the
// supplied channel and can be cancelled using ctx.
func ChanOutputContext(ctx context.Context, progressChan chan<- Progress) ProgressWriter {
	pw := &progressWriter{
		in:     make(chan Progress),
		out:    progressChan,
		ctx:    ctx,
		closed: make(chan struct{}),
	}
	pw.wg.Go(func() {
		for {
			select {
			case <-pw.ctx.Done():
				return
			case p := <-pw.in:
				if !pw.writeOut(p) {
					return
				}
			case <-pw.closed:
				for {
					select {
					case p := <-pw.in:
						if !pw.writeOut(p) {
							return
						}
					default:
						return
					}
				}
			}
		}
	})

	return pw
}

func (pw *progressWriter) writeOut(p Progress) bool {
	select {
	case <-pw.ctx.Done():
		return false
	case pw.out <- p:
		return true
	}
}

func (pw *progressWriter) Close() error {
	pw.once.Do(func() { close(pw.closed) })
	pw.wg.Wait()
	return nil
}

type discardOutput struct{}

func (discardOutput) WriteProgress(Progress) error {
	return nil
}

// DiscardOutput returns an Output that discards progress
func DiscardOutput() ProgressWriter {
	return discardOutput{}
}

func (discardOutput) Close() error {
	return nil
}

// Update is a convenience function to write a progress update to the channel.
func Update(out ProgressWriter, id, action string) {
	_ = out.WriteProgress(Progress{ID: id, Action: action})
}

// Updatef is a convenience function to write a printf-formatted progress update
// to the channel.
func Updatef(out ProgressWriter, id, format string, a ...any) {
	Update(out, id, fmt.Sprintf(format, a...))
}

// Message is a convenience function to write a progress message to the channel.
func Message(out ProgressWriter, id, message string) {
	_ = out.WriteProgress(Progress{ID: id, Message: message})
}

// Messagef is a convenience function to write a printf-formatted progress
// message to the channel.
func Messagef(out ProgressWriter, id, format string, a ...any) {
	Message(out, id, fmt.Sprintf(format, a...))
}

// Aux sends auxiliary information over a progress interface, which will not be
// formatted for the UI. This is used for things such as push signing.
func Aux(out ProgressWriter, a any) {
	_ = out.WriteProgress(Progress{Aux: a})
}
