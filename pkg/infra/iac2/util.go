package iac2

import (
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
)

var lowerThenUpper = regexp.MustCompile("([a-z0-9])([A-Z])")

func camelToSnake(s string) string {
	snakedButUppers := lowerThenUpper.ReplaceAllString(s, "${1}_${2}")
	return strings.ToLower(snakedButUppers)
}

func getStructValues(o any) map[string]any {
	val := reflect.ValueOf(o)
	valType := reflect.TypeOf(o)

	fieldCount := val.NumField()
	result := make(map[string]any, fieldCount)

	for i := 0; i < fieldCount; i++ {
		valField := valType.Field(i)
		if !valField.IsExported() {
			panic(fmt.Sprintf(
				`cannot output %s because it has unexported fields (this is a Klotho bug)`,
				valType.Name()))
		}
		fieldName := valField.Name
		fieldValue := val.Field(i).Interface()
		result[fieldName] = fieldValue
	}
	return result
}

func quoteTsString(str string) string {
	result := strings.Builder{}
	result.WriteRune('"')
	for _, char := range str {
		switch char {
		case '"':
			result.WriteString(`\"`)
		case '\'':
			result.WriteString(`\'`)
		case '\\':
			result.WriteString(`\\`)
		case '\b':
			result.WriteString(`\b`)
		case '\f':
			result.WriteString(`\f`)
		case '\n':
			result.WriteString(`\n`)
		case '\r':
			result.WriteString(`\r`)
		case '\t':
			result.WriteString(`\t`)
		default:
			if char < 32 || char > 126 {
				result.WriteString("\\u")
				result.WriteString(strconv.FormatInt(int64(char), 16))
			} else {
				result.WriteRune(char)
			}
		}
	}
	result.WriteRune('"')
	return result.String()
}
