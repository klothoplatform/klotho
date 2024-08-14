package reflectutil

import (
	"reflect"
	"testing"
)

func TestGetField(t *testing.T) {
	type Nested struct {
		C string
	}

	type Example struct {
		A struct {
			B []Nested
		}
	}

	example := Example{
		A: struct{ B []Nested }{
			B: []Nested{{C: "value"}},
		},
	}

	examplePtr := &example

	exampleMap := map[string]any{
		"A": map[string]any{
			"B": []map[string]any{
				{"C": "value"},
			},
		},
	}

	exampleNestedMap := map[string]any{
		"A": map[string]any{
			"B": map[string]any{
				"C": "value",
			},
		},
	}

	exampleEmptyFields := struct {
		A struct {
			B []struct {
				C string
			}
		}
	}{}

	exampleDeeplyNested := struct {
		A struct {
			B struct {
				C struct {
					D struct {
						E string
					}
				}
			}
		}
	}{
		A: struct {
			B struct {
				C struct {
					D struct {
						E string
					}
				}
			}
		}{
			B: struct {
				C struct {
					D struct {
						E string
					}
				}
			}{
				C: struct {
					D struct {
						E string
					}
				}{
					D: struct {
						E string
					}{
						E: "deepValue",
					},
				},
			},
		},
	}

	exampleInterface := map[string]any{
		"A": struct {
			B []any
		}{
			B: []any{Nested{C: "value"}},
		},
	}

	examplePrimitive := struct {
		A int
		B string
		C float64
	}{
		A: 42,
		B: "hello",
		C: 3.14,
	}

	tests := []struct {
		name      string
		v         any
		fieldExpr string
		want      any
		wantErr   bool
	}{
		{
			name:      "Valid field path",
			v:         example,
			fieldExpr: "A.B[0].C",
			want:      "value",
			wantErr:   false,
		},
		{
			name:      "Valid field path with pointer",
			v:         examplePtr,
			fieldExpr: "A.B[0].C",
			want:      "value",
			wantErr:   false,
		},
		{
			name:      "Valid field path with map",
			v:         exampleMap,
			fieldExpr: "A.B[0].C",
			want:      "value",
			wantErr:   false,
		},
		{
			name:      "Valid field path with top-level slice",
			v:         exampleMap["A"].(map[string]any)["B"],
			fieldExpr: "[0].C",
			want:      "value",
			wantErr:   false,
		},
		{
			name:      "Valid field path with nested map",
			v:         exampleNestedMap,
			fieldExpr: "A.B.C",
			want:      "value",
			wantErr:   false,
		},
		{
			name:      "Empty fields",
			v:         exampleEmptyFields,
			fieldExpr: "A.B[0].C",
			want:      nil,
			wantErr:   true,
		},
		{
			name:      "Deeply nested struct",
			v:         exampleDeeplyNested,
			fieldExpr: "A.B.C.D.E",
			want:      "deepValue",
			wantErr:   false,
		},
		{
			name:      "Invalid field name",
			v:         example,
			fieldExpr: "A.X[0].C",
			want:      nil,
			wantErr:   true,
		},
		{
			name:      "Index out of range",
			v:         example,
			fieldExpr: "A.B[1].C",
			want:      nil,
			wantErr:   true,
		},
		{
			name:      "Field is not slice or array",
			v:         example,
			fieldExpr: "A.B.C",
			want:      nil,
			wantErr:   true,
		},
		{
			name:      "Valid field path with interface",
			v:         exampleInterface,
			fieldExpr: "A.B[0].C",
			want:      "value",
			wantErr:   false,
		},
		{
			name:      "Primitive type int",
			v:         examplePrimitive,
			fieldExpr: "A",
			want:      42,
			wantErr:   false,
		},
		{
			name:      "Primitive type string",
			v:         examplePrimitive,
			fieldExpr: "B",
			want:      "hello",
			wantErr:   false,
		},
		{
			name:      "Primitive type float64",
			v:         examplePrimitive,
			fieldExpr: "C",
			want:      3.14,
			wantErr:   false,
		},
		{
			name:      "Value is primitive type",
			v:         42,
			fieldExpr: "A",
			wantErr:   true,
		},
		{
			name:      "Value is nil",
			v:         reflect.ValueOf(nil),
			fieldExpr: "A",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := reflect.ValueOf(tt.v)
			got, err := GetField(v, tt.fieldExpr)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetField() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !reflect.DeepEqual(got.Interface(), tt.want) {
				t.Errorf("GetField() = %v, want %v", got.Interface(), tt.want)
			}
		})
	}
}

