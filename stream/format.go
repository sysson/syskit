package stream

import (
	"encoding/json"
	"fmt"
	"io"
	"sync"
)

const streamNewline = "\r\n"

type jsonProgressFormatter struct{}

func appendNewline(source []byte) []byte {
	return append(source, []byte(streamNewline)...)
}

// FormatStatus formats the specified objects according to the specified format (and id).
func FormatStatus(id, format string, a ...any) []byte {
	str := fmt.Sprintf(format, a...)
	b, err := json.Marshal(&StreamMessage{ID: id, Status: str})
	if err != nil {
		return FormatError(err)
	}
	return appendNewline(b)
}

// FormatError formats the error as a JSON object
func FormatError(err error) []byte {
	jsonError, ok := err.(*StreamError)
	if !ok {
		jsonError = &StreamError{Message: err.Error()}
	}
	if b, err := json.Marshal(&StreamMessage{Error: jsonError}); err == nil {
		return appendNewline(b)
	}
	return []byte(`{"error":"format error"}` + streamNewline)
}

func (sf *jsonProgressFormatter) formatStatus(id, format string, a ...any) []byte {
	return FormatStatus(id, format, a...)
}

// formatProgress formats the progress information for a specified action.
func (sf *jsonProgressFormatter) formatProgress(id, action string, progress *StreamProgress, aux any) []byte {
	if progress == nil {
		progress = &StreamProgress{}
	}
	var auxJSON *json.RawMessage
	if aux != nil {
		auxJSONBytes, err := json.Marshal(aux)
		if err != nil {
			return nil
		}
		auxJSON = new(json.RawMessage)
		*auxJSON = auxJSONBytes
	}
	b, err := json.Marshal(&StreamMessage{
		Status:   action,
		Progress: progress,
		ID:       id,
		Aux:      auxJSON,
	})
	if err != nil {
		return nil
	}
	return appendNewline(b)
}

// NewJSONProgressOutput returns a progress.Output that formats output
// using JSON objects
func NewJSONProgressOutput(out io.Writer, newLines bool) ProgressWriter {
	return &progressOutput{sf: &jsonProgressFormatter{}, out: out, newLines: newLines}
}

type formatProgress interface {
	formatStatus(id, format string, a ...any) []byte
	formatProgress(id, action string, progress *StreamProgress, aux any) []byte
}

type progressOutput struct {
	sf       formatProgress
	out      io.Writer
	newLines bool
	mu       sync.Mutex
}

// WriteProgress formats progress information from a ProgressReader.
func (out *progressOutput) WriteProgress(prog Progress) error {
	var formatted []byte
	if prog.Message != "" {
		formatted = out.sf.formatStatus(prog.ID, "%s", prog.Message)
	} else {
		jsonProgress := StreamProgress{
			Current:    prog.Current,
			Total:      prog.Total,
			HideCounts: prog.HideCounts,
			Units:      prog.Units,
		}
		formatted = out.sf.formatProgress(prog.ID, prog.Action, &jsonProgress, prog.Aux)
	}

	out.mu.Lock()
	defer out.mu.Unlock()
	_, err := out.out.Write(formatted)
	if err != nil {
		return err
	}

	if out.newLines && prog.LastUpdate {
		_, err = out.out.Write(out.sf.formatStatus("", ""))
		return err
	}

	return nil
}

func (out *progressOutput) Close() error {
	if closer, ok := out.out.(io.Closer); ok {
		return closer.Close()
	}
	return nil
}

// AuxFormatter is a streamFormatter that writes aux progress messages
type AuxFormatter struct {
	io.Writer
}

// Emit emits the given interface as an aux progress message
func (sf *AuxFormatter) Emit(id string, aux any) error {
	auxJSONBytes, err := json.Marshal(aux)
	if err != nil {
		return err
	}
	auxJSON := new(json.RawMessage)
	*auxJSON = auxJSONBytes
	msgJSON, err := json.Marshal(&StreamMessage{ID: id, Aux: auxJSON})
	if err != nil {
		return err
	}
	msgJSON = appendNewline(msgJSON)
	n, err := sf.Write(msgJSON)
	if n != len(msgJSON) {
		return io.ErrShortWrite
	}
	return err
}
