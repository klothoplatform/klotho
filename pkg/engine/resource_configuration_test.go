package engine

import (
	"reflect"
	"testing"

	"github.com/klothoplatform/klotho/pkg/construct"
	"github.com/klothoplatform/klotho/pkg/engine/enginetesting"
	"github.com/stretchr/testify/assert"
)

func Test_parseFieldName(t *testing.T) {
	tests := []struct {
		name      string
		resource  *enginetesting.MockResource6
		fieldName string
		resources []construct.Resource
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
			want:      map[string]construct.IaCValue(nil),
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
				Map1: map[string]construct.IaCValue{"key1": {Property: "value1"}},
			},
			fieldName: "Map1[key1]",
			want:      construct.IaCValue{Property: "value1"},
			mapKey: &SetMapKey{
				Map:   reflect.ValueOf(map[string]construct.IaCValue{"key1": {Property: "value1"}}),
				Key:   reflect.ValueOf("key1"),
				Value: reflect.ValueOf(construct.IaCValue{Property: "value1"}),
			},
		},
		{
			name: "map sub field index does not exist",
			resource: &enginetesting.MockResource6{
				Map1: map[string]construct.IaCValue(nil),
			},
			fieldName: "Map1[key1]",
			want:      construct.IaCValue{},
			mapKey: &SetMapKey{
				Map:   reflect.ValueOf(map[string]construct.IaCValue{}),
				Key:   reflect.ValueOf("key1"),
				Value: reflect.ValueOf(construct.IaCValue{}),
			},
		},
		{
			name:      "map sub field, nil map",
			resource:  &enginetesting.MockResource6{},
			fieldName: "Map1[key1]",
			want:      construct.IaCValue{},
			mapKey: &SetMapKey{
				Map:   reflect.ValueOf(map[string]construct.IaCValue{}),
				Key:   reflect.ValueOf("key1"),
				Value: reflect.ValueOf(construct.IaCValue{}),
			},
		},
		{
			name: "map value's property",
			resource: &enginetesting.MockResource6{
				Map1: map[string]construct.IaCValue{"key1": {Property: "value1"}},
			},
			fieldName: "Map1[key1].Property",
			want:      "value1",
			mapKey: &SetMapKey{
				Map:   reflect.ValueOf(map[string]construct.IaCValue{"key1": {Property: "value1"}}),
				Key:   reflect.ValueOf("key1"),
				Value: reflect.ValueOf(construct.IaCValue{Property: "value1"}),
			},
		},
		{
			name: "map with resource id string key",
			resource: &enginetesting.MockResource6{
				Map1: map[string]construct.IaCValue{"test": {Property: "value1"}},
			},
			fieldName: "Map1[mock:mock6:test#Name]",
			want:      construct.IaCValue{Property: "value1"},
			mapKey: &SetMapKey{
				Map:   reflect.ValueOf(map[string]construct.IaCValue{"test": {Property: "value1"}}),
				Key:   reflect.ValueOf("test"),
				Value: reflect.ValueOf(construct.IaCValue{Property: "value1"}),
			},
			resources: []construct.Resource{
				&enginetesting.MockResource6{Name: "test"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			dag := construct.NewResourceGraph()
			for _, res := range tt.resources {
				dag.AddResource(res)
			}
			value, setMapKey, err := parseFieldName(tt.resource, tt.fieldName, dag, true)
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