func TestTracePath(t *testing.T) {

	type Address struct {
		Street string
		City   string
	}

	type Person struct {
		Name    string
		Age     int
		Address Address
	}

	type Company struct {
		Name      string
		Employees []Person
	}

	company := Company{
		Name: "Tech Corp",
		Employees: []Person{
			{
				Name: "Alice",
				Age:  30,
				Address: Address{
					Street: "123 Main St",
					City:   "Techville",
				},
			},
			{
				Name: "Bob",
				Age:  35,
				Address: Address{
					Street: "456 Oak Rd",
					City:   "Codeburg",
				},
			},
		},
	}

	nestedMap := map[string]any{
		"users": map[string]any{
			"alice": map[string]any{
				"age":  30,
				"city": "New York",
			},
			"bob": map[string]any{
				"age": 28,
				"address": map[string]string{
					"street": "123 Main St",
					"zip":    "12345",
				},
			},
		},
		"settings": map[string]bool{
			"active": true,
			"debug":  false,
		},
	}

	tests := []struct {
		name        string
		value       any
		path        string
		wantLevels  int
		wantLastVal any
		wantErr     bool
	}{
		{
			name:        "Simple field access",
			value:       company,
			path:        "Name",
			wantLevels:  2,
			wantLastVal: "Tech Corp",
			wantErr:     false,
		},
		{
			name:        "Nested struct field access",
			value:       company,
			path:        "Employees.0.Name",
			wantLevels:  4,
			wantLastVal: "Alice",
			wantErr:     false,
		},
		{
			name:        "Deep nested struct field access",
			value:       company,
			path:        "Employees.1.Address.City",
			wantLevels:  5,
			wantLastVal: "Codeburg",
			wantErr:     false,
		},
		{
			name:        "Array index access",
			value:       company,
			path:        "Employees.1",
			wantLevels:  3,
			wantLastVal: company.Employees[1],
			wantErr:     false,
		},
		{
			name:       "Non-existent field",
			value:      company,
			path:       "Location",
			wantLevels: 0,
			wantErr:    true,
		},
		{
			name:       "Invalid array index",
			value:      company,
			path:       "Employees.5",
			wantLevels: 0,
			wantErr:    true,
		},
		{
			name:       "Invalid path on primitive",
			value:      42,
			path:       "Value",
			wantLevels: 0,
			wantErr:    true,
		},
		{
			name:        "Empty path",
			value:       company,
			path:        "",
			wantLevels:  1,
			wantLastVal: company,
			wantErr:     false,
		},
		{
			name:       "Nil value",
			path:       "Any",
			wantLevels: 0,
			wantErr:    true,
		},
		{
			name:        "Simple map access",
			value:       nestedMap,
			path:        "settings.active",
			wantLevels:  3,
			wantLastVal: true,
			wantErr:     false,
		},
		{
			name:        "Nested map access",
			value:       nestedMap,
			path:        "users.alice.age",
			wantLevels:  4,
			wantLastVal: 30,
			wantErr:     false,
		},
		{
			name:        "Deep nested map access",
			value:       nestedMap,
			path:        "users.bob.address.street",
			wantLevels:  5,
			wantLastVal: "123 Main St",
			wantErr:     false,
		},
		{
			name:        "Non-existent map key",
			value:       nestedMap,
			path:        "users.charlie",
			wantLevels:  0,
			wantLastVal: nil,
			wantErr:     true,
		},
		{
			name:    "Map key with dot",
			value:   map[string]int{"key.with.dot": 42},
			path:    "key.with.dot",
			wantErr: true,
		},
		{
			name: "Map with non-string key",
			value: map[int]string{
				1: "one",
				2: "two",
			},
			path:    "2",
			wantErr: true,
		},
		{
			name: "Mixed struct and map",
			value: struct {
				Data map[string]int
			}{
				Data: map[string]int{"value": 42},
			},
			path:        "Data.value",
			wantLevels:  3,
			wantLastVal: 42,
			wantErr:     false,
		},
		{
			name:    "Empty map",
			value:   map[string]any{},
			path:    "any.path",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := TracePath(reflect.ValueOf(tt.value), tt.path)

			if (err != nil) != tt.wantErr {
				t.Errorf("TracePath() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if len(got) != tt.wantLevels {
					t.Errorf("TracePath() returned %d levels, want %d", len(got), tt.wantLevels)
				}

				lastVal := got[len(got)-1].Interface()
				if !reflect.DeepEqual(lastVal, tt.wantLastVal) {
					t.Errorf("TracePath() last value = %v, want %v", lastVal, tt.wantLastVal)
				}

				// Check if the first element is the root
				if !reflect.DeepEqual(got[0].Interface(), tt.value) {
					t.Errorf("TracePath() first value = %v, want %v", got[0].Interface(), tt.value)
				}
			}
		})
	}
}

