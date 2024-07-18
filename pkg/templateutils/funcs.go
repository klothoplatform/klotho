package templateutils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"text/template"
)

var UtilityFunctions = template.FuncMap{
	"split":                strings.Split,
	"join":                 strings.Join,
	"basename":             filepath.Base,
	"filterMatch":          FilterMatch,
	"mapString":            MapString,
	"zipToMap":             ZipToMap,
	"keysToMapWithDefault": KeysToMapWithDefault,
	"replace":              ReplaceAllRegex,
	"hasSuffix":            HasSuffix,
	"toLower":              strings.ToLower,
	"add":                  Add,
	"sub":                  Sub,
	"last":                 Last,
	"makeSlice":            MakeSlice,
	"appendSlice":          AppendSlice,
	"sliceContains":        SliceContains,
	"matches":              Matches,
}

func WithCommonFuncs(funcMap template.FuncMap) template.FuncMap {
	for k, v := range UtilityFunctions {
		funcMap[k] = v
	}
	return funcMap
}

// ToJSON converts any value to a JSON string.
func ToJSON(v any) (string, error) {
	buf := new(bytes.Buffer)
	enc := json.NewEncoder(buf)
	if err := enc.Encode(v); err != nil {
		return "", err
	}
	return strings.TrimSpace(buf.String()), nil
}

// ToJSONPretty converts any value to a pretty-printed JSON string.
func ToJSONPretty(v any) (string, error) {
	buf := new(bytes.Buffer)
	enc := json.NewEncoder(buf)
	enc.SetIndent("", "    ")
	if err := enc.Encode(v); err != nil {
		return "", err
	}
	return strings.TrimSpace(buf.String()), nil
}

// FileBase returns the last element of a filepath.
func FileBase(path string) string {
	return filepath.Base(path)
}

// FileTrimExtFunc returns the path without the extension.
func FileTrimExtFunc(path string) string {
	return strings.TrimSuffix(path, filepath.Ext(path))
}

// FileSep returns the separator for the current OS.
func FileSep() string {
	return string(filepath.Separator)
}

// ReplaceAll replaces all occurrences of old with new in s.
func ReplaceAll(s string, old string, new string) string {
	return strings.ReplaceAll(s, old, new)
}

// Matches returns true if the value matches the regex pattern.
func Matches(pattern, value string) (bool, error) {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return false, err
	}
	return re.MatchString(value), nil
}

// FilterMatch returns a json array by filtering the values array with the regex pattern
func FilterMatch(pattern string, values []string) ([]string, error) {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, err
	}

	matches := make([]string, 0, len(values))
	for _, v := range values {
		if ok := re.MatchString(v); ok {
			matches = append(matches, v)
		}
	}
	return matches, nil
}

// MapString takes in a regex pattern and replacement as well as a json array of strings
// roughly `unmarshal value | sed s/pattern/replace/g | marshal`
func MapString(pattern, replace string, values []string) ([]string, error) {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, err
	}

	nv := make([]string, len(values))
	for i, v := range values {
		nv[i] = re.ReplaceAllString(v, replace)
	}
	return nv, nil
}

// ZipToMap returns a json map by zipping the keys and values arrays
// Example: zipToMap(['a', 'b'], [1, 2]) => {"a": 1, "b": 2}
func ZipToMap(keys []string, valuesArg any) (map[string]any, error) {
	// Have to use reflection here because technically, []string is not assignable to []any
	// thanks Go.
	valuesRefl := reflect.ValueOf(valuesArg)
	if valuesRefl.Kind() != reflect.Slice && valuesRefl.Kind() != reflect.Array {
		return nil, fmt.Errorf("values is not a slice or array")
	}
	if len(keys) != valuesRefl.Len() {
		return nil, fmt.Errorf("key length (%d) != value length (%d)", len(keys), valuesRefl.Len())
	}

	m := make(map[string]any)
	for i, k := range keys {
		m[k] = valuesRefl.Index(i).Interface()
	}
	return m, nil
}

// KeysToMapWithDefault returns a json map by mapping the keys array to the static defaultValue
// Example keysToMapWithDefault(0, ['a', 'b']) => {"a": 0, "b": 0}
func KeysToMapWithDefault(defaultValue any, keys []string) (map[string]any, error) {
	m := make(map[string]any)
	for _, k := range keys {
		m[k] = defaultValue
	}
	return m, nil
}

// Add returns the sum of all the arguments.
func Add(args ...int) int {
	total := 0
	for _, a := range args {
		total += a
	}
	return total
}

// Sub returns the difference of all the arguments.
func Sub(args ...int) int {
	if len(args) == 0 {
		return 0
	}
	total := args[0]
	for _, a := range args[1:] {
		total -= a
	}
	return total
}

// Last returns the last element of a list.
func Last(list any) (any, error) {
	v := reflect.ValueOf(list)
	if v.Kind() != reflect.Slice && v.Kind() != reflect.Array {
		return nil, fmt.Errorf("list is not a slice or array, is %s", v.Kind())
	}
	if v.Len() == 0 {
		return nil, fmt.Errorf("list is empty")
	}
	return v.Index(v.Len() - 1).Interface(), nil
}

// ReplaceAllRegex replaces all occurrences of the regex pattern with the replace string in value.
func ReplaceAllRegex(pattern, replace, value string) (string, error) {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return "", err
	}
	s := re.ReplaceAllString(value, replace)
	return s, nil
}

// MakeSlice creates and returns a new slice of any type.
func MakeSlice() []any {
	return []any{}
}

// AppendSlice appends a value to a slice and returns the updated slice.
func AppendSlice(slice []any, value any) []any {
	return append(slice, value)
}

// SliceContains checks if a slice contains a specific value.
func SliceContains(slice []any, value any) bool {
	for _, v := range slice {
		if v == value {
			return true
		}
	}
	return false
}

// HasSuffix checks if a string has a specific suffix.
func HasSuffix(s, suffix string) bool {
	return strings.HasSuffix(s, suffix)
}
