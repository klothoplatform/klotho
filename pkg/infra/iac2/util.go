package iac2

import (
	"github.com/klothoplatform/klotho/pkg/core"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"unicode"

	"go.uber.org/zap"
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

func getStructValues(o any) map[string]any {
	val := reflect.ValueOf(o)
	for val.Kind() == reflect.Pointer {
		val = val.Elem()
	}
	if val.Kind() != reflect.Struct {
		return nil
	}
	valType := val.Type()

	fieldCount := val.NumField()
	result := make(map[string]any, fieldCount)

	for i := 0; i < fieldCount; i++ {
		valField := valType.Field(i)
		if !valField.IsExported() {
			zap.S().Debugf(`Ignoring unexported field %s on %s`, valField.Name, valType.Name())
			continue
		}
		fieldValue := val.Field(i)
		fieldData := fieldValue.Interface()
		result[valField.Name] = fieldData
	}
	return result
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
func quoteTsString(str string) string {
	result := strings.Builder{}
	result.WriteRune('`')
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
	result.WriteRune('`')
	return result.String()
}
