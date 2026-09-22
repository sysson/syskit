package remux

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"
)

// test path value extraction from the request URL
func TestPathValue(t *testing.T) {
	m := New()
	m.HandleFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(r.PathValue("id")))
	},
		Path("/images/{id}/create"),
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
	m.HandleFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("ok"))
	},
		PathPrefix("/test"),
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
	m.HandleFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("short"))
	},
		PathPrefix("/test"),
	)
	m.HandleFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("long"))
	},
		PathPrefix("/test/long"),
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
	sub.HandleFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(r.URL.Path))
	},
		PathPrefix("/testing"),
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
	m.HandleFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("ok"))
	},
		Get(),
		Post(),
		PathPrefix("/test"),
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
	m.HandleFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("path"))
	},
		PathRegexp(regexp.MustCompile("^/test$")),
	)
	m.HandleFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("prefix"))
	},
		PathPrefix("/prefix"),
	)
	m.HandleFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("query"))
	},
		QueryRegexp("key", regexp.MustCompile("^value$")),
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

// test common pattern parsing using helper functions for path, path prefix, and path regexp
func TestHelperFunctions(t *testing.T) {
	m := New()
	fn := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		val := r.PathValue("id")
		_, _ = fmt.Fprintf(w, "ok:%v", val)
	})
	m.Get("/{id:*}/create", fn)
	m.Prefix("/{id}/test", fn)
	// Test the helper function route
	req := "/123/create"
	resp := httptest.NewRecorder()
	reqObj, _ := http.NewRequest("GET", req, nil)
	m.ServeHTTP(resp, reqObj)
	if resp.Body.String() != "ok:123" {
		t.Errorf("Expected response body to be 'ok:123', got %q", resp.Body.String())
	}

	req = "/456/abc/789/create"
	resp = httptest.NewRecorder()
	reqObj, _ = http.NewRequest("GET", req, nil)
	m.ServeHTTP(resp, reqObj)
	if resp.Body.String() != "ok:456/abc/789" {
		t.Errorf("Expected response body to be 'ok:456/abc/789', got %q", resp.Body.String())
	}

	// Test the regexp route
	req = "/123/test/456"
	resp = httptest.NewRecorder()
	reqObj, _ = http.NewRequest("GET", req, nil)
	m.ServeHTTP(resp, reqObj)
	if resp.Body.String() != "ok:123" {
		t.Errorf("Expected response body to be 'ok:123', got %q", resp.Body.String())
	}
}

// test header matching using the Header helper function
func TestHeader(t *testing.T) {
	m := New()
	m.Header("X-Test", "value", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("header"))
	}), Path("/images"))
	m.Get("/images", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("no header"))
	}))
	req := "/images"
	resp := httptest.NewRecorder()
	reqObj, _ := http.NewRequest("GET", req, nil)
	reqObj.Header.Set("X-Test", "value")
	m.ServeHTTP(resp, reqObj)
	if resp.Body.String() != "header" {
		t.Errorf("Expected response body to be 'header', got %q", resp.Body.String())
	}
	// Test without the header
	req = "/images"
	resp = httptest.NewRecorder()
	reqObj, _ = http.NewRequest("GET", req, nil)
	m.ServeHTTP(resp, reqObj)
	if resp.Body.String() != "no header" {
		t.Errorf("Expected response body to be 'no header', got %q", resp.Body.String())
	}
}

// test hostname matching using the HostName helper function
func TestHostName(t *testing.T) {
	m := New()
	m.HostName("example.com", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("hostname"))
	}))
	m.HostName("{subdomain}.example.com", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(r.PathValue("subdomain")))
	}))
	m.Get("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("no hostname"))
	}))
	req := "/"
	resp := httptest.NewRecorder()
	reqObj, _ := http.NewRequest("GET", req, nil)
	reqObj.Host = "example.com"
	m.ServeHTTP(resp, reqObj)
	if resp.Body.String() != "hostname" {
		t.Errorf("Expected response body to be 'hostname', got %q", resp.Body.String())
	}
	// Test without the hostname
	req = "/"
	resp = httptest.NewRecorder()
	reqObj, _ = http.NewRequest("GET", req, nil)
	reqObj.Host = "other.com"
	m.ServeHTTP(resp, reqObj)
	if resp.Body.String() != "no hostname" {
		t.Errorf("Expected response body to be 'no hostname', got %q", resp.Body.String())
	}
	// Test with a subdomain
	req = "/"
	resp = httptest.NewRecorder()
	reqObj, _ = http.NewRequest("GET", req, nil)
	reqObj.Host = "sub.example.com"
	m.ServeHTTP(resp, reqObj)
	if resp.Body.String() != "sub" {
		t.Errorf("Expected response body to be 'sub', got %q", resp.Body.String())
	}
}

