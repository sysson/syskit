package remux

import (
	"net/http"
	"regexp"
)

// HandlerOptions defines a function type that modifies a handler based on a specific pattern or condition.
type HandlerOptions func(*handler)

// Header returns a HandlerOptions that matches requests with the specified HTTP header key and value.
func Header(key, value string) HandlerOptions {
	return CustomMatcher(header{key: key, value: value})
}

// HeaderRegexp returns a HandlerOptions that matches requests with the specified HTTP header key and a value matching the given regular expression.
func HeaderRegexp(key string, valueRe *regexp.Regexp) HandlerOptions {
	return CustomMatcher(headerRegexp{key: key, re: valueRe, names: regexpNames(valueRe)})
}

// HostName returns a HandlerOptions that matches requests with the specified hostname.
func HostName(h string) HandlerOptions {
	return CustomMatcher(hostname{h: h})
}

// HostNameRegexp returns a HandlerOptions that matches requests with a hostname matching the given regular expression.
func HostNameRegexp(hostRe *regexp.Regexp) HandlerOptions {
	return CustomMatcher(hostNameRegexp{re: hostRe, names: regexpNames(hostRe)})
}

// Method returns a HandlerOptions that matches requests with the specified HTTP method.
func Methods(m ...string) HandlerOptions {
	return func(h *handler) {
		h.method = append(h.method, m...)
		h.matchPrio += len(m)
	}
}

// Get returns a HandlerOptions that matches requests with the GET HTTP method.
func Get() HandlerOptions {
	return Methods(MethodGet)
}

// Query returns a HandlerOptions that matches requests with the QUERY HTTP method.
func QueryMethod() HandlerOptions {
	return Methods(MethodQuery)
}

// Post returns a HandlerOptions that matches requests with the POST HTTP method.
func Post() HandlerOptions {
	return Methods(MethodPost)
}

// Put returns a HandlerOptions that matches requests with the PUT HTTP method.
func Put() HandlerOptions {
	return Methods(MethodPut)
}

// Delete returns a HandlerOptions that matches requests with the DELETE HTTP method.
func Delete() HandlerOptions {
	return Methods(MethodDelete)
}

// Patch returns a HandlerOptions that matches requests with the PATCH HTTP method.
func Patch() HandlerOptions {
	return Methods(MethodPatch)
}

// Options returns a HandlerOptions that matches requests with the OPTIONS HTTP method.
func Options() HandlerOptions {
	return Methods(MethodOptions)
}

// Head returns a HandlerOptions that matches requests with the HEAD HTTP method.
func Head() HandlerOptions {
	return Methods(MethodHead)
}

// Connect returns a HandlerOptions that matches requests with the CONNECT HTTP method.
func Connect() HandlerOptions {
	return Methods(MethodConnect)
}

// Trace returns a HandlerOptions that matches requests with the TRACE HTTP method.
func Trace() HandlerOptions {
	return Methods(MethodTrace)
}

// Path returns a HandlerOptions that matches requests with the specified URL path.
func Path(p string) HandlerOptions {
	return CustomMatcher(path{p: p})
}

// PathRegexp returns a HandlerOptions that matches requests with a URL path matching the given regular expression.
func PathRegexp(pathRe *regexp.Regexp) HandlerOptions {
	return CustomMatcher(pathRegexp{re: pathRe, names: regexpNames(pathRe)})
}

// PathPrefix returns a HandlerOptions that matches requests with the specified URL path prefix.
func PathPrefix(p string) HandlerOptions {
	return CustomMatcher(pathPrefix{p: p})
}

// Query returns a HandlerOptions that matches requests with the specified query parameter key and value.
func Query(key, value string) HandlerOptions {
	return CustomMatcher(query{key: key, value: value})
}

// QueryRegexp returns a HandlerOptions that matches requests with the specified query parameter key and a value matching the given regular expression.
func QueryRegexp(key string, valueRe *regexp.Regexp) HandlerOptions {
	return CustomMatcher(queryRegexp{key: key, re: valueRe, names: regexpNames(valueRe)})
}

// Priority returns a HandlerOptions that sets the priority of the handler.
func Priority(p int) HandlerOptions {
	return func(h *handler) {
		h.setPrio = p
	}
}

// Use returns a HandlerOptions that adds middleware to the handler.
func Use(mw ...func(http.Handler) http.Handler) HandlerOptions {
	return func(h *handler) {
		h.mw = append(h.mw, mw...)
	}
}

// regexpNames extracts the names of the capturing groups from the given regular expression. It returns a boolean indicating whether the regular expression has named capturing groups and a slice of the group names.
func regexpNames(re *regexp.Regexp) []string {
	return re.SubexpNames()
}

// CustomMatcher returns a HandlerOptions that adds a custom matcher to the handler.
func CustomMatcher(m Matcher) HandlerOptions {
	return func(h *handler) {
		h.matchers = append(h.matchers, m)
		h.matchPrio += m.Priority()
	}
}

// Or returns a HandlerOptions that combines multiple matchers with a logical OR.
func Or(m ...Matcher) HandlerOptions {
	return CustomMatcher(orMatcher{m: m})
}

// And returns a HandlerOptions that combines multiple matchers with a logical AND.
func And(m ...Matcher) HandlerOptions {
	return CustomMatcher(andMatcher{m: m})
}

// Not returns a HandlerOptions that negates the given matcher with a logical NOT.
func Not(m Matcher) HandlerOptions {
	return CustomMatcher(notMatcher{m: m})
}
