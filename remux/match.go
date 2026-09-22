package remux

import (
	"net"
	"net/http"
	"regexp"
	"strings"
)

const (
	PriorityQueryRegexp = iota * 20
	PriorityQuery
	PriorityPathRegexp
	PriorityPathPrefix
	PriorityPath
	PriorityHeaderRegexp
	PriorityHeader
	PriorityHostNameRegexp
	PriorityHostname
)

type Matcher interface {
	Match(*http.Request, *MatchContext) bool
	Priority() int
	String() string
}

type MatchContext struct {
	Names  []string
	Values []string
}

func (m *MatchContext) Reset() {
	m.Names = m.Names[:0]
	m.Values = m.Values[:0]
}

// header represents a matcher for HTTP headers with a specific key-value pair.
type header struct {
	key   string
	value string
}

func (h header) Match(req *http.Request, ctx *MatchContext) bool {
	return req.Header.Get(h.key) == h.value
}

func (h header) Priority() int {
	return len(h.key) + len(h.value) + PriorityHeader
}

func (h header) String() string {
	return h.key + ": " + h.value
}

// headerRegexp represents a matcher for HTTP headers using a regular expression with named capture groups.
type headerRegexp struct {
	key   string
	re    *regexp.Regexp
	names []string
}

func (h headerRegexp) Match(req *http.Request, ctx *MatchContext) bool {
	ok, values := matchRegexpNames(h.re, req.Header.Get(h.key))
	if ok {
		addMatchesToContext(ctx, h.names, values)
	}
	return ok
}

func (h headerRegexp) Priority() int {
	return len(h.key) + len(h.re.String()) + PriorityHeaderRegexp
}

func (h headerRegexp) String() string {
	return h.key + ": " + h.re.String()
}

// hostname represents a matcher for the request's hostname.
type hostname struct {
	h string
}

func getHostname(req *http.Request) string {
	if req.URL.Hostname() != "" {
		return req.URL.Hostname()
	}
	reqHost := req.Host
	host, _, err := net.SplitHostPort(reqHost)
	if err != nil {
		return reqHost
	}
	return host
}

func (h hostname) Match(req *http.Request, ctx *MatchContext) bool {
	return getHostname(req) == h.h
}

func (h hostname) Priority() int {
	return len(h.h) + PriorityHostname
}

func (h hostname) String() string {
	return h.h
}

// hostNameRegexp represents a matcher for the request's hostname using a regular expression with named capture groups.
type hostNameRegexp struct {
	re    *regexp.Regexp
	names []string
}

func (h hostNameRegexp) Match(req *http.Request, ctx *MatchContext) bool {
	ok, values := matchRegexpNames(h.re, getHostname(req))
	if ok {
		addMatchesToContext(ctx, h.names, values)
	}
	return ok
}

func (h hostNameRegexp) Priority() int {
	return len(h.re.String()) + PriorityHostNameRegexp
}

func (h hostNameRegexp) String() string {
	return h.re.String()
}

// path represents a matcher for the request's URL path.
type path struct {
	p string
}

func (p path) Match(req *http.Request, ctx *MatchContext) bool {
	return req.URL.Path == p.p
}
func (p path) Priority() int {
	return len(p.p) + PriorityPath
}

func (p path) String() string {
	return p.p
}

// pathRegexp represents a matcher for the request's URL path using a regular expression with named capture groups.
type pathRegexp struct {
	re    *regexp.Regexp
	names []string
}

func (p pathRegexp) Match(req *http.Request, ctx *MatchContext) bool {
	ok, values := matchRegexpNames(p.re, req.URL.Path)
	if ok {
		addMatchesToContext(ctx, p.names, values)
	}
	return ok
}

func (p pathRegexp) Priority() int {
	return len(p.re.String()) + PriorityPathRegexp
}

func (p pathRegexp) String() string {
	return p.re.String()
}

// pathPrefix represents a matcher for the request's URL path prefix.
type pathPrefix struct {
	p string
}

func (p pathPrefix) Match(req *http.Request, ctx *MatchContext) bool {
	return strings.HasPrefix(req.URL.String(), p.p)
}

func (p pathPrefix) Priority() int {
	return len(p.p) + PriorityPathPrefix
}

func (p pathPrefix) String() string {
	return p.p
}

// query represents a matcher for the request's URL query parameters with a specific key-value pair.
type query struct {
	key   string
	value string
}

func (q query) Match(req *http.Request, ctx *MatchContext) bool {
	return req.URL.Query().Get(q.key) == q.value
}

func (q query) Priority() int {
	return len(q.key) + len(q.value) + PriorityQuery
}

func (q query) String() string {
	return q.key + "=" + q.value
}

// queryRegexp represents a matcher for the request's URL query parameters using a regular expression with named capture groups.
type queryRegexp struct {
	key   string
	re    *regexp.Regexp
	names []string
}

func (q queryRegexp) Match(req *http.Request, ctx *MatchContext) bool {
	ok, values := matchRegexpNames(q.re, req.URL.Query().Get(q.key))
	if ok {
		addMatchesToContext(ctx, q.names, values)
	}
	return ok
}

func (q queryRegexp) Priority() int {
	return len(q.key) + len(q.re.String()) + PriorityQueryRegexp
}

func (q queryRegexp) String() string {
	return q.key + "=" + q.re.String()
}

// matchRegexpNames matches the given string against the regular expression and returns the result along with the named capture groups.
func matchRegexpNames(re *regexp.Regexp, str string) (bool, []string) {
	submatches := re.FindStringSubmatch(str)
	if submatches == nil {
		return false, nil
	}
	return true, submatches
}

func addMatchesToContext(ctx *MatchContext, names []string, values []string) {
	ctx.Names = append(ctx.Names, names...)
	ctx.Values = append(ctx.Values, values...)
}

// orMatcher represents a logical OR combination of multiple matchers.
type orMatcher struct {
	m []Matcher
}

func (o orMatcher) Match(req *http.Request, ctx *MatchContext) bool {
	for _, m := range o.m {
		if m.Match(req, ctx) {
			return true
		}
	}
	return false
}

func (o orMatcher) Priority() int {
	p := 0
	for _, m := range o.m {
		p += m.Priority()
	}
	return p
}

func (o orMatcher) String() string {
	strs := make([]string, len(o.m))
	for i, m := range o.m {
		strs[i] = m.String()
	}
	return "OR(" + strings.Join(strs, ", ") + ")"
}

// andMatcher represents a logical AND combination of multiple matchers.
type andMatcher struct {
	m []Matcher
}

func (a andMatcher) Match(req *http.Request, ctx *MatchContext) bool {
	for _, m := range a.m {
		if !m.Match(req, ctx) {
			return false
		}
	}
	return true
}
func (a andMatcher) String() string {
	strs := make([]string, len(a.m))
	for i, m := range a.m {
		strs[i] = m.String()
	}
	return "AND(" + strings.Join(strs, ", ") + ")"
}
func (a andMatcher) Priority() int {
	p := 0
	for _, m := range a.m {
		p += m.Priority()
	}
	return p
}

// notMatcher represents a logical NOT combination of a single matcher.
type notMatcher struct {
	m Matcher
}

func (n notMatcher) Match(req *http.Request, ctx *MatchContext) bool {
	return !n.m.Match(req, ctx)
}

func (n notMatcher) Priority() int {
	return n.m.Priority()
}
func (n notMatcher) String() string {
	return "NOT(" + n.m.String() + ")"
}
