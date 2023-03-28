package iac2

import (
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"unicode"

	"github.com/klothoplatform/klotho/pkg/core"
)

var lowerThenUpper = regexp.MustCompile("([a-z0-9])([A-Z])")

func camelToSnake(s string) string {
	snakedButUppers := lowerThenUpper.ReplaceAllString(s, "${1}_${2}")
	return strings.ToLower(snakedButUppers)
}

func lowercaseFirst(s string) string {
	if s == "" {
		return s
	}
	firstChar := s[:1]
	rest := s[1:]
	return strings.ToLower(firstChar) + rest
}

func toUpperCamel(s string) string {
	sb := strings.Builder{}
	sb.Grow(len(s))
	capitalizeNext := true
	for _, char := range s {
		if unicode.IsLetter(char) || unicode.IsDigit(char) {
			if capitalizeNext {
				char = unicode.ToUpper(char)
				capitalizeNext = false
			}
			sb.WriteRune(char)
		} else {
			capitalizeNext = true
		}
	}
	return sb.String()
}

func structName(v core.Resource) string {
	vType := reflect.TypeOf(v)
	for vType.Kind() == reflect.Pointer {
		vType = vType.Elem()
	}
	return vType.Name()
}

// quoteTsString converts the string into a TypeScript backticked string. We do that rather than a standard json string
// so that it looks nicer in the resulting source code. For example, instead of:
//
//	const someStr = "{\n\t"hello": "world",\n}";
//
// you would get:
//
//	const SomeStr = `{
//		"hello": "world",
//	}`;
func quoteTsString(str string, useDoubleQuotedStrings bool) string {
	result := strings.Builder{}
	if useDoubleQuotedStrings {
		result.WriteString(`"`)
	} else {
		result.WriteRune('`')
	}
	for _, char := range str {
		switch char {
		case '`':
			result.WriteString("\\`")
		case '\b':
			result.WriteString(`\b`)
		case '\f':
			result.WriteString(`\f`)
		case '\r':
			result.WriteString(`\r`)
		case '\\':
			result.WriteString(`\\`)
		case '\t', '"', '\'', '\n':
			result.WriteRune(char)
		default:
			if char < 32 || char > 126 {
				result.WriteString(`\u`)
				result.WriteString(strconv.FormatInt(int64(char), 16))
			} else {
				result.WriteRune(char)
			}
		}
	}
	if useDoubleQuotedStrings {
		result.WriteString(`"`)
	} else {
		result.WriteRune('`')
	}
	return result.String()
}
