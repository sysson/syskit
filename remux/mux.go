package remux

import (
	"net/http"
	"slices"
	"sort"
	"strings"
)

type HTTPMethod string

const (
	MethodGet     HTTPMethod = "GET"
	MethodPost    HTTPMethod = "POST"
	MethodPut     HTTPMethod = "PUT"
	MethodDelete  HTTPMethod = "DELETE"
	MethodPatch   HTTPMethod = "PATCH"
	MethodOptions HTTPMethod = "OPTIONS"
	MethodHead    HTTPMethod = "HEAD"
)

type handler struct {
	matchers  []Matcher
	matchPrio int
	setPrio   int
	handler   http.Handler
	method    []HTTPMethod
	mw        []func(http.Handler) http.Handler
}

// match checks if the request matches all the matchers of the handler.
func (h *handler) match(r *http.Request, ctx *MatchContext) bool {
	for _, matcher := range h.matchers {
		if !matcher.Match(r, ctx) {
			return false
		}
	}
	return true
}

// checks if the given HTTP method is allowed for the handler.
// Returns a boolean indicating if the method is allowed and a list of allowed methods.
func (h *handler) methodAllowed(method string) (bool, []string) {
	if len(h.method) == 0 {
		return true, nil
	}
	method = strings.ToUpper(method)
	allowed := make([]string, len(h.method))
	for i, m := range h.method {
		if string(m) == method {
			return true, allowed
		}
		allowed[i] = string(m)
	}
	return false, allowed
}

// priority returns the effective priority of the handler, considering both the set priority and the match priority.
func (h *handler) priority() int {
	if h.setPrio != 0 {
		return h.setPrio
	}
	return h.matchPrio
}

type ReMux struct {
	handlers         []*handler
	mw               []func(http.Handler) http.Handler
	notFound         http.HandlerFunc
	methodNotAllowed http.HandlerFunc
}

// sortKeys sorts the handlers in the ReMux based on their priority in descending order.
func (m *ReMux) sortKeys() {
	sort.Slice(m.handlers, func(i, j int) bool {
		return m.handlers[i].priority() > m.handlers[j].priority()
	})
}

// New creates and returns a new instance of ReMux with default settings.
func New() *ReMux {
	return &ReMux{
		handlers:         []*handler{},
		mw:               []func(http.Handler) http.Handler{},
		notFound:         notFoundHandler(),
		methodNotAllowed: methodNotAllowedHandler(),
	}
}

// Handle registers a new handler with the given pattern matchers and optional middleware. It appends the handler to the list of handlers and sorts them based on priority.
func (m *ReMux) Handle(opts ...PatternMatcher) {
	hh := &handler{
		matchers: []Matcher{},
		mw:       []func(http.Handler) http.Handler{},
		setPrio:  0,
		handler:  notFoundHandler(),
	}
	for _, opt := range opts {
		opt(hh)
	}
	m.handlers = append(m.handlers, hh)
	m.sortKeys()
}

// NotFound sets the handler to be called when no matching route is found. If nil is passed, it resets to the default not found handler.
func (m *ReMux) NotFound(h http.HandlerFunc) {
	if h == nil {
		h = notFoundHandler()
	}
	m.notFound = h
}

// MethodNotAllowed sets the handler to be called when a route matches but the HTTP method is not allowed. If nil is passed, it resets to the default method not allowed handler.
func (m *ReMux) MethodNotAllowed(h http.HandlerFunc) {
	if h == nil {
		h = methodNotAllowedHandler()
	}
	m.methodNotAllowed = h
}

// ServeHTTP dispatches the request to the handler whose pattern most closely matches the request URL. It applies the middleware in the correct order and handles not found and method not allowed cases.
func (m *ReMux) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h, matches, allowedMethods := m.findMatch(r)
	if h == nil {
		if len(allowedMethods) == 0 {
			m.notFound.ServeHTTP(w, r)
		} else {
			addAllowedMethodsHeader(w, allowedMethods)
			m.methodNotAllowed.ServeHTTP(w, r)
		}
		return
	}

	addPathValues(r, matches)

	wrapHandler(
		wrapHandler(
			h.handler,
			h.mw...,
		),
		m.mw...,
	).ServeHTTP(w, r)

}

// Use adds middleware to the ReMux.
func (m *ReMux) Use(mw ...func(http.Handler) http.Handler) {
	m.mw = append(m.mw, mw...)
}

// Sub creates a sub-router with the given prefix.
// The sub-router inherits the NotFound and MethodNotAllowed handlers from the parent router.
// Requests to the sub-router will have the prefix stripped before being passed to its handlers.
func (m *ReMux) Sub(prefix string) *ReMux {
	subMux := New()
	subMux.notFound = m.notFound
	subMux.methodNotAllowed = m.methodNotAllowed

	fn := func(h http.Handler) http.Handler {
		return http.StripPrefix(prefix, h)
	}

	m.Handle(
		Handler(subMux),
		PathPrefix(prefix+"/"),
		Use(fn),
	)
	return subMux
}

// addPathValues adds the captured named groups from the regular expression match to the request's path values.
func addPathValues(r *http.Request, captures *MatchContext) {
	if captures == nil {
		return
	}
	if len(captures.Names) != len(captures.Values) {
		return
	}
	for i, name := range captures.Names {
		if name != "" {
			r.SetPathValue(name, captures.Values[i])
		}
	}
}

// addAllowedMethodsHeader sets the Allow header with the allowed HTTP methods.
func addAllowedMethodsHeader(w http.ResponseWriter, allowedMethods []string) {
	w.Header().Set("Allow", strings.Join(allowedMethods, ", "))
}

// findMatch searches for the best matching handler for the given request.
// It returns a boolean indicating if a route was found but the method was not allowed,
// the matched handler, and the captured named groups from the regular expression match.
func (m *ReMux) findMatch(r *http.Request) (*handler, *MatchContext, []string) {
	var allowedMethods []string
	var matches MatchContext
	for _, h := range m.handlers {
		matches.Reset()
		if h.match(r, &matches) {
			ok, allowed := h.methodAllowed(r.Method)
			if ok {
				return h, &matches, allowedMethods
			}
			allowedMethods = append(allowedMethods, allowed...)
			if len(h.method) == 0 {
				return h, &matches, allowedMethods
			}
		}
	}
	return nil, nil, allowedMethods
}

// wrapHandler applies the given middleware to the handler in the correct order.
func wrapHandler(h http.Handler, mw ...func(http.Handler) http.Handler) http.Handler {
	for _, mwFunc := range slices.Backward(mw) {
		h = mwFunc(h)
	}
	return h
}

// default not found handler that returns a 404 status code.
func notFoundHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}
}

// default method not allowed handler that returns a 405 status code.
func methodNotAllowedHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
	}
}
