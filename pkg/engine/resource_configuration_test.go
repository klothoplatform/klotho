package engine

import (
	"reflect"
	"testing"

	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/engine/enginetesting"
	"github.com/stretchr/testify/assert"
)

func Test_ConfigureField(t *testing.T) {
	tests := []struct {
		name     string
		resource *enginetesting.MockResource6
		config   core.Configuration
		want     *enginetesting.MockResource6
	}{
		// {
		// 	name:     "simple int",
		// 	resource: &enginetesting.MockResource6{},
		// 	config: core.Configuration{
		// 		Field: "Field1",
		// 		Value: 1,
		// 	},
		// 	want: &enginetesting.MockResource6{
		// 		Field1: 1,
		// 	},
		// },
		// {
		// 	name:     "simple string",
		// 	resource: &enginetesting.MockResource6{},
		// 	config: core.Configuration{
		// 		Field: "Field2",
		// 		Value: "two",
		// 	},
		// 	want: &enginetesting.MockResource6{
		// 		Field2: "two",
		// 	},
		// },
		// {
		// 	name:     "simple bool",
		// 	resource: &enginetesting.MockResource6{},
		// 	config: core.Configuration{
		// 		Field: "Field3",
		// 		Value: true,
		// 	},
		// 	want: &enginetesting.MockResource6{
		// 		Field3: true,
		// 	},
		// },
		// {
		// 	name:     "simple array",
		// 	resource: &enginetesting.MockResource6{},
		// 	config: core.Configuration{
		// 		Field: "Arr1",
		// 		Value: []string{"1", "2", "3"},
		// 	},
		// 	want: &enginetesting.MockResource6{
		// 		Arr1: []string{"1", "2", "3"},
		// 	},
		// },
		// {
		// 	name:     "struct array",
		// 	resource: &enginetesting.MockResource6{},
		// 	config: core.Configuration{
		// 		Field: "Arr2",
		// 		Value: []map[string]interface{}{
		// 			{
		// 				"Field1": 1,
		// 				"Field2": "two",
		// 				"Field3": true,
		// 			},
		// 			{
		// 				"Field1": 2,
		// 				"Field2": "three",
		// 				"Field3": false,
		// 			},
		// 		},
		// 	},
		// 	want: &enginetesting.MockResource6{
		// 		Arr2: []enginetesting.TestRes1{
		// 			{
		// 				Field1: 1,
		// 				Field2: "two",
		// 				Field3: true,
		// 			},
		// 			{
		// 				Field1: 2,
		// 				Field2: "three",
		// 				Field3: false,
		// 			},
		// 		},
		// 	},
		// },
		// {
		// 	name:     "pointer array",
		// 	resource: &enginetesting.MockResource6{},
		// 	config: core.Configuration{
		// 		Field: "Arr3",
		// 		Value: []map[string]interface{}{
		// 			{
		// 				"Field1": 1,
		// 				"Field2": "two",
		// 				"Field3": true,
		// 			},
		// 			{
		// 				"Field1": 2,
		// 				"Field2": "three",
		// 				"Field3": false,
		// 			},
		// 		},
		// 	},
		// 	want: &enginetesting.MockResource6{
		// 		Arr3: []*enginetesting.TestRes1{
		// 			{
		// 				Field1: 1,
		// 				Field2: "two",
		// 				Field3: true,
		// 			},
		// 			{
		// 				Field1: 2,
		// 				Field2: "three",
		// 				Field3: false,
		// 			},
		// 		},
		// 	},
		// },
		// {
		// 	name:     "struct",
		// 	resource: &enginetesting.MockResource6{},
		// 	config: core.Configuration{
		// 		Field: "Struct1", Value: map[string]interface{}{
		// 			"Field1": 1,
		// 			"Field2": "two",
		// 			"Field3": true,
		// 			"Arr1":   []string{"1", "2", "3"},
		// 		},
		// 	},
		// 	want: &enginetesting.MockResource6{
		// 		Struct1: enginetesting.TestRes1{
		// 			Field1: 1,
		// 			Field2: "two",
		// 			Field3: true,
		// 			Arr1:   []string{"1", "2", "3"},
		// 		},
		// 	},
		// },
		// {
		// 	name:     "pointer",
		// 	resource: &enginetesting.MockResource6{},
		// 	config: core.Configuration{
		// 		Field: "Struct2", Value: map[string]interface{}{
		// 			"Field1": 1,
		// 			"Field2": "two",
		// 			"Field3": true,
		// 			"Arr1":   []string{"1", "2", "3"},
		// 		},
		// 	},
		// 	want: &enginetesting.MockResource6{
		// 		Struct2: &enginetesting.TestRes1{
		// 			Field1: 1,
		// 			Field2: "two",
		// 			Field3: true,
		// 			Arr1:   []string{"1", "2", "3"},
		// 		},
		// 	},
		// },
		// {
		// 	name:     "pointer sub field",
		// 	resource: &enginetesting.MockResource6{},
		// 	config: core.Configuration{
		// 		Field: "Struct2.Field1", Value: 3,
		// 	},
		// 	want: &enginetesting.MockResource6{
		// 		Struct2: &enginetesting.TestRes1{
		// 			Field1: 3,
		// 		},
		// 	},
		// },
		// {
		// 	name:     "struct sub field",
		// 	resource: &enginetesting.MockResource6{},
		// 	config: core.Configuration{
		// 		Field: "Struct1.Field1", Value: 4,
		// 	},
		// 	want: &enginetesting.MockResource6{
		// 		Struct1: enginetesting.TestRes1{
		// 			Field1: 4,
		// 		},
		// 	},
		// },
		// {
		// 	name: "appends to array",
		// 	resource: &enginetesting.MockResource6{
		// 		Arr2: []enginetesting.TestRes1{
		// 			{
		// 				Field1: 1,
		// 				Field2: "two",
		// 				Field3: true,
		// 			},
		// 		},
		// 	},
		// 	config: core.Configuration{
		// 		Field: "Arr2", Value: []map[string]interface{}{
		// 			{
		// 				"Field1": 2,
		// 				"Field2": "three",
		// 				"Field3": false,
		// 			},
		// 		},
		// 	},
		// 	want: &enginetesting.MockResource6{
		// 		Arr2: []enginetesting.TestRes1{{Field1: 1, Field2: "two", Field3: true}, {Field1: 2, Field2: "three", Field3: false}},
		// 	},
		// },
		{
			name: "does not append duplicates to array",
			resource: &enginetesting.MockResource6{
				Arr2: []enginetesting.TestRes1{
					{
						Field1: 1,
						Field2: "two",
						Field3: true,
					},
				},
			},
			config: core.Configuration{
				Field: "Arr2", Value: []map[string]interface{}{
					{
						"Field1": 1,
						"Field2": "two",
						"Field3": true,
					},
				},
			},
			want: &enginetesting.MockResource6{
				Arr2: []enginetesting.TestRes1{{Field1: 1, Field2: "two", Field3: true}},
			},
		},
		{
			name: "does not append duplicates to array",
			resource: &enginetesting.MockResource6{
				Arr1: []string{"1", "2", "3"},
			},
			config: core.Configuration{
				Field: "Arr1", Value: []string{"1", "2", "3", "4"},
			},
			want: &enginetesting.MockResource6{
				Arr1: []string{"1", "2", "3", "4"},
			},
		},
		// {
		// 	name: "overwrites map key",
		// 	resource: &enginetesting.MockResource6{
		// 		Map1: map[string]core.IaCValue{
		// 			"key1": {ResourceId: core.ResourceId{Name: "a"}, Property: "value1"},
		// 		},
		// 	},
		// 	config: core.Configuration{
		// 		Field: "Map1[key1]", Value: map[string]interface{}{
		// 			"Property": "value2",
		// 		},
		// 	},
		// 	want: &enginetesting.MockResource6{
		// 		Map1: map[string]core.IaCValue{
		// 			"key1": {Property: "value2", ResourceId: core.ResourceId{}},
		// 		},
		// 	},
		// },
		// {
		// 	name: "set field on map value",
		// 	resource: &enginetesting.MockResource6{
		// 		Map1: map[string]core.IaCValue{
		// 			"key1": {ResourceId: core.ResourceId{Name: "a"}, Property: "value1"},
		// 		},
		// 	},
		// 	config: core.Configuration{
		// 		Field: "Map1[key1].Property", Value: "value2",
		// 	},
		// 	want: &enginetesting.MockResource6{
		// 		Map1: map[string]core.IaCValue{
		// 			"key1": {ResourceId: core.ResourceId{Name: "a"}, Property: "value2"},
		// 		},
		// 	},
		// },
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)

			err := ConfigureField(tt.resource, tt.config.Field, tt.config.Value, tt.config.ZeroValueAllowed, nil)
			if !assert.NoError(err) {
				return
			}
			assert.Equal(tt.want, tt.resource)
		})
	}
}

