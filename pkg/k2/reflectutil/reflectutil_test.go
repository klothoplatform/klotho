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

	exampleMap := map[string]interface{}{
		"A": map[string]interface{}{
			"B": []map[string]interface{}{
				{"C": "value"},
			},
		},
	}

	exampleNestedMap := map[string]interface{}{
		"A": map[string]interface{}{
			"B": map[string]interface{}{
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

	exampleInterface := map[string]interface{}{
		"A": struct {
			B []interface{}
		}{
			B: []interface{}{Nested{C: "value"}},
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
		v         interface{}
		fieldExpr string
		want      interface{}
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
			v:         exampleMap["A"].(map[string]interface{})["B"],
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
