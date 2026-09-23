package remux

import (
	"net/http"
	"slices"
	"sort"
	"strings"
)

const (
	MethodGet     string = "GET"
	MethodPost    string = "POST"
	MethodPut     string = "PUT"
	MethodDelete  string = "DELETE"
	MethodPatch   string = "PATCH"
	MethodOptions string = "OPTIONS"
	MethodHead    string = "HEAD"
	MethodConnect string = "CONNECT"
	MethodTrace   string = "TRACE"
	MethodQuery   string = "QUERY"
)

type handler struct {
	matchers  []Matcher
	matchPrio int
	setPrio   int
	index     int
	handler   http.Handler
	method    []string
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
		if m == method {
			return true, allowed
		}
		allowed[i] = m
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
		// If priorities are equal, sort by index to maintain insertion order.
		if m.handlers[i].priority() == m.handlers[j].priority() {
			return m.handlers[i].index < m.handlers[j].index
		}
		// Otherwise, sort by priority in descending order.
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
func (m *ReMux) Handle(h http.Handler, opts ...HandlerOptions) {
	hh := &handler{
		matchers: []Matcher{},
		mw:       []func(http.Handler) http.Handler{},
		setPrio:  0,
		handler:  h,
		index:    len(m.handlers),
	}
	for _, opt := range opts {
		opt(hh)
	}
	m.handlers = append(m.handlers, hh)
	m.sortKeys()
}

// Handle registers a new handler with the given pattern matchers and optional middleware. It appends the handler to the list of handlers and sorts them based on priority.
func (m *ReMux) HandleFunc(hf http.HandlerFunc, opts ...HandlerOptions) {
	m.Handle(http.HandlerFunc(hf), opts...)
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

			wrapHandler(
				m.notFound,
				m.mw...,
			).ServeHTTP(w, r)

		} else {

			addAllowedMethodsHeader(w, allowedMethods)
			wrapHandler(
				m.methodNotAllowed,
				m.mw...,
			).ServeHTTP(w, r)

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
		subMux,
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
	var allowedMethods = make(map[string]struct{})
	var matches MatchContext
	for _, h := range m.handlers {
		matches.Reset()
		if h.match(r, &matches) {
			// If the handler has no specific methods, it matches all methods.
			if len(h.method) == 0 {
				return h, &matches, mapKeys(allowedMethods)
			}
			// Check if the method is allowed for this handler.
			ok, allowed := h.methodAllowed(r.Method)
			if ok {
				return h, &matches, mapKeys(allowedMethods)
			}
			// If the method is not allowed, add it to the allowed methods set.
			for _, method := range allowed {
				allowedMethods[method] = struct{}{}
			}
		}
	}
	allowed := mapKeys(allowedMethods)
	sort.Strings(allowed)
	return nil, nil, allowed
}

func mapKeys(m map[string]struct{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
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

// handle is a helper method that sets up the route with the specified HTTP methods and path pattern.
func (m *ReMux) Method(method string, path string, handler http.Handler, opts ...HandlerOptions) {
	opts = append(opts, Path(path), Methods(method))
	m.Handle(handler, opts...)
}

// Get is a helper method that sets up a route for the GET HTTP method.
func (m *ReMux) Get(path string, handler http.Handler, opts ...HandlerOptions) {
	m.Method("GET", path, handler, opts...)
}

// Prefix is a helper method that sets up a route for any HTTP method with the specified path prefix.
func (m *ReMux) Prefix(path string, handler http.Handler, opts ...HandlerOptions) {
	opts = append(opts, PathPrefix(path))
	m.Handle(handler, opts...)
}

// Post is a helper method that sets up a route for the POST HTTP method.
func (m *ReMux) Post(path string, handler http.Handler, opts ...HandlerOptions) {
	m.Method("POST", path, handler, opts...)
}

// Put is a helper method that sets up a route for the PUT HTTP method.
func (m *ReMux) Put(path string, handler http.Handler, opts ...HandlerOptions) {
	m.Method("PUT", path, handler, opts...)
}

// Delete is a helper method that sets up a route for the DELETE HTTP method.
func (m *ReMux) Delete(path string, handler http.Handler, opts ...HandlerOptions) {
	m.Method("DELETE", path, handler, opts...)
}

// Patch is a helper method that sets up a route for the PATCH HTTP method.
func (m *ReMux) Patch(path string, handler http.Handler, opts ...HandlerOptions) {
	m.Method("PATCH", path, handler, opts...)
}

// Options is a helper method that sets up a route for the OPTIONS HTTP method.
func (m *ReMux) Options(path string, handler http.Handler, opts ...HandlerOptions) {
	m.Method("OPTIONS", path, handler, opts...)
}

// Head is a helper method that sets up a route for the HEAD HTTP method.
func (m *ReMux) Head(path string, handler http.Handler, opts ...HandlerOptions) {
	m.Method("HEAD", path, handler, opts...)
}

// Connect is a helper method that sets up a route for the CONNECT HTTP method.
func (m *ReMux) Connect(path string, handler http.Handler, opts ...HandlerOptions) {
	m.Method("CONNECT", path, handler, opts...)
}

// Trace is a helper method that sets up a route for the TRACE HTTP method.
func (m *ReMux) Trace(path string, handler http.Handler, opts ...HandlerOptions) {
	m.Method("TRACE", path, handler, opts...)
}

// Header is a helper method that sets up a route for the HEADER HTTP method.
func (m *ReMux) Header(k, v string, handler http.Handler, opts ...HandlerOptions) {
	opts = append(opts, Header(k, v))
	m.Handle(handler, opts...)
}

func (m *ReMux) Query(k, v string, handler http.Handler, opts ...HandlerOptions) {
	opts = append(opts, Query(k, v))
	m.Handle(handler, opts...)
}

func (m *ReMux) HostName(v string, handler http.Handler, opts ...HandlerOptions) {
	opts = append(opts, HostName(v))
	m.Handle(handler, opts...)
}

// Handler returns the http.Handler that matches the given request.
// the request is not modified by this method.
func (m *ReMux) Handler(r *http.Request) http.Handler {
	h, _, _ := m.findMatch(r)
	return h.handler
}

type HandlerInfo struct {
	Priority int
	Index    int
	Handler  http.Handler
	Matchers []string
	Methods  []string
}

func (m *ReMux) Debug() []HandlerInfo {
	infos := []HandlerInfo{}
	for _, h := range m.handlers {
		var matchStr []string
		for _, m := range h.matchers {
			matchStr = append(matchStr, m.String())
		}
		methods := append([]string{}, h.method...)
		infos = append(infos, HandlerInfo{
			Priority: h.priority(),
			Index:    h.index,
			Handler:  h.handler,
			Matchers: matchStr,
			Methods:  methods,
		})
	}
	return infos
}