func TestFirstOfType(t *testing.T) {
	tests := []struct {
		name     string
		values   []interface{}
		wantType interface{}
		want     interface{}
		found    bool
	}{
		{
			name:     "Find int in mixed slice",
			values:   []interface{}{"string", 42, true, 3.14},
			wantType: 0,
			want:     42,
			found:    true,
		},
		{
			name:     "Find string in mixed slice",
			values:   []interface{}{42, true, "hello", 3.14},
			wantType: "",
			want:     "hello",
			found:    true,
		},
		{
			name:     "Find bool in mixed slice",
			values:   []interface{}{42, "string", 3.14, false},
			wantType: false,
			want:     false,
			found:    true,
		},
		{
			name:     "Find float64 in mixed slice",
			values:   []interface{}{42, "string", true, 3.14},
			wantType: float64(0),
			want:     3.14,
			found:    true,
		},
		{
			name:     "Type not found",
			values:   []interface{}{42, "string", true, 3.14},
			wantType: []int{},
			want:     []int(nil),
			found:    false,
		},
		{
			name:     "Empty slice",
			values:   []interface{}{},
			wantType: int(0),
			want:     0,
			found:    false,
		},
		{
			name:     "Find struct in mixed slice",
			values:   []interface{}{42, struct{ name string }{"John"}, "string"},
			wantType: struct{ name string }{},
			want:     struct{ name string }{"John"},
			found:    true,
		},
		{
			name:     "Find pointer in mixed slice",
			values:   []interface{}{42, &struct{ name string }{"John"}, "string"},
			wantType: (*struct{ name string })(nil),
			want:     &struct{ name string }{"John"},
			found:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			values := make([]reflect.Value, len(tt.values))
			for i, v := range tt.values {
				values[i] = reflect.ValueOf(v)
			}

			switch tt.wantType.(type) {
			case int:
				got, found := FirstOfType[int](values)
				if found != tt.found || (found && got != tt.want) {
					t.Errorf("FirstOfType() = %v, %v, want %v, %v", got, found, tt.want, tt.found)
				}
			case string:
				got, found := FirstOfType[string](values)
				if found != tt.found || (found && got != tt.want) {
					t.Errorf("FirstOfType() = %v, %v, want %v, %v", got, found, tt.want, tt.found)
				}
			case bool:
				got, found := FirstOfType[bool](values)
				if found != tt.found || (found && got != tt.want) {
					t.Errorf("FirstOfType() = %v, %v, want %v, %v", got, found, tt.want, tt.found)
				}
			case float64:
				got, found := FirstOfType[float64](values)
				if found != tt.found || (found && got != tt.want) {
					t.Errorf("FirstOfType() = %v, %v, want %v, %v", got, found, tt.want, tt.found)
				}
			case []int:
				got, found := FirstOfType[[]int](values)
				if found != tt.found || (found && !reflect.DeepEqual(got, tt.want)) {
					t.Errorf("FirstOfType() = %v, %v, want %v, %v", got, found, tt.want, tt.found)
				}
			case struct{ name string }:
				got, found := FirstOfType[struct{ name string }](values)
				if found != tt.found || (found && !reflect.DeepEqual(got, tt.want)) {
					t.Errorf("FirstOfType() = %v, %v, want %v, %v", got, found, tt.want, tt.found)
				}
			case *struct{ name string }:
				got, found := FirstOfType[*struct{ name string }](values)
				if found != tt.found || (found && !reflect.DeepEqual(got, tt.want)) {
					t.Errorf("FirstOfType() = %v, %v, want %v, %v", got, found, tt.want, tt.found)
				}
			default:
				t.Errorf("Unsupported type in test case: %T", tt.wantType)
			}
		})
	}
}

