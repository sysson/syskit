package remux

import (
	"net/http"
	"regexp"
	"strings"
)

// PatternMatcher defines a function type that modifies a handler based on a specific pattern or condition.
type PatternMatcher func(*handler)

// Handler returns a PatternMatcher that sets the handler for the route.
func Handler(h http.Handler) PatternMatcher {
	return func(hh *handler) {
		hh.handler = h
	}
}

// HandlerFunc returns a PatternMatcher that sets the handler function for the route.
func HandlerFunc(f func(http.ResponseWriter, *http.Request)) PatternMatcher {
	return func(hh *handler) {
		hh.handler = http.HandlerFunc(f)
	}
}

// Header returns a PatternMatcher that matches requests with the specified HTTP header key and value.
func Header(key, value string) PatternMatcher {
	return CustomMatcher(header{key: key, value: value})
}

// HeaderRegexp returns a PatternMatcher that matches requests with the specified HTTP header key and a value matching the given regular expression.
func HeaderRegexp(key string, valueRe *regexp.Regexp) PatternMatcher {
	return CustomMatcher(headerRegexp{key: key, re: valueRe, names: regexpNames(valueRe)})
}

// HeaderReString returns a PatternMatcher that matches requests with the specified HTTP header key and a value matching the given string pattern.
func HeaderReString(key string, pattern string) PatternMatcher {
	re := regexp.MustCompile(Pattern(pattern))
	return CustomMatcher(headerRegexp{key: key, re: re, names: regexpNames(re)})
}

// HostName returns a PatternMatcher that matches requests with the specified hostname.
func HostName(h string) PatternMatcher {
	return CustomMatcher(hostname{h: h})
}

// HostNameRegexp returns a PatternMatcher that matches requests with a hostname matching the given regular expression.
func HostNameRegexp(hostRe *regexp.Regexp) PatternMatcher {
	return CustomMatcher(hostNameRegexp{re: hostRe, names: regexpNames(hostRe)})
}

func HostNameReString(pattern string) PatternMatcher {
	re := regexp.MustCompile(Pattern(pattern))
	return CustomMatcher(hostNameRegexp{re: re, names: regexpNames(re)})
}

// Method returns a PatternMatcher that matches requests with the specified HTTP method.
func Methods(m ...HTTPMethod) PatternMatcher {
	return func(h *handler) {
		var prio int
		for _, method := range m {
			h.method = append(h.method, method)
			prio += len(method)
		}
		h.matchPrio += prio
	}
}

// Get returns a PatternMatcher that matches requests with the GET HTTP method.
func Get() PatternMatcher {
	return Methods(MethodGet)
}

// Post returns a PatternMatcher that matches requests with the POST HTTP method.
func Post() PatternMatcher {
	return Methods(MethodPost)
}

// Put returns a PatternMatcher that matches requests with the PUT HTTP method.
func Put() PatternMatcher {
	return Methods(MethodPut)
}

// Delete returns a PatternMatcher that matches requests with the DELETE HTTP method.
func Delete() PatternMatcher {
	return Methods(MethodDelete)
}

// Patch returns a PatternMatcher that matches requests with the PATCH HTTP method.
func Patch() PatternMatcher {
	return Methods(MethodPatch)
}

// Options returns a PatternMatcher that matches requests with the OPTIONS HTTP method.
func Options() PatternMatcher {
	return Methods(MethodOptions)
}

// Head returns a PatternMatcher that matches requests with the HEAD HTTP method.
func Head() PatternMatcher {
	return Methods(MethodHead)
}

// Path returns a PatternMatcher that matches requests with the specified URL path.
func Path(p string) PatternMatcher {
	return CustomMatcher(path{p: p})
}

// PathRegexp returns a PatternMatcher that matches requests with a URL path matching the given regular expression.
func PathRegexp(pathRe *regexp.Regexp) PatternMatcher {
	return CustomMatcher(pathRegexp{re: pathRe, names: regexpNames(pathRe)})
}

// PathReString returns a PatternMatcher that matches requests with a URL path matching the given string pattern.
func PathReString(pattern string) PatternMatcher {
	re := regexp.MustCompile(Pattern(pattern))
	return CustomMatcher(pathRegexp{re: re, names: regexpNames(re)})
}

// PathPrefix returns a PatternMatcher that matches requests with the specified URL path prefix.
func PathPrefix(p string) PatternMatcher {
	return CustomMatcher(pathPrefix{p: p})
}

// PathPrefixRegexp returns a PatternMatcher that matches requests with a URL path prefix matching the given regular expression.
func PathPrefixRegexp(pathPrefixRe *regexp.Regexp) PatternMatcher {
	return CustomMatcher(pathPrefixRegexp{re: pathPrefixRe, names: regexpNames(pathPrefixRe)})
}

// PathPrefixReString returns a PatternMatcher that matches requests with a URL path prefix matching the given string pattern. It removes the ending $ if present.
func PathPrefixReString(pattern string) PatternMatcher {
	re := regexp.MustCompile(strings.TrimSuffix(Pattern(pattern),"$"))
	return CustomMatcher(pathPrefixRegexp{re: re, names: regexpNames(re)})
}

// Query returns a PatternMatcher that matches requests with the specified query parameter key and value.
func Query(key, value string) PatternMatcher {
	return CustomMatcher(query{key: key, value: value})
}

// QueryRegexp returns a PatternMatcher that matches requests with the specified query parameter key and a value matching the given regular expression.
func QueryRegexp(key string, valueRe *regexp.Regexp) PatternMatcher {
	return CustomMatcher(queryRegexp{key: key, re: valueRe, names: regexpNames(valueRe)})
}

// QueryReString returns a PatternMatcher that matches requests with the specified query parameter key and a value matching the given string pattern.
func QueryReString(key, pattern string) PatternMatcher {
	re := regexp.MustCompile(Pattern(pattern))
	return CustomMatcher(queryRegexp{key: key, re: re, names: regexpNames(re)})
}

// Priority returns a PatternMatcher that sets the priority of the handler.
func Priority(p int) PatternMatcher {
	return func(h *handler) {
		h.setPrio = p
	}
}

// Use returns a PatternMatcher that adds middleware to the handler.
func Use(mw ...func(http.Handler) http.Handler) PatternMatcher {
	return func(h *handler) {
		h.mw = append(h.mw, mw...)
	}
}

// regexpNames extracts the names of the capturing groups from the given regular expression. It returns a boolean indicating whether the regular expression has named capturing groups and a slice of the group names.
func regexpNames(re *regexp.Regexp) []string {
	return re.SubexpNames()
}

// CustomMatcher returns a PatternMatcher that adds a custom matcher to the handler.
func CustomMatcher(m Matcher) PatternMatcher {
	return func(h *handler) {
		h.matchers = append(h.matchers, m)
		h.matchPrio += m.Priority()
	}
}

// Or returns a PatternMatcher that combines multiple matchers with a logical OR.
func Or(m ...Matcher) PatternMatcher {
	return CustomMatcher(orMatcher{m: m})
}

// And returns a PatternMatcher that combines multiple matchers with a logical AND.
func And(m ...Matcher) PatternMatcher {
	return CustomMatcher(andMatcher{m: m})
}

// Not returns a PatternMatcher that negates the given matcher with a logical NOT.
func Not(m Matcher) PatternMatcher {
	return CustomMatcher(notMatcher{m: m})
}
