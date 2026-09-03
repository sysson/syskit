# syskit

A collection of small, composable Go utility packages for building robust backend systems. Each package is focused on solving specific problems with minimal dependencies and maximum ease of use.

## Packages

### [`logx`](logx/) — Context-Aware Structured Logging

Wrapper around Go's `log/slog` that integrates with `context.Context` for distributed tracing and structured logging.

**Features:**
- Context propagation through logger chains
- Fluent API for adding fields and grouping logs
- Compatible with standard `slog` handlers
- Convenience shorthand `G(ctx)` for quick access

**Example:**
```go
package main

import (
	"context"
	"github.com/sysson/syskit/logx"
)

func main() {
	ctx := context.Background()
	
	// Get logger from context
	log := logx.GetLogger(ctx)
	
	// Log with structured fields
	log.Info("user registered", "user_id", 42, "email", "alice@example.com")
	
	// Chain methods to add context
	log.
		With("request_id", "abc-123").
		WithGroup("auth").
		Info("authentication successful", "user", "alice")
	
	// Shorthand syntax
	logx.G(ctx).Error("operation failed", "reason", "timeout")
}
```

---

### [`httpx`](httpx/) — HTTP Request/Response Utilities

Utilities for handling HTTP requests and responses consistently, including validation, error handling, and JSON parsing.

**Features:**
- Generic validation middleware with automatic JSON parsing
- Standardized HTTP error responses
- Type-safe JSON reading/writing
- Consistent error logging with `ErrorLogger` middleware
- Client cancellation detection

**Example:**
```go
package main

import (
	"errors"
	"net/http"
	"github.com/sysson/syskit/httpx"
	"github.com/sysson/syskit/logx"
)

// Define a request type implementing httpx.Validator
type CreateUserRequest struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

func (r CreateUserRequest) Validate() error {
	if r.Name == "" {
		return errors.New("name is required")
	}
	if r.Email == "" {
		return errors.New("email is required")
	}
	return nil
}

// Handler receives validated data
func createUser(w http.ResponseWriter, r *http.Request, req CreateUserRequest) error {
	logx.G(r.Context()).Info("creating user", "name", req.Name, "email", req.Email)
	
	return httpx.WriteJSON(w, http.StatusCreated, map[string]string{
		"id":    "user-123",
		"email": req.Email,
	})
}

func main() {
	// Wire up the handler with validation middleware and error logging
	handler := httpx.ErrorLogger(httpx.ValidateMiddleware[CreateUserRequest](createUser))
	
	http.HandleFunc("POST /users", handler)
	http.ListenAndServe(":8080", nil)
	
	// Request:  POST /users
	// Body:     {"name": "Alice", "email": "alice@example.com"}
	// Response: 201 Created {"id": "user-123", "email": "alice@example.com"}
	
	// Invalid request:
	// Body:     {"name": "", "email": "alice@example.com"}
	// Response: 400 Bad Request {"message": "name is required"}
}
```

**Key Functions:**
- `ValidateMiddleware[V Validator](handler) HTTPErrorFunc` — Auto-parse and validate JSON requests
- `ErrorLogger(handler HTTPErrorFunc) http.HandlerFunc` — Middleware for consistent error handling and logging
- `WriteJSON(w, statusCode, v) error` — Write JSON responses
- `ParseJSON(r, v) error` — Parse JSON request bodies
- `KeyFromContext[T, V](ctx, key) (V, bool)` — Type-safe context value retrieval

---

### [`iox`](iox/) — I/O Utilities

Utilities for wrapping and modifying I/O streams, particularly useful for HTTP response interception and cancellable I/O.

**Features:**
- Wrapping readers with custom cleanup functions
- Context-aware cancellable readers
- HTTP response interception and buffering (up to 64KB)
- Response header and body modification
- Support for connection hijacking

**Example:**
```go
package main

import (
	"context"
	"io"
	"net/http"
	"time"
	"github.com/sysson/syskit/iox"
)

// Wrap a reader with custom cleanup
func readWithCleanup(r io.Reader, cleanup func() error) io.ReadCloser {
	return iox.NewReadCloserWrapper(r, cleanup)
}

// Add cancellation to a reader based on context
func readWithTimeout(ctx context.Context, r io.ReadCloser) io.ReadCloser {
	timeoutCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	return iox.NewCancelReadCloser(timeoutCtx, r)
}

// Intercept and modify HTTP responses
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Wrap the response writer to capture output
		rm := iox.NewResponseModifier(w)
		
		// Call the handler
		next.ServeHTTP(rm, r)
		
		// Now we can inspect and modify
		statusCode := rm.StatusCode()
		body := rm.RawBody()
		
		// Log response details
		println("Response Status:", statusCode)
		println("Response Body Size:", len(body))
		
		// Flush to client
		rm.FlushAll()
	})
}
```

---

## Installation

```bash
go get github.com/sysson/syskit
```

## License

See [LICENSE](LICENSE) file.