// test query matching using the Query helper function
func TestQuery(t *testing.T) {
	m := New()
	m.Query("q", "value", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("query"))
	}), Path("/search"))
	m.Query("version", "v{id}", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(r.PathValue("id")))
	}), Path("/search"))
	m.Get("/search", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("no query"))
	}))
	req := "/search?q=value"
	resp := httptest.NewRecorder()
	reqObj, _ := http.NewRequest("GET", req, nil)
	m.ServeHTTP(resp, reqObj)
	if resp.Body.String() != "query" {
		t.Errorf("Expected response body to be 'query', got %q", resp.Body.String())
	}
	// Test without the query parameter
	req = "/search"
	resp = httptest.NewRecorder()
	reqObj, _ = http.NewRequest("GET", req, nil)
	m.ServeHTTP(resp, reqObj)
	if resp.Body.String() != "no query" {
		t.Errorf("Expected response body to be 'no query', got %q", resp.Body.String())
	}
	// Test with the version query parameter
	req = "/search?version=v123"
	resp = httptest.NewRecorder()
	reqObj, _ = http.NewRequest("GET", req, nil)
	m.ServeHTTP(resp, reqObj)
	if resp.Body.String() != "123" {
		t.Errorf("Expected response body to be '123', got %q", resp.Body.String())
	}
	// Test with an invalid version query parameter
	req = "/search?version=123"
	resp = httptest.NewRecorder()
	reqObj, _ = http.NewRequest("GET", req, nil)
	m.ServeHTTP(resp, reqObj)
	if resp.Body.String() != "no query" {
		t.Errorf("Expected response body to be 'no query', got %q", resp.Body.String())
	}
}

// test literal paths
func TestLiteralPaths(t *testing.T) {
	m := New()
	m.Get("/home", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("home"))
	}))
	m.Get("/about", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("about"))
	}))
	m.Get("/foo.bar", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("foo bar"))
	}))

	req := "/home"
	resp := httptest.NewRecorder()
	reqObj, _ := http.NewRequest("GET", req, nil)
	m.ServeHTTP(resp, reqObj)
	if resp.Body.String() != "home" {
		t.Errorf("Expected response body to be 'home', got %q", resp.Body.String())
	}

	req = "/about"
	resp = httptest.NewRecorder()
	reqObj, _ = http.NewRequest("GET", req, nil)
	m.ServeHTTP(resp, reqObj)
	if resp.Body.String() != "about" {
		t.Errorf("Expected response body to be 'about', got %q", resp.Body.String())
	}

	req = "/foo.bar"
	resp = httptest.NewRecorder()
	reqObj, _ = http.NewRequest("GET", req, nil)
	m.ServeHTTP(resp, reqObj)
	if resp.Body.String() != "foo bar" {
		t.Errorf("Expected response body to be 'foo bar', got %q", resp.Body.String())
	}
}

// test sub router
func TestSubRouter(t *testing.T) {
	m := New()
	sub := m.Sub("/api")
	sub.Get("/users", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("users"))
	}))

	req := "/api/users"
	resp := httptest.NewRecorder()
	reqObj, _ := http.NewRequest("GET", req, nil)
	m.ServeHTTP(resp, reqObj)
	if resp.Body.String() != "users" {
		t.Errorf("Expected response body to be 'users', got %q", resp.Body.String())
	}
}

// conflict tests

func TestConflictPaths(t *testing.T) {
	m := New()
	m.Get("/users/me", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("first"))
	}))
	m.Get("/users/{id}", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("second"))
	}))

	req := "/users/me"
	resp := httptest.NewRecorder()
	reqObj, _ := http.NewRequest("GET", req, nil)
	m.ServeHTTP(resp, reqObj)
	if resp.Body.String() != "first" {
		t.Errorf("Expected response body to be 'first', got %q", resp.Body.String())
	}

	m.Get("/assets/{id}", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("first"))
	}))
	m.Get("/assets/js/{id}", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("second"))
	}))

	req = "/assets/js/123"
	resp = httptest.NewRecorder()
	reqObj, _ = http.NewRequest("GET", req, nil)
	m.ServeHTTP(resp, reqObj)
	if resp.Body.String() != "second" {
		t.Errorf("Expected response body to be 'second', got %q", resp.Body.String())
	}
}

// test check debug output
func TestDebugOutput(t *testing.T) {
	m := New()
	m.Get("/users/{id}", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("user"))
	}))
	debugInfo := m.Debug()
	if len(debugInfo) != 1 {
		t.Errorf("Expected 1 handler in debug output, got %d", len(debugInfo))
	}
	if debugInfo[0].Methods[0] != "GET" {
		t.Errorf("Expected method to be 'GET', got %q", debugInfo[0].Methods[0])
	}
	if debugInfo[0].Matchers[0] != "^/users/(?P<id>.*)$" {
		t.Errorf("Expected matcher to be '^/users/(?P<id>.*)$', got %q", debugInfo[0].Matchers[0])
	}
}

// test find handler from request
func TestHandlerFromRequest(t *testing.T) {
	m := New()
	m.Get("/users/{id}", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("user"))
	}))
	req := "/users/123"
	reqObj, _ := http.NewRequest("GET", req, nil)
	handler := m.Handler(reqObj)
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, reqObj)
	if resp.Body.String() != "user" {
		t.Errorf("Expected response body to be 'user', got %q", resp.Body.String())
	}
}
