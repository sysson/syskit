package remux

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// test path value extraction from the request URL
func TestPathValue(t *testing.T) {
	m := New()
	m.Handle(
		PathReString("/images/{id:*}/create"),
		HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte(r.PathValue("id")))
		}),
	)

	req := "/images/123/create"
	resp := httptest.NewRecorder()
	reqObj, _ := http.NewRequest("GET", req, nil)
	m.ServeHTTP(resp, reqObj)
	if resp.Body.String() != "123" {
		t.Errorf("Expected response body to be '123', got %q", resp.Body.String())
	}
}

// test wrapping middleware
func TestWrappingMiddleware(t *testing.T) {
	m := New()
	m.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			s := w.Header().Get("X-Test")
			w.Header().Set("X-Test", s+"1")
			next.ServeHTTP(w, r)
		})
	})
	m.Use(func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			s := w.Header().Get("X-Test")
			w.Header().Set("X-Test", s+"2")
			h.ServeHTTP(w, r)
		})
	})
	m.Handle(
		PathPrefix("/test"),
		HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte("ok"))
		}),

		Use(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				s := w.Header().Get("X-Test")
				w.Header().Set("X-Test", s+"3")
				next.ServeHTTP(w, r)
			})
		}),
		Use(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				s := w.Header().Get("X-Test")
				w.Header().Set("X-Test", s+"4")
				next.ServeHTTP(w, r)
			})
		}),
	)

	req := "/test"
	resp := httptest.NewRecorder()
	reqObj, _ := http.NewRequest("GET", req, nil)
	m.ServeHTTP(resp, reqObj)
	if resp.Header().Get("X-Test") != "1234" {
		t.Errorf("Expected header 'X-Test' to be '1234', got %q", resp.Header().Get("X-Test"))
	}
	if resp.Body.String() != "ok" {
		t.Errorf("Expected response body to be 'ok', got %q", resp.Body.String())
	}
}

// test priority on longer matches
func TestLongerMatchPriority(t *testing.T) {
	m := New()
	m.Handle(
		PathPrefix("/test"),
		HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte("short"))
		}),
	)
	m.Handle(
		PathPrefix("/test/long"),
		HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte("long"))
		}),
	)

	req := "/test/long"
	resp := httptest.NewRecorder()
	reqObj, _ := http.NewRequest("GET", req, nil)
	m.ServeHTTP(resp, reqObj)
	if resp.Body.String() != "long" {
		t.Errorf("Expected response body to be 'long', got %q", resp.Body.String())
	}
}

// test stripped prefix on sub mux
func TestStrippedPrefixOnSubMux(t *testing.T) {
	m := New()
	sub := m.Sub("/sub")
	sub.Handle(
		PathPrefix("/testing"),
		HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte(r.URL.Path))
		}),
	)

	req := "/sub/testing"
	resp := httptest.NewRecorder()
	reqObj, _ := http.NewRequest("GET", req, nil)
	m.ServeHTTP(resp, reqObj)
	if resp.Body.String() != "/testing" {
		t.Errorf("Expected response body to be '/testing', got %q", resp.Body.String())
	}
}

// test not found and method not allowed scenarios
func TestNotFoundAndMethodNotAllowed(t *testing.T) {
	m := New()
	m.Handle(
		Get(),
		Post(),
		PathPrefix("/test"),
		HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte("ok"))
		}),
	)

	// Not Found
	req := "/nonexistent"
	resp := httptest.NewRecorder()
	reqObj, _ := http.NewRequest("GET", req, nil)
	m.ServeHTTP(resp, reqObj)
	if resp.Code != http.StatusNotFound {
		t.Errorf("Expected status code to be 404, got %d", resp.Code)
	}

	// Method Not Allowed
	req = "/test"
	resp = httptest.NewRecorder()
	reqObj, _ = http.NewRequest("PATCH", req, nil)
	m.ServeHTTP(resp, reqObj)
	if resp.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status code to be 405, got %d", resp.Code)
	}
	if resp.Header().Get("Allow") != "GET, POST" {
		t.Errorf("Expected Allow header to be 'GET, POST', got %q", resp.Header().Get("Allow"))
	}
}

// test passing in string patterns for path, path prefix, and query
func TestStringPatterns(t *testing.T) {
	m := New()
	m.Handle(
		PathReString("^/test$"),
		HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte("path"))
		}),
	)
	m.Handle(
		PathPrefixReString("^/prefix"),
		HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte("prefix"))
		}),
	)
	m.Handle(
		QueryReString("key", "^value$"),
		HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte("query"))
		}),
	)

	// Test PathReString
	req := "/test"
	resp := httptest.NewRecorder()
	reqObj, _ := http.NewRequest("GET", req, nil)
	m.ServeHTTP(resp, reqObj)
	if resp.Body.String() != "path" {
		t.Errorf("Expected response body to be 'path', got %q", resp.Body.String())
	}

	// Test PathPrefixReString
	req = "/prefix/something"
	resp = httptest.NewRecorder()
	reqObj, _ = http.NewRequest("GET", req, nil)
	m.ServeHTTP(resp, reqObj)
	if resp.Body.String() != "prefix" {
		t.Errorf("Expected response body to be 'prefix', got %q", resp.Body.String())
	}

	// Test QueryReString
	req = "/?key=value"
	resp = httptest.NewRecorder()
	reqObj, _ = http.NewRequest("GET", req, nil)
	m.ServeHTTP(resp, reqObj)
	if resp.Body.String() != "query" {
		t.Errorf("Expected response body to be 'query', got %q", resp.Body.String())
	}
}
