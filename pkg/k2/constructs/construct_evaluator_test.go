package constructs

import (
	"reflect"
	"testing"

	"github.com/klothoplatform/klotho/pkg/k2/constructs/template"

	"github.com/klothoplatform/klotho/pkg/construct"
	"github.com/klothoplatform/klotho/pkg/k2/model"
	"github.com/stretchr/testify/assert"
)

type stringerStructInput struct {
	Field1 string
}

func (s stringerStructInput) String() string {
	return s.Field1
}

func TestInterpolateValue(t *testing.T) {
	simpleStruct := struct {
		Field1 string
	}{
		Field1: "Hello",
	}

	mockConstruct := &Construct{
		URN: model.URN{ResourceID: "test-construct"},
		Inputs: map[string]any{
			"stringInput": "Hello",
			"intInput":    42,
			"mapInput": map[string]any{
				"key1":                  "value1",
				"key2":                  2,
				"${inputs:stringInput}": "value3",
			},
			"sliceInput": []any{"a", "b", "c"},
			"stringerStructInput": stringerStructInput{
				Field1: "Hello",
			},
			"structInput": simpleStruct,
		},
		Resources: map[string]*Resource{
			"testResource": {
				Id: construct.ResourceId{
					Provider: "test",
					Type:     "resource",
					Name:     "testResource",
				},
				Properties: construct.Properties{
					"prop1": "value1",
					"prop2": 2,
				},
			},
		},
	}

	tests := []struct {
		name     string
		rawValue any
		data     DynamicValueData
		expected any
		hasError bool
	}{
		{
			name:     "Simple string interpolation",
			rawValue: "${inputs:stringInput}",
			data: DynamicValueData{
				currentOwner: mockConstruct,
			},
			expected: "Hello",
		},
		{
			name:     "Integer interpolation",
			rawValue: "${inputs:intInput}",
			data: DynamicValueData{
				currentOwner: mockConstruct,
			},
			expected: 42,
		},
		{
			name:     "Map value interpolation",
			rawValue: "${inputs:mapInput.key1}",
			data: DynamicValueData{
				currentOwner: mockConstruct,
			},
			expected: "value1",
		},
		{
			name:     "Slice value interpolation",
			rawValue: "${inputs:sliceInput[1]}",
			data: DynamicValueData{
				currentOwner: mockConstruct,
			},
			expected: "b",
		},
		{
			name:     "Resource property interpolation",
			rawValue: "${resources:testResource.prop1}",
			data: DynamicValueData{
				currentOwner: mockConstruct,
			},
			expected: "value1",
		},
		{
			name:     "Struct field interpolation",
			rawValue: "${inputs:structInput.Field1}",
			data: DynamicValueData{
				currentOwner: mockConstruct,
			},
			expected: "Hello",
		},
		{
			name:     "Struct interpolation",
			rawValue: "${inputs:structInput}",
			data: DynamicValueData{
				currentOwner: mockConstruct,
			},
			expected: simpleStruct,
		},
		{
			name:     "Mixed string interpolation",
			rawValue: "Prefix ${inputs:stringInput} Suffix",
			data: DynamicValueData{
				currentOwner: mockConstruct,
			},
			expected: "Prefix Hello Suffix",
		},
		{
			name:     "IaC reference interpolation",
			rawValue: "${resources:testResource#prop1}",
			data: DynamicValueData{
				currentOwner: mockConstruct,
			},
			expected: template.ResourceRef{
				ResourceKey:  "testResource",
				Property:     "prop1",
				Type:         template.ResourceRefTypeIaC,
				ConstructURN: model.URN{ResourceID: "test-construct"},
			},
		},
		{
			name:     "Invalid interpolation prefix",
			rawValue: "${invalid:key}",
			data: DynamicValueData{
				currentOwner: mockConstruct,
			},
			hasError: true,
		},
		{
			name: "Struct field interpolation",
			rawValue: struct {
				Field1 string
				Field2 int
			}{
				Field1: "${inputs:stringInput}",
				Field2: 42,
			},
			data: DynamicValueData{
				currentOwner: mockConstruct,
			},
			expected: struct {
				Field1 string
				Field2 int
			}{
				Field1: "Hello",
				Field2: 42,
			},
		},
		{
			name: "Map entry interpolation",
			rawValue: map[string]any{
				"key1":                  "${inputs:stringInput}",
				"key2":                  42,
				"${inputs:stringInput}": "value3",
			},
			data: DynamicValueData{
				currentOwner: mockConstruct,
			},
			expected: map[string]any{
				"key1":  "Hello",
				"key2":  42,
				"Hello": "value3",
			},
		},
		{
			name:     "Slice interpolation",
			rawValue: []any{"${inputs:stringInput}", 42},
			data: DynamicValueData{
				currentOwner: mockConstruct,
			},
			expected: []any{"Hello", 42},
		},
		{
			name:     "ResourceRef interpolation",
			rawValue: template.ResourceRef{ResourceKey: "testResource", Property: "prop1", Type: template.ResourceRefTypeInterpolated},
			data: DynamicValueData{
				currentOwner: mockConstruct,
			},
			expected: "testResource",
		},
		{
			name:     "ResourceRef template type",
			rawValue: template.ResourceRef{ResourceKey: "testResource", Property: "prop1", Type: template.ResourceRefTypeTemplate},
			data: DynamicValueData{
				currentOwner: mockConstruct,
			},
			expected: template.ResourceRef{ResourceKey: "testResource", Property: "prop1", Type: template.ResourceRefTypeTemplate, ConstructURN: model.URN{ResourceID: "test-construct"}},
		},
		{
			name:     "Nested map interpolation",
			rawValue: map[string]any{"outer": map[string]any{"inner": "${inputs:stringInput}"}},
			data: DynamicValueData{
				currentOwner: mockConstruct,
			},
			expected: map[string]any{"outer": map[string]any{"inner": "Hello"}},
		},
		{
			name:     "Nested slice interpolation",
			rawValue: []any{"${inputs:stringInput}", []any{"nested", "${inputs:intInput}"}},
			data: DynamicValueData{
				currentOwner: mockConstruct,
			},
			expected: []any{"Hello", []any{"nested", 42}},
		},
		{
			name:     "Non-existent input",
			rawValue: "${inputs:nonExistentInput}",
			data: DynamicValueData{
				currentOwner: mockConstruct,
			},
			expected: nil,
		},
		{
			name:     "Non-existent resource",
			rawValue: "${resources:nonExistentResource.prop}",
			data: DynamicValueData{
				currentOwner: mockConstruct,
			},
			hasError: false,
		},
		{
			name:     "Non-existent resource property",
			rawValue: "${resources:testResource.nonExistentProp}",
			data: DynamicValueData{
				currentOwner: mockConstruct,
			},
			expected: nil,
		},
		{
			name:     "Invalid array index",
			rawValue: "${inputs:sliceInput[invalid]}",
			data: DynamicValueData{
				currentOwner: mockConstruct,
			},
			expected: nil,
		},
		{
			name:     "Out of bounds array index",
			rawValue: "${inputs:sliceInput[10]}",
			data: DynamicValueData{
				currentOwner: mockConstruct,
			},
			expected: nil,
		},
		{
			name:     "Multiple interpolations in a string",
			rawValue: "Start ${inputs:stringInput} middle ${inputs:intInput} end",
			data: DynamicValueData{
				currentOwner: mockConstruct,
			},
			expected: "Start Hello middle 42 end",
		},
		{
			name:     "Mixed string interpolation with slice interpolation",
			rawValue: "${inputs:stringInput} ${inputs:sliceInput}",
			data: DynamicValueData{
				currentOwner: mockConstruct,
			},
			expected: "Hello [a b c]",
		},
		{
			name:     "Mixed string interpolation with map interpolation",
			rawValue: "${inputs:stringInput} ${inputs:mapInput}",
			data: DynamicValueData{
				currentOwner: mockConstruct,
			},
			// the dynamic key has not been interpolated (that would occur in a separate step)
			expected: "Hello map[${inputs:stringInput}:value3 key1:value1 key2:2]",
		},
		{
			name:     "Mixed string interpolation with struct interpolation",
			rawValue: "${inputs:stringInput} ${inputs:stringerStructInput}",
			data: DynamicValueData{
				currentOwner: mockConstruct,
			},
			expected: "Hello Hello",
		},
		{
			name:     "Go template interpolation",
			rawValue: "Hello {{ .Inputs.stringInput }}",
			data:     DynamicValueData{currentOwner: mockConstruct},
			expected: "Hello Hello",
		},
		{
			name:     "Go template evaluated before interpolation string",
			rawValue: `Hello ${inputs:{{"stringInput"}}}`,
			data:     DynamicValueData{currentOwner: mockConstruct},
			expected: "Hello Hello",
		},
		{
			name:     "Interleaved Go template and interpolation",
			rawValue: "Hello {{ .Inputs.stringInput }} ${inputs:stringInput}",
			data:     DynamicValueData{currentOwner: mockConstruct},
			expected: "Hello Hello Hello",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ce := &ConstructEvaluator{}
			result, err := ce.interpolateValue(&tt.data, tt.rawValue)

			if tt.hasError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestGetValueFromSource(t *testing.T) {
	tests := []struct {
		name     string
		source   any
		key      string
		flat     bool
		expected any
		err      string
	}{
		{
			name: "Simple map access",
			source: map[string]any{
				"foo": "bar",
			},
			key:      "foo",
			expected: "bar",
		},
		{
			name: "Nested map access",
			source: map[string]any{
				"foo": map[string]any{
					"bar": "baz",
				},
			},
			key:      "foo.bar",
			expected: "baz",
		},
		{
			name: "Array access",
			source: map[string]any{
				"foo": []any{"bar", "baz"},
			},
			key:      "foo[1]",
			expected: "baz",
		},
		{
			name: "Mixed map and array access",
			source: map[string]any{
				"foo": []any{
					map[string]any{"bar": "baz"},
					map[string]any{"qux": "quux"},
				},
			},
			key:      "foo[1].qux",
			expected: "quux",
		},
		{
			name: "Resource property implicit access",
			source: map[string]any{"foo": &Resource{
				Properties: map[string]any{
					"bar": "baz",
				},
			}},
			key:      "foo.bar",
			expected: "baz",
		},
		{
			name: "Resource property explicit access",
			source: map[string]any{"foo": &Resource{
				Properties: map[string]any{
					"bar": "baz",
				},
			}},
			key: "foo.Properties.bar",
			// Explicit access is not supported, so we expect an error
			err: "could not get value for key: Properties.bar",
		},
		{
			name: "Flat key access",
			source: map[string]any{
				"foo.bar": "baz",
				"foo":     map[string]any{"bar": "other"},
			},
			key:      "foo.bar",
			flat:     true,
			expected: "baz",
		},
		{
			name: "Invalid key",
			source: map[string]any{
				"foo": "bar",
			},
			key: "baz",
			err: "could not get value for key: baz",
		},
		{
			name: "Invalid array index",
			source: map[string]any{
				"foo": []any{"bar"},
			},
			key: "foo[1]",
			err: "index out of bounds: 1",
		},
		{
			name: "Invalid type for array access",
			source: map[string]any{
				"foo": "bar",
			},
			key: "foo[0]",
			err: "invalid type: string",
		},
		{
			name: "Resource",
			source: map[string]any{
				"foo": &Resource{
					Id: construct.ResourceId{
						Provider: "test",
						Type:     "resource",
						Name:     "foo",
					},
				},
			},
			key: "foo",
			expected: &Resource{
				Id: construct.ResourceId{
					Provider: "test",
					Type:     "resource",
					Name:     "foo",
				},
			},
		},
		{
			name: "Resource with IaC reference",
			source: map[string]any{
				"foo": &Resource{
					Id: construct.ResourceId{
						Provider: "test",
						Type:     "resource",
						Name:     "foo",
					},
				},
			},
			key: "foo#bar",
			// IaC resource references aren't resolved in this function, so property suffix is dropped and we just return the resource for further processing
			expected: &Resource{
				Id: construct.ResourceId{
					Provider: "test",
					Type:     "resource",
					Name:     "foo",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ce, err := NewConstructEvaluator(nil, nil)
			if !assert.NoError(t, err) {
				return
			}
			result, err := ce.getValueFromSource(tt.source, tt.key, tt.flat)

			if tt.err != "" {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.err)
			} else {
				assert.NoError(t, err)
				assert.True(t, reflect.DeepEqual(result, tt.expected), "Expected %v, but got %v", tt.expected, result)
			}
		})
	}
}

func TestNewConstruct(t *testing.T) {
	tests := []struct {
		name         string
		urn          string
		inputs       map[string]any
		expectedErr  bool
		expectedName string
	}{
		{
			name:         "Valid inputs",
			urn:          "urn:accountid:project:dev::construct/klotho.aws.Bucket:my-bucket",
			inputs:       map[string]any{"someKey": "someValue"},
			expectedErr:  false,
			expectedName: "my-bucket",
		},
		{
			name:        "Reserved Name key",
			urn:         "urn:accountid:project:dev::construct/klotho.aws.Bucket:my-bucket",
			inputs:      map[string]any{"Name": "invalid"},
			expectedErr: true,
		},
		{
			name:        "Invalid URN type",
			urn:         "urn:accountid:project:dev::resource/klotho.aws.Bucket:invalidType",
			inputs:      map[string]any{"someKey": "someValue"},
			expectedErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			constructURN, err := model.ParseURN(tt.urn)
			assert.NoError(t, err)

			ce, err := NewConstructEvaluator(nil, nil)
			if !assert.NoError(t, err) {
				return
			}

			c, err := ce.newConstruct(*constructURN, tt.inputs)
			if tt.expectedErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedName, c.Inputs["Name"])
			}
		})
	}
}
