// Package logx provides a convenient wrapper around the standard library's log/slog logger,
// integrating context for easier distributed tracing and structured logging across goroutines.
//
// # Basic Usage
//
// Get a logger from context and log messages with structured fields:
//
//	ctx := context.Background()
//	logx.G(ctx).Info("user registered", "user_id", 42, "email", "alice@example.com")
//
// # Method Chaining
//
// Chain methods to add context fields to all subsequent log messages:
//
//	log := logx.G(ctx).
//		With("request_id", "abc-123").
//		WithGroup("auth").
//		WithError(err)
//	log.Info("login attempt failed")
//
// # Context Injection
//
// Inject a custom logger into context for use throughout your application:
//
//	customLogger := &logx.Log{log: slog.New(handler), ctx: ctx}
//	ctxWithLogger := logx.WithLogger(ctx, customLogger)
//	log := logx.G(ctxWithLogger)  // retrieves customLogger
package logx

import (
	"context"
	"log/slog"
)

// G is a shorthand convenience for the GetLogger function.
//
//	logx.G(ctx).Info("quick log", "level", "fast")
var G = GetLogger

// L is the default logger.
var L = slog.Default()

// logKey is used as a key for storing the Log in context values.
type logKey struct{}

// Log wraps slog.Logger with an associated context. This allows logging to propagate
// context information (like trace IDs) through the logger, enabling better observability
// in distributed systems.
type Log struct {
	log *slog.Logger
	ctx context.Context
}

// SetDefault sets the default slog logger L to the provided logger.
// Call this before any logging occurs to ensure the default logger is set correctly.
func SetDefault(logger *slog.Logger) {
	L = logger
}

// GetLogger retrieves a Log instance from the provided context. If a Log is stored in the context
// (via WithLogger), it returns a new Log with the same underlying logger and the provided context.
// Otherwise, it returns a Log using the default logger L with the provided context.
func GetLogger(ctx context.Context) *Log {
	if l, ok := ctx.Value(logKey{}).(*Log); ok {
		return &Log{log: l.log, ctx: ctx}
	}
	return &Log{log: L, ctx: ctx}
}

// WithLogger returns a new context with the provided Log stored as a value.
// Subsequent calls to GetLogger with this context will retrieve this logger.
// This is useful for injecting custom loggers (with specific handlers or configuration) into request contexts.
func WithLogger(ctx context.Context, logger *Log) context.Context {
	return context.WithValue(ctx, logKey{}, &Log{log: logger.log, ctx: ctx})
}

// Debug logs a message at the DEBUG level with the Log's context.
// Arguments are key-value pairs that will be included as structured fields.
func (l *Log) Debug(msg string, args ...any) {
	l.log.DebugContext(l.ctx, msg, args...)
}

// Info logs a message at the INFO level with the Log's context.
// Arguments are key-value pairs that will be included as structured fields.
func (l *Log) Info(msg string, args ...any) {
	l.log.InfoContext(l.ctx, msg, args...)
}

// Warn logs a message at the WARN level with the Log's context.
// Arguments are key-value pairs that will be included as structured fields.
func (l *Log) Warn(msg string, args ...any) {
	l.log.WarnContext(l.ctx, msg, args...)
}

// Error logs a message at the ERROR level with the Log's context.
// Arguments are key-value pairs that will be included as structured fields.
func (l *Log) Error(msg string, args ...any) {
	l.log.ErrorContext(l.ctx, msg, args...)
}

// Log logs a message at the specified level with the Log's context.
// Arguments are key-value pairs that will be included as structured fields.
func (l *Log) Log(level slog.Level, msg string, args ...any) {
	l.log.Log(l.ctx, level, msg, args...)
}

// LogAttrs logs a message at the specified level with slog.Attr objects.
// This is useful when you have pre-constructed attributes or want more control over attribute creation.
func (l *Log) LogAttrs(level slog.Level, msg string, attrs ...slog.Attr) {
	l.log.LogAttrs(l.ctx, level, msg, attrs...)
}

// WithContext returns a new Log with the same underlying logger but a different context.
// This is useful when you want to propagate a new context (e.g., a child context or a context with updated values) to the logger.
func (l *Log) WithContext(ctx context.Context) *Log {
	return &Log{log: l.log, ctx: ctx}
}

// WithError returns a new Log with an "error" field added to all subsequent log messages.
// This is a convenience method for attaching error information to logs.
func (l *Log) WithError(err error) *Log {
	return &Log{log: l.log.With("error", err), ctx: l.ctx}
}

// With returns a new Log with additional key-value pairs added as fields to all subsequent log messages.
// Arguments are key-value pairs and are passed directly to the underlying slog logger.
func (l *Log) With(args ...any) *Log {
	return &Log{log: l.log.With(args...), ctx: l.ctx}
}

// WithGroup returns a new Log where all subsequent log messages are grouped under the specified name.
// This is useful for organizing related logs and is commonly used for middleware, request handlers, or service components.
func (l *Log) WithGroup(name string) *Log {
	return &Log{log: l.log.WithGroup(name), ctx: l.ctx}
}
