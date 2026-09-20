package remux

import (
	"regexp"
	"testing"
)

// tdd test for Pattern function
func TestPatternMatch(t *testing.T) {
	type testCase struct {
		pattern string
		input   string
		match   bool
		named   map[string]string
	}

	tests := []testCase{
		{pattern: "/images/{id:*}/create",
			input: "/images/123/create",
			match: true, named: map[string]string{"id": "123"},
		},
		{pattern: "/images/{id:*}/create",
			input: "/images/123/delete",
			match: false,
		},
		{pattern: "/images/{prefix:*}/{id:*}/create",
			input: "/images/456/create",
			match: false,
		},
		{pattern: "/images/{prefix:*}/{id:*}/create",
			input: "/images/abc/456/create",
			match: true,
			named: map[string]string{"id": "456", "prefix": "abc"},
		},
		{pattern: "/v{version:[0-9.]+}/images/{id:*}/create",
			input: "/v1.2.3/images/789/123/456/abc/create",
			match: true,
			named: map[string]string{"version": "1.2.3", "id": "789/123/456/abc"},
		},
	}

	for _, tc := range tests {
		patternStr := Pattern(tc.pattern)
		pattern := regexp.MustCompile(patternStr)
		if pattern.MatchString(tc.input) != tc.match {
			t.Errorf("Pattern %q match with input %q expected %v", tc.pattern, tc.input, tc.match)
			t.Errorf("regexp string %q", pattern.String())
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
