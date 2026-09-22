package remux

import (
	"regexp"
	"testing"
)

// tdd test for Pattern function
func TestPatternMatch(t *testing.T) {
	type testCase struct {
		pattern     string
		patternType patternType
		input       string
		prefix      bool
		output      string
		match       bool
		named       map[string]string
	}

	tests := []testCase{
		{pattern: "/images/{id:*}/create",
			patternType: patternPath,
			output:      "^/images/(?P<id>.*)/create$",
			input:       "/images/123/create",
			prefix:      false,
			match:       true, named: map[string]string{"id": "123"},
		},
		{pattern: "/images/{id:*}/create",
			patternType: patternPath,
			output:      "^/images/(?P<id>.*)/create$",
			input:       "/images/123/delete",
			prefix:      false,
			match:       false,
		},
		{pattern: "/images/{prefix:*}/{id:*}/create",
			patternType: patternPath,
			output:      "^/images/(?P<prefix>.*)/(?P<id>.*)/create$",
			input:       "/images/456/create",
			prefix:      false,
			match:       false,
		},
		{pattern: "/images/{prefix:*}/{id:*}/create",
			patternType: patternPath,
			output:      "^/images/(?P<prefix>.*)/(?P<id>.*)/create$",
			input:       "/images/abc/456/create",
			prefix:      false,
			match:       true,
			named:       map[string]string{"id": "456", "prefix": "abc"},
		},
		{pattern: "/v{version:[0-9.]+}/images/{id:*}/create",
			patternType: patternPath,
			output:      "^/v(?P<version>[0-9.]+)/images/(?P<id>.*)/create$",
			input:       "/v1.2.3/images/789/123/456/abc/create",
			prefix:      false,
			match:       true,
			named:       map[string]string{"version": "1.2.3", "id": "789/123/456/abc"},
		},
		{pattern: "/foo.bar",
			patternType: patternPath,
			output:      "^/foo\\.bar$",
			input:       "/foo.bar",
			prefix:      false,
			match:       true,
		},
		{pattern: "/foo{bar:[0-9]+}/baz",
			patternType: patternPath,
			output:      "^/foo(?P<bar>[0-9]+)/baz$",
			input:       "/foo123/baz",
			prefix:      false,
			match:       true,
			named:       map[string]string{"bar": "123"},
		},
		{pattern: "/{id}/",
			patternType: patternPath,
			output:      "^/(?P<id>[^/]+)/$",
			input:       "/123/",
			prefix:      false,
			match:       true,
			named:       map[string]string{"id": "123"},
		},
		{pattern: "{sub:[^.]+}.example.com",
			patternType: patternHost,
			output:      "^(?P<sub>[^.]+)\\.example\\.com$",
			input:       "foo.example.com",
			prefix:      false,
			match:       true,
			named:       map[string]string{"sub": "foo"},
		},
		{pattern: "/{id:*}/create",
			patternType: patternPath,
			output:      "^/(?P<id>.*)/create$",
			input:       "/123/create",
			prefix:      false,
			match:       true,
			named:       map[string]string{"id": "123"},
		},
		{pattern: "/{id:*}/create",
			patternType: patternPath,
			output:      "^/(?P<id>.*)/create",
			input:       "/123/create",
			prefix:      true,
			match:       true,
			named:       map[string]string{"id": "123"},
		},
		{pattern: "/{id:*",
			patternType: patternPath,
			output:      "^/\\{id:\\*",
			input:       "/123/create",
			prefix:      true,
			match:       false,
			named:       map[string]string{},
		},
	}

	for _, tc := range tests {
		patternStr := buildRegexpFromPatternParts(parsePatternTemplate(cleanTemplate(tc.pattern)), tc.patternType, tc.prefix)
		pattern := regexp.MustCompile(patternStr)
		if pattern.MatchString(tc.input) != tc.match {
			t.Errorf("Pattern %q match with input %q expected %v", tc.pattern, tc.input, tc.match)
			t.Errorf("regexp string %q", pattern.String())
		}
		if pattern.String() != tc.output {
			t.Errorf("Pattern %q expected output regexp %q, got %q", tc.pattern, tc.output, pattern.String())
		}
		if tc.named != nil && tc.match {
			matches := pattern.FindStringSubmatch(tc.input)
			names := pattern.SubexpNames()
			if len(matches) != len(names) {
				t.Errorf("Pattern %q match with input %q expected %v named captures, got %v", tc.pattern, tc.input, len(tc.named), len(matches))
				continue
			}
			named := make(map[string]string)
			for i, name := range names {
				if i != 0 && name != "" {
					named[name] = matches[i]
				}
			}
			for k, v := range tc.named {
				if named[k] != v {
					t.Errorf("Pattern %q match with input %q expected named %v, got %v", tc.pattern, tc.input, tc.named, named)
				}
			}
		}
	}
}

// test for detectMatchType function
func TestDetectMatchType(t *testing.T) {
	type testCase struct {
		pattern   string
		matchType matchType
	}

	tests := []testCase{
		{pattern: "/images/123/create", matchType: matchTypeLiteral},
		{pattern: "^/images/create$", matchType: matchTypeLiteral},
		{pattern: "^/images/create", matchType: matchTypePrefix},
		{pattern: "^/images/create.*", matchType: matchTypePrefix},
		{pattern: "^/images/create.*$", matchType: matchTypePrefix},
		{pattern: "/images/{id:[0-9]+}/create", matchType: matchTypeRegexp},
		{pattern: "/images/{prefix:*}/{id:*}/create", matchType: matchTypeRegexp},
		{pattern: "/{id:*}/create", matchType: matchTypeRegexp},
	}

	for _, tc := range tests {
		mt, _, err := detectMatchType(tc.pattern)
		if err != nil {
			t.Errorf("Pattern %q detection error: %v", tc.pattern, err)
			continue
		}
		if mt != tc.matchType {
			t.Errorf("Pattern %q expected match type %v, got %v", tc.pattern, tc.matchType, mt)
		}
	}
}

// test simple regexp converts back down to a literal
func TestSimpleRegexpToLiteral(t *testing.T) {
	type testCase struct {
		pattern string
		prefix  bool
		output  string
	}

	tests := []testCase{
		{pattern: "^/images/create$", prefix: false, output: "/images/create"},
		{pattern: "^/images/create", prefix: true, output: "/images/create"},
	}

	for _, tc := range tests {
		patternStr := buildRegexpFromPatternParts(parsePatternTemplate(cleanTemplate(tc.pattern)), patternPath, tc.prefix)
		_, pattern, _ := detectMatchType(patternStr)

		if pattern != tc.output {
			t.Errorf("Pattern %q expected literal %q, got %q", tc.pattern, tc.output, pattern)
		}
	}
}
