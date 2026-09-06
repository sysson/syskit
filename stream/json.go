package stream

import "encoding/json"

// Message defines a message struct. It describes
// the created time, where it from, status, ID of the
// message.
type StreamMessage struct {
	Stream   string           `json:"stream,omitempty"`
	Status   string           `json:"status,omitempty"`
	Progress *StreamProgress  `json:"progressDetail,omitempty"`
	ID       string           `json:"id,omitempty"`
	Error    *StreamError     `json:"errorDetail,omitempty"`
	Aux      *json.RawMessage `json:"aux,omitempty"` // Aux contains out-of-band data.
}

// Progress describes a progress message in a JSON stream.
type StreamProgress struct {
	Current    int64  `json:"current,omitempty"`    // Current is the current status and value of the progress made towards Total.
	Total      int64  `json:"total,omitempty"`      // Total is the end value describing when we made 100% progress for an operation.
	Start      int64  `json:"start,omitempty"`      // Start is the initial value for the operation.
	HideCounts bool   `json:"hidecounts,omitempty"` // HideCounts. if true, hides the progress count indicator (xB/yB).
	Units      string `json:"units,omitempty"`      // Units is the unit to print for progress. It defaults to "bytes" if empty.
}

// Error wraps a concrete Code and Message, Code is
// an integer error code, Message is the error message.
type StreamError struct {
	Code    int    `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
}

func (e *StreamError) Error() string {
	if e == nil {
		return "<nil>"
	}
	return e.Message
}