func Test_parseFieldName(t *testing.T) {
	tests := []struct {
		name      string
		resource  *enginetesting.MockResource6
		fieldName string
		resources []core.Resource
		want      interface{}
		mapKey    *SetMapKey
		wantErr   bool
	}{
		{
			name:      "simple int",
			resource:  &enginetesting.MockResource6{},
			fieldName: "Field1",
			want:      0,
		},
		{
			name:      "simple string",
			resource:  &enginetesting.MockResource6{Field2: "test"},
			fieldName: "Field2",
			want:      "test",
		},
		{
			name:      "simple bool",
			resource:  &enginetesting.MockResource6{Field3: true},
			fieldName: "Field3",
			want:      true,
		},
		{
			name:      "struct",
			resource:  &enginetesting.MockResource6{},
			fieldName: "Struct1",
			want:      enginetesting.TestRes1{},
		},
		{
			name:      "pointer",
			resource:  &enginetesting.MockResource6{},
			fieldName: "Struct2",
			want:      &enginetesting.TestRes1{},
		},
		{
			name:      "array",
			resource:  &enginetesting.MockResource6{},
			fieldName: "Arr3",
			want:      []*enginetesting.TestRes1(nil),
		},
		{
			name:      "map",
			resource:  &enginetesting.MockResource6{},
			fieldName: "Map1",
			want:      map[string]core.IaCValue(nil),
		},
		{
			name:      "struct sub field",
			resource:  &enginetesting.MockResource6{},
			fieldName: "Struct1.Field1",
			want:      0,
		},
		{
			name:      "pointer sub field",
			resource:  &enginetesting.MockResource6{},
			fieldName: "Struct2.Field1",
			want:      0,
		},
		{
			name: "array sub field",
			resource: &enginetesting.MockResource6{
				Arr3: []*enginetesting.TestRes1{{Field1: 1}},
			},
			fieldName: "Arr3[0].Field1",
			want:      1,
		},
		{
			name:      "array sub field, length to short, error",
			resource:  &enginetesting.MockResource6{},
			fieldName: "Arr3[0].Field1",
			wantErr:   true,
		},
		{
			name: "map sub field",
			resource: &enginetesting.MockResource6{
				Map1: map[string]core.IaCValue{"key1": {Property: "value1"}},
			},
			fieldName: "Map1[key1]",
			want:      core.IaCValue{Property: "value1"},
			mapKey: &SetMapKey{
				Map:   reflect.ValueOf(map[string]core.IaCValue{"key1": {Property: "value1"}}),
				Key:   reflect.ValueOf("key1"),
				Value: reflect.ValueOf(core.IaCValue{Property: "value1"}),
			},
		},
		{
			name: "map sub field index does not exist",
			resource: &enginetesting.MockResource6{
				Map1: map[string]core.IaCValue(nil),
			},
			fieldName: "Map1[key1]",
			want:      core.IaCValue{},
			mapKey: &SetMapKey{
				Map:   reflect.ValueOf(map[string]core.IaCValue{}),
				Key:   reflect.ValueOf("key1"),
				Value: reflect.ValueOf(core.IaCValue{}),
			},
		},
		{
			name:      "map sub field, nil map",
			resource:  &enginetesting.MockResource6{},
			fieldName: "Map1[key1]",
			want:      core.IaCValue{},
			mapKey: &SetMapKey{
				Map:   reflect.ValueOf(map[string]core.IaCValue{}),
				Key:   reflect.ValueOf("key1"),
				Value: reflect.ValueOf(core.IaCValue{}),
			},
		},
		{
			name: "map value's property",
			resource: &enginetesting.MockResource6{
				Map1: map[string]core.IaCValue{"key1": {Property: "value1"}},
			},
			fieldName: "Map1[key1].Property",
			want:      "value1",
			mapKey: &SetMapKey{
				Map:   reflect.ValueOf(map[string]core.IaCValue{"key1": {Property: "value1"}}),
				Key:   reflect.ValueOf("key1"),
				Value: reflect.ValueOf(core.IaCValue{Property: "value1"}),
			},
		},
		{
			name: "map with resource id string key",
			resource: &enginetesting.MockResource6{
				Map1: map[string]core.IaCValue{"test": {Property: "value1"}},
			},
			fieldName: "Map1[mock:mock6:test#Name]",
			want:      core.IaCValue{Property: "value1"},
			mapKey: &SetMapKey{
				Map:   reflect.ValueOf(map[string]core.IaCValue{"test": {Property: "value1"}}),
				Key:   reflect.ValueOf("test"),
				Value: reflect.ValueOf(core.IaCValue{Property: "value1"}),
			},
			resources: []core.Resource{
				&enginetesting.MockResource6{Name: "test"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			dag := core.NewResourceGraph()
			for _, res := range tt.resources {
				dag.AddResource(res)
			}
			value, setMapKey, err := parseFieldName(tt.resource, tt.fieldName, dag)
			if tt.wantErr {
				assert.Error(err)
				return
			}
			if !assert.NoError(err) {
				return
			}
			assert.Equal(tt.want, value.Interface())
			if tt.mapKey != nil {
				if !assert.NotNil(setMapKey) {
					return
				}
				assert.Equal(tt.mapKey.Map.Interface(), setMapKey.Map.Interface())
				assert.Equal(tt.mapKey.Key.Interface(), setMapKey.Key.Interface())
				assert.Equal(tt.mapKey.Value.Interface(), setMapKey.Value.Interface())
			}
		})
	}
}
