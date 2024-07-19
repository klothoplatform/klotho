package templateutils

import (
	"reflect"
	"testing"
	"text/template"
)

func TestUtilityFunctions(t *testing.T) {
	tests := []struct {
		name     string
		funcName string
		args     []any
		want     any
		wantErr  bool
	}{
		// split tests
		{"Split basic", "split", []any{"a,b,c", ","}, []string{"a", "b", "c"}, false},
		{"Split no separator", "split", []any{"abc", ","}, []string{"abc"}, false},
		{"Split empty string", "split", []any{"", ","}, []string{""}, false},
		{"Split with empty separator", "split", []any{"abc", ""}, []string{"a", "b", "c"}, false},

		// join tests
		{"Join basic", "join", []any{[]string{"a", "b", "c"}, ","}, "a,b,c", false},
		{"Join empty slice", "join", []any{[]string{}, ","}, "", false},
		{"Join single element", "join", []any{[]string{"a"}, ","}, "a", false},
		{"Join with empty separator", "join", []any{[]string{"a", "b", "c"}, ""}, "abc", false},

		// basename tests
		{"Basename basic", "basename", []any{"/path/to/file.txt"}, "file.txt", false},
		{"Basename no directory", "basename", []any{"file.txt"}, "file.txt", false},
		{"Basename with trailing slash", "basename", []any{"/path/to/directory/"}, "directory", false},
		{"Basename empty string", "basename", []any{""}, ".", false},

		// filterMatch tests
		{"FilterMatch basic", "filterMatch", []any{"^a", []string{"apple", "banana", "avocado"}}, []string{"apple", "avocado"}, false},
		{"FilterMatch no matches", "filterMatch", []any{"^z", []string{"apple", "banana", "avocado"}}, []string{}, false},
		{"FilterMatch empty slice", "filterMatch", []any{"^a", []string{}}, []string{}, false},
		{"FilterMatch invalid regex", "filterMatch", []any{"[", []string{"apple", "banana", "avocado"}}, nil, true},

		// mapString tests
		{"MapString basic", "mapString", []any{"a", "A", []string{"apple", "banana", "avocado"}}, []string{"Apple", "bAnAnA", "AvocAdo"}, false},
		{"MapString no matches", "mapString", []any{"z", "Z", []string{"apple", "banana", "avocado"}}, []string{"apple", "banana", "avocado"}, false},
		{"MapString empty slice", "mapString", []any{"a", "A", []string{}}, []string{}, false},
		{"MapString invalid regex", "mapString", []any{"[", "A", []string{"apple", "banana", "avocado"}}, nil, true},

		// zipToMap tests
		{"ZipToMap basic", "zipToMap", []any{[]string{"a", "b"}, []int{1, 2}}, map[string]any{"a": 1, "b": 2}, false},
		{"ZipToMap empty slices", "zipToMap", []any{[]string{}, []int{}}, map[string]any{}, false},
		{"ZipToMap mismatched lengths", "zipToMap", []any{[]string{"a", "b"}, []int{1}}, nil, true},
		{"ZipToMap non-slice values", "zipToMap", []any{[]string{"a", "b"}, 1}, nil, true},

		// keysToMapWithDefault tests
		{"KeysToMapWithDefault basic", "keysToMapWithDefault", []any{0, []string{"a", "b"}}, map[string]any{"a": 0, "b": 0}, false},
		{"KeysToMapWithDefault empty slice", "keysToMapWithDefault", []any{0, []string{}}, map[string]any{}, false},
		{"KeysToMapWithDefault string default", "keysToMapWithDefault", []any{"default", []string{"a", "b"}}, map[string]any{"a": "default", "b": "default"}, false},

		// replaceAll tests
		{"ReplaceAll basic", "replaceAll", []any{"hello world", "o", "0"}, "hell0 w0rld", false},
		{"ReplaceAll no matches", "replaceAll", []any{"hello world", "z", "0"}, "hello world", false},
		{"ReplaceAll empty string", "replaceAll", []any{"", "o", "0"}, "", false},
		{"ReplaceAll replace with empty", "replaceAll", []any{"hello world", "o", ""}, "hell wrld", false},

		// hasSuffix tests
		{"HasSuffix true", "hasSuffix", []any{"filename.txt", ".txt"}, true, false},
		{"HasSuffix false", "hasSuffix", []any{"filename.txt", ".jpg"}, false, false},
		{"HasSuffix empty suffix", "hasSuffix", []any{"filename.txt", ""}, true, false},
		{"HasSuffix empty string", "hasSuffix", []any{"", ".txt"}, false, false},

		// toLower tests
		{"ToLower basic", "toLower", []any{"Hello World"}, "hello world", false},
		{"ToLower already lowercase", "toLower", []any{"hello world"}, "hello world", false},
		{"ToLower empty string", "toLower", []any{""}, "", false},
		{"ToLower with numbers", "toLower", []any{"HeLLo 123"}, "hello 123", false},

		// toUpper tests
		{"ToUpper basic", "toUpper", []any{"Hello World"}, "HELLO WORLD", false},
		{"ToUpper already uppercase", "toUpper", []any{"HELLO WORLD"}, "HELLO WORLD", false},
		{"ToUpper empty string", "toUpper", []any{""}, "", false},
		{"ToUpper with numbers", "toUpper", []any{"HeLLo 123"}, "HELLO 123", false},

		// add tests
		{"Add basic", "add", []any{1, 2, 3}, 6, false},
		{"Add single number", "add", []any{5}, 5, false},
		{"Add no numbers", "add", []any{}, 0, false},
		{"Add negative numbers", "add", []any{1, -2, 3}, 2, false},

		// sub tests
		{"Sub basic", "sub", []any{10, 3, 2}, 5, false},
		{"Sub single number", "sub", []any{5}, 5, false},
		{"Sub no numbers", "sub", []any{}, 0, false},
		{"Sub negative numbers", "sub", []any{1, -2, 3}, 0, false},

		// last tests
		{"Last basic", "last", []any{[]int{1, 2, 3}}, 3, false},
		{"Last single element", "last", []any{[]int{1}}, 1, false},
		{"Last empty slice", "last", []any{[]int{}}, nil, true},
		{"Last non-slice", "last", []any{1}, nil, true},

		// makeSlice tests
		{"MakeSlice basic", "makeSlice", []any{}, []any{}, false},

		// appendSlice tests
		{"AppendSlice basic", "appendSlice", []any{[]any{1, 2}, 3}, []any{1, 2, 3}, false},
		{"AppendSlice to empty slice", "appendSlice", []any{[]any{}, 1}, []any{1}, false},
		{"AppendSlice different types", "appendSlice", []any{[]any{1, "two"}, 3.0}, []any{1, "two", 3.0}, false},

		// sliceContains tests
		{"SliceContains true", "sliceContains", []any{[]any{1, 2, 3}, 2}, true, false},
		{"SliceContains false", "sliceContains", []any{[]any{1, 2, 3}, 4}, false, false},
		{"SliceContains empty slice", "sliceContains", []any{[]any{}, 1}, false, false},
		{"SliceContains different types", "sliceContains", []any{[]any{1, "two", 3.0}, "two"}, true, false},

		// matches tests
		{"Matches true", "matches", []any{"^a", "apple"}, true, false},
		{"Matches false", "matches", []any{"^b", "apple"}, false, false},
		{"Matches empty string", "matches", []any{".*", ""}, true, false},
		{"Matches invalid regex", "matches", []any{"[", "apple"}, nil, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fn := UtilityFunctions[tt.funcName]
			if fn == nil {
				t.Fatalf("function %s not found in UtilityFunctions", tt.funcName)
			}

			fnValue := reflect.ValueOf(fn)
			args := make([]reflect.Value, len(tt.args))
			for i, arg := range tt.args {
				args[i] = reflect.ValueOf(arg)
			}

			results := fnValue.Call(args)

			if len(results) == 0 {
				t.Fatalf("function %s returned no results", tt.funcName)
			}

			var got any
			var err error

			if len(results) == 2 {
				got = results[0].Interface()
				err, _ = results[1].Interface().(error)
			} else {
				got = results[0].Interface()
			}

			if (err != nil) != tt.wantErr {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected an error, but got nil")
				}
				return
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

// Additional tests for functions not in UtilityFunctions map

func TestToJSON(t *testing.T) {
	tests := []struct {
		name    string
		input   any
		want    string
		wantErr bool
	}{
		{"Basic map", map[string]int{"a": 1, "b": 2}, `{"a":1,"b":2}`, false},
		{"Basic slice", []int{1, 2, 3}, `[1,2,3]`, false},
		{"Empty map", map[string]int{}, `{}`, false},
		{"Empty slice", []int{}, `[]`, false},
		{"Nested structure", map[string]any{"a": 1, "b": []int{2, 3}}, `{"a":1,"b":[2,3]}`, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ToJSON(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ToJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ToJSON() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestToJSONPretty(t *testing.T) {
	tests := []struct {
		name    string
		input   any
		want    string
		wantErr bool
	}{
		{"Basic map", map[string]int{"a": 1, "b": 2}, "{\n    \"a\": 1,\n    \"b\": 2\n}", false},
		{"Basic slice", []int{1, 2, 3}, "[\n    1,\n    2,\n    3\n]", false},
		{"Empty map", map[string]int{}, "{}", false},
		{"Empty slice", []int{}, "[]", false},
		{"Nested structure", map[string]any{"a": 1, "b": []int{2, 3}}, "{\n    \"a\": 1,\n    \"b\": [\n        2,\n        3\n    ]\n}", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ToJSONPretty(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ToJSONPretty() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ToJSONPretty() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFileBase(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"Basic path", "/path/to/file.txt", "file.txt"},
		{"No directory", "file.txt", "file.txt"},
		{"Trailing slash", "/path/to/directory/", "directory"},
		{"Empty string", "", "."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := FileBase(tt.input); got != tt.want {
				t.Errorf("FileBase() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFileTrimExtFunc(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"Basic file", "file.txt", "file"},
		{"No extension", "file", "file"},
		{"Multiple dots", "file.tar.gz", "file.tar"},
		{"Hidden file (not supported)", ".hidden", ""}, // change want to ".hidden" if adding support for hidden files
		{"Empty string", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := FileTrimExtFunc(tt.input); got != tt.want {
				t.Errorf("FileTrimExtFunc() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFileSep(t *testing.T) {
	got := FileSep()
	if got != "/" && got != "\\" {
		t.Errorf("FileSep() = %v, want either '/' or '\\'", got)
	}
}

func TestReplaceAllRegex(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		replace string
		value   string
		want    string
		wantErr bool
	}{
		{"Basic replacement", "a+", "b", "aaa bbb aaa", "b bbb b", false},
		{"No matches", "z+", "b", "aaa bbb aaa", "aaa bbb aaa", false},
		{"Empty string", "a+", "b", "", "", false},
		{"Replace with empty", "a+", "", "aaa bbb aaa", " bbb ", false},
		{"Invalid regex", "[", "b", "aaa bbb aaa", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ReplaceAllRegex(tt.pattern, tt.replace, tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("ReplaceAllRegex() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ReplaceAllRegex() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestWithCommonFuncs tests the WithCommonFuncs function
func TestWithCommonFuncs(t *testing.T) {
	// Create a custom FuncMap
	customFuncMap := template.FuncMap{
		"customFunc": func() string { return "custom" },
	}

	// Apply WithCommonFuncs
	resultFuncMap := WithCommonFuncs(customFuncMap)

	// Check if the custom function is still present
	if customFunc, ok := resultFuncMap["customFunc"]; !ok {
		t.Errorf("WithCommonFuncs() did not preserve custom function")
	} else {
		if customFunc.(func() string)() != "custom" {
			t.Errorf("WithCommonFuncs() altered custom function behavior")
		}
	}

	// Check if common functions were added
	for funcName := range UtilityFunctions {
		if _, ok := resultFuncMap[funcName]; !ok {
			t.Errorf("WithCommonFuncs() did not add common function %s", funcName)
		}
	}

	// Ensure no functions were lost
	expectedLength := len(UtilityFunctions) + 1 // +1 for the custom function
	if len(resultFuncMap) != expectedLength {
		t.Errorf("WithCommonFuncs() resulted in unexpected number of functions. Got %d, want %d", len(resultFuncMap), expectedLength)
	}
}

// Additional helper function to test error cases
func TestErrorCases(t *testing.T) {
	errorTests := []struct {
		name     string
		funcName string
		args     []any
		wantErr  bool
	}{
		{"ZipToMap mismatched lengths", "zipToMap", []any{[]string{"a", "b"}, []int{1}}, true},
		{"ZipToMap non-slice values", "zipToMap", []any{[]string{"a", "b"}, 1}, true},
		{"FilterMatch invalid regex", "filterMatch", []any{"[", []string{"apple", "banana", "avocado"}}, true},
		{"MapString invalid regex", "mapString", []any{"[", "A", []string{"apple", "banana", "avocado"}}, true},
		{"Last empty slice", "last", []any{[]int{}}, true},
		{"Last non-slice", "last", []any{1}, true},
		{"Matches invalid regex", "matches", []any{"[", "apple"}, true},
	}

	for _, tt := range errorTests {
		t.Run(tt.name, func(t *testing.T) {
			fn := UtilityFunctions[tt.funcName]
			if fn == nil {
				t.Fatalf("function %s not found in UtilityFunctions", tt.funcName)
			}

			fnValue := reflect.ValueOf(fn)
			args := make([]reflect.Value, len(tt.args))
			for i, arg := range tt.args {
				args[i] = reflect.ValueOf(arg)
			}

			results := fnValue.Call(args)

			if len(results) != 2 {
				t.Fatalf("expected function %s to return 2 values (result and error)", tt.funcName)
			}

			err, ok := results[1].Interface().(error)
			if !ok {
				t.Fatalf("second return value of function %s is not an error", tt.funcName)
			}

			if (err != nil) != tt.wantErr {
				t.Errorf("function %s error = %v, wantErr %v", tt.funcName, err, tt.wantErr)
			}
		})
	}
}
