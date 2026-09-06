package httpx

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type testRequest struct {
	Name string `json:"name"`
}

func (r testRequest) Validate() error {
	if r.Name == "" {
		return errors.New("name is required")
	}
	return nil
}

func TestWriteJSON(t *testing.T) {
	recorder := httptest.NewRecorder()

	if err := WriteJSON(recorder, http.StatusCreated, map[string]string{"name": "Ada"}); err != nil {
		t.Fatalf("WriteJSON() error = %v", err)
	}
	if recorder.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusCreated)
	}
	if got, want := recorder.Header().Get("Content-Type"), "application/json"; got != want {
		t.Fatalf("Content-Type = %q, want %q", got, want)
	}
	if got, want := recorder.Body.String(), `{"name":"Ada"}`; got != want {
		t.Fatalf("body = %q, want %q", got, want)
	}
}

func TestParseJSONClosesBody(t *testing.T) {
	body := &trackingReadCloser{Reader: strings.NewReader(`{"name":"Ada"}`)}
	request := httptest.NewRequest(http.MethodPost, "/", body)
	var parsed testRequest

	if err := ParseJSON(request, &parsed); err != nil {
		t.Fatalf("ParseJSON() error = %v", err)
	}
	if parsed.Name != "Ada" {
		t.Fatalf("parsed name = %q, want %q", parsed.Name, "Ada")
	}
	if !body.closed {
		t.Fatal("ParseJSON() did not close the request body")
	}
}

func TestValidatePassesDecodedRequest(t *testing.T) {
	called := false
	handler := Validate(func(w http.ResponseWriter, _ *http.Request, request testRequest) error {
		called = true
		return WriteJSON(w, http.StatusAccepted, Message(request.Name))
	})
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"name":"Ada"}`))

	if err := handler(recorder, request); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if !called || recorder.Code != http.StatusAccepted {
		t.Fatalf("handler called = %v, status = %d", called, recorder.Code)
	}
}

func TestValidateRejectsInvalidJSONAndValidation(t *testing.T) {
	tests := []struct {
		name string
		body string
		want string
	}{
		{name: "invalid json", body: "{", want: "unexpected EOF"},
		{name: "invalid request", body: `{ "name": "" }`, want: "name is required"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			handler := Validate(func(http.ResponseWriter, *http.Request, testRequest) error {
				t.Fatal("validated handler should not be called")
				return nil
			})
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(test.body))

			err := handler(recorder, request)
			var httpErr *HTTPError
			if !errors.As(err, &httpErr) {
				t.Fatalf("error = %v, want *HTTPError", err)
			}
			if httpErr.StatusCode != http.StatusBadRequest || !strings.Contains(httpErr.Error(), test.want) {
				t.Fatalf("error = %v, want bad request containing %q", httpErr, test.want)
			}
		})
	}
}

func TestKeyFromContext(t *testing.T) {
	type key string
	ctx := context.WithValue(context.Background(), key("id"), 42)

	if value, ok := KeyFromContext[key, int](ctx, key("id")); !ok || value != 42 {
		t.Fatalf("KeyFromContext() = (%d, %v), want (42, true)", value, ok)
	}
	if _, ok := KeyFromContext[key, string](ctx, key("id")); ok {
		t.Fatal("KeyFromContext() matched a value with the wrong type")
	}
}

func TestErrorLoggerMapsErrors(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		cancel     bool
		wantStatus int
		wantBody   string
	}{
		{name: "client error", err: NotFound(errors.New("missing")), wantStatus: http.StatusNotFound, wantBody: `{"message":"missing"}`},
		{name: "server error", err: errors.New("secret details"), wantStatus: http.StatusInternalServerError, wantBody: `{"message":"internal server error"}`},
		{name: "cancelled", err: errors.New("request stopped"), cancel: true, wantStatus: 499, wantBody: ""},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if test.cancel {
				cancel()
			}
			handler := ErrorLogger(func(http.ResponseWriter, *http.Request) error { return test.err })
			recorder := httptest.NewRecorder()
			request := httptest.NewRequestWithContext(ctx, http.MethodGet, "/", nil)

			handler(recorder, request)
			if recorder.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d", recorder.Code, test.wantStatus)
			}
			if test.wantBody != "" && recorder.Body.String() != test.wantBody {
				t.Fatalf("body = %q, want %q", recorder.Body.String(), test.wantBody)
			}
		})
	}
}

type trackingReadCloser struct {
	io.Reader
	closed bool
}

func (r *trackingReadCloser) Close() error {
	r.closed = true
	return nil
}
