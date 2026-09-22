package remux

import (
	"regexp"
	"regexp/syntax"
	"strings"
)

type patternParts struct {
	literal string

	name    string
	pattern string
}

// parsePatternTemplate parses the given pattern template string into its constituent parts.
func parsePatternTemplate(str string) []patternParts {
	var parts []patternParts
	var raw strings.Builder
	for i := 0; i < len(str); i++ {
		if str[i] != '{' {
			raw.WriteByte(str[i])
			continue
		}

		// flush accumulated literal text
		if raw.Len() > 0 {
			parts = append(parts, patternParts{literal: raw.String()})
			raw.Reset()
		}

		j := i + 1
		indent := 0

		for j < len(str) && (str[j] != '}' || indent > 0) {
			switch str[j] {
			case '{':
				indent++
			case '}':
				indent--
			}
			j++
		}

		// malformed pattern, treat as literal text
		if j >= len(str) {
			raw.WriteString(str[i:])
			break
		}

		content := str[i+1 : j]
		partsSplit := strings.SplitN(content, ":", 2)

		if len(partsSplit) == 2 {
			parts = append(parts, patternParts{name: partsSplit[0], pattern: partsSplit[1]})
		} else {
			parts = append(parts, patternParts{name: content})
		}

		i = j
	}

	// flush trailing literal text
	if raw.Len() > 0 {
		parts = append(parts, patternParts{literal: raw.String()})
	}

	return parts

}

func cleanTemplate(str string) string {
	str = strings.TrimPrefix(str, "^")
	str = strings.TrimSuffix(str, "$")
	return str
}

type patternType string

const (
	patternPath patternType = "[^/]+"
	patternHost patternType = "[^.]+"
	patternAny  patternType = ".*"
)

// buildRegexpFromPatternParts constructs a regular expression string from the given pattern parts.
// The patternType parameter specifies the type of pattern (e.g., path, host, any).
func buildRegexpFromPatternParts(parts []patternParts, defaultPattern patternType, prefix bool) string {
	var result strings.Builder
	result.WriteString("^")
	for _, part := range parts {
		if part.literal != "" {
			result.WriteString(regexp.QuoteMeta(part.literal))
			continue
		}
		if part.name == "" {
			continue
		}
		if part.pattern == "" {
			part.pattern = string(defaultPattern)
		}
		if part.pattern == "*" {
			if defaultPattern == patternPath {
				part.pattern = string(patternAny)
			} else {
				part.pattern = string(defaultPattern)
			}
		}
		result.WriteString("(?P<" + part.name + ">" + part.pattern + ")")
	}
	if !prefix {
		result.WriteString("$")
	}
	return result.String()
}

type matchType int

const (
	matchTypeRegexp matchType = iota
	matchTypeLiteral
	matchTypePrefix
)

// detectMatchType determines the type of match for the given pattern string.
// It returns the match type, the literal or prefix value if applicable, and an error if the pattern is invalid.
func detectMatchType(pattern string) (matchType, string, error) {
	re, err := syntax.Parse(pattern, syntax.Perl)
	if err != nil {
		return matchTypeRegexp, "", err
	}

	re = re.Simplify()

	if v, ok := isLiteral(re); ok {
		return matchTypeLiteral, v, nil
	}

	if v, ok := isPrefix(re); ok {
		return matchTypePrefix, v, nil
	}
	return matchTypeRegexp, "", nil
}

// isLiteral checks if the given regular expression represents a literal pattern.
func isLiteral(re *syntax.Regexp) (string, bool) {
	if re.Op == syntax.OpLiteral {
		return string(re.Rune), true
	}

	if re.Op == syntax.OpConcat &&
		len(re.Sub) == 3 &&
		re.Sub[0].Op == syntax.OpBeginText &&
		re.Sub[1].Op == syntax.OpLiteral &&
		re.Sub[2].Op == syntax.OpEndText {

		return string(re.Sub[1].Rune), true
	}

	return "", false
}

// isPrefix checks if the given regular expression represents a prefix pattern.
func isPrefix(re *syntax.Regexp) (string, bool) {
	if re.Op != syntax.OpConcat {
		return "", false
	}

	// ^literal
	if len(re.Sub) == 2 &&
		re.Sub[0].Op == syntax.OpBeginText &&
		re.Sub[1].Op == syntax.OpLiteral {
		return string(re.Sub[1].Rune), true
	}

	// ^literal.*
	if len(re.Sub) >= 3 &&
		re.Sub[0].Op == syntax.OpBeginText &&
		re.Sub[1].Op == syntax.OpLiteral {

		starIdx := 2

		if re.Sub[starIdx].Op == syntax.OpStar &&
			len(re.Sub[starIdx].Sub) == 1 &&
			re.Sub[starIdx].Sub[0].Op == syntax.OpAnyCharNotNL {

			// allow optional trailing EndText
			if len(re.Sub) == 3 ||
				(len(re.Sub) == 4 &&
					re.Sub[3].Op == syntax.OpEndText) {

				return string(re.Sub[1].Rune), true
			}
		}
	}

	return "", false
}

func strToPattern(in string, pt patternType, prefix bool) (string, *regexp.Regexp, error) {

	parts := parsePatternTemplate(cleanTemplate(in))
	str := buildRegexpFromPatternParts(parts, pt, prefix)

	re, simple, err := detectMatchType(str)
	if err != nil {
		return "", nil, err
	}

	switch re {
	case matchTypeLiteral:
		return simple, nil, nil
	case matchTypePrefix:
		return simple, nil, nil
	case matchTypeRegexp:
		reg, err := regexp.Compile(str)
		if err != nil {
			return "", nil, err
		}
		return str, reg, nil
	}

	return "", nil, nil
}

func mustStrToPattern(in string, pt patternType, prefix bool) (string, *regexp.Regexp) {
	str, reg, err := strToPattern(in, pt, prefix)
	if err != nil {
		panic(err)
	}
	return str, reg
}