func TestLastOfType(t *testing.T) {
	tests := []struct {
		name     string
		values   []interface{}
		wantType interface{}
		want     interface{}
		found    bool
	}{
		{
			name:     "Find last int in mixed slice",
			values:   []interface{}{"string", 42, true, 3.14, 99},
			wantType: 0,
			want:     99,
			found:    true,
		},
		{
			name:     "Find last string in mixed slice",
			values:   []interface{}{42, "hello", true, 3.14, "world"},
			wantType: "",
			want:     "world",
			found:    true,
		},
		{
			name:     "Find last bool in mixed slice",
			values:   []interface{}{42, true, "string", 3.14, false},
			wantType: false,
			want:     false,
			found:    true,
		},
		{
			name:     "Find last float64 in mixed slice",
			values:   []interface{}{3.14, 42, "string", true, 2.718},
			wantType: float64(0),
			want:     2.718,
			found:    true,
		},
		{
			name:     "Type not found",
			values:   []interface{}{42, "string", true, 3.14},
			wantType: []int{},
			want:     []int(nil),
			found:    false,
		},
		{
			name:     "Empty slice",
			values:   []interface{}{},
			wantType: 0,
			want:     0,
			found:    false,
		},
		{
			name:     "Find last struct in mixed slice",
			values:   []interface{}{struct{ name string }{"John"}, 42, struct{ name string }{"Jane"}, "string"},
			wantType: struct{ name string }{},
			want:     struct{ name string }{"Jane"},
			found:    true,
		},
		{
			name:     "Find last pointer in mixed slice",
			values:   []interface{}{&struct{ name string }{"John"}, 42, &struct{ name string }{"Jane"}, "string"},
			wantType: (*struct{ name string })(nil),
			want:     &struct{ name string }{"Jane"},
			found:    true,
		},
		{
			name:     "Only one matching type",
			values:   []interface{}{"string", 42, true, 3.14},
			wantType: 0,
			want:     42,
			found:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			values := make([]reflect.Value, len(tt.values))
			for i, v := range tt.values {
				values[i] = reflect.ValueOf(v)
			}

			switch tt.wantType.(type) {
			case int:
				got, found := LastOfType[int](values)
				if found != tt.found || (found && got != tt.want) {
					t.Errorf("LastOfType() = %v, %v, want %v, %v", got, found, tt.want, tt.found)
				}
			case string:
				got, found := LastOfType[string](values)
				if found != tt.found || (found && got != tt.want) {
					t.Errorf("LastOfType() = %v, %v, want %v, %v", got, found, tt.want, tt.found)
				}
			case bool:
				got, found := LastOfType[bool](values)
				if found != tt.found || (found && got != tt.want) {
					t.Errorf("LastOfType() = %v, %v, want %v, %v", got, found, tt.want, tt.found)
				}
			case float64:
				got, found := LastOfType[float64](values)
				if found != tt.found || (found && got != tt.want) {
					t.Errorf("LastOfType() = %v, %v, want %v, %v", got, found, tt.want, tt.found)
				}
			case []int:
				got, found := LastOfType[[]int](values)
				if found != tt.found || (found && !reflect.DeepEqual(got, tt.want)) {
					t.Errorf("LastOfType() = %v, %v, want %v, %v", got, found, tt.want, tt.found)
				}
			case struct{ name string }:
				got, found := LastOfType[struct{ name string }](values)
				if found != tt.found || (found && !reflect.DeepEqual(got, tt.want)) {
					t.Errorf("LastOfType() = %v, %v, want %v, %v", got, found, tt.want, tt.found)
				}
			case *struct{ name string }:
				got, found := LastOfType[*struct{ name string }](values)
				if found != tt.found || (found && !reflect.DeepEqual(got, tt.want)) {
					t.Errorf("LastOfType() = %v, %v, want %v, %v", got, found, tt.want, tt.found)
				}
			default:
				t.Errorf("Unsupported type in test case: %T", tt.wantType)
			}
		})
	}
}
