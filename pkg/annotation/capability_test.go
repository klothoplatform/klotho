package annotation

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseCapability(t *testing.T) {
	tests := []struct {
		name    string
		text    string
		want    []*Capability
		wantErr bool
	}{
		{
			name: "no directives",
			text: "@klotho::thing",
			want: []*Capability{{
				Name: "thing",
			}},
		},
		{
			name: "no directives empty block",
			text: `@klotho::thing {
	}`,
			want: []*Capability{{
				Name: "thing",
			}},
		},
		{
			name: "no match",
			text: "some comment",
			want: nil,
		},
		{
			name: "one directive",
			text: `@klotho::thing {
		key1 = "value1"
	}`,
			want: []*Capability{{
				Name: "thing",
				Directives: Directives{
					"key1": "value1",
				},
			}},
		},
		{
			name: "one directive with extra",
			text: `@klotho::thing {
		key1 = "value1"
	}
	some other comment`,
			want: []*Capability{{
				Name: "thing",
				Directives: Directives{
					"key1": "value1",
				},
			}},
		},
		{
			name: "oneline with directive",
			text: `@klotho::thing { key1 = "value1" }`,
			want: []*Capability{{
				Name: "thing",
				Directives: Directives{
					"key1": "value1",
				},
			}},
		},
		{
			name: "multiple string directives",
			text: `@klotho::thing {
		key1 = "value1"
		key2 = "value2"
		key3 = "value3"
	}`,
			want: []*Capability{{
				Name: "thing",
				Directives: Directives{
					"key1": "value1",
					"key2": "value2",
					"key3": "value3",
				},
			},
			}},
		{
			name: "boolean directive",
			text: `@klotho::thing {
		key1 = true
	}`,
			want: []*Capability{{
				Name: "thing",
				Directives: Directives{
					"key1": true,
				},
			}},
		},
		{
			name: "number directive",
			text: `@klotho::thing {
		key1 = 1234
	}`,
			want: []*Capability{{
				Name: "thing",
				Directives: Directives{
					"key1": int64(1234),
				},
			}},
		},
		{
			name: "map directive",
			text: `
			@klotho::thing {
			  [key1]
			  a = 1
			  b = 2
			}`,
			want: []*Capability{{
				Name: "thing",
				Directives: Directives{
					"key1": map[string]interface{}{"a": int64(1), "b": int64(2)},
				},
			}},
		},
		{
			name: "inline map directive",
			text: `
			@klotho::thing {
			  map1 = {a = 1, map2 = {b = [1, 2, 3]}}
			}`,
			want: []*Capability{{
				Name: "thing",
				Directives: Directives{
					"map1": map[string]interface{}{
						"a": int64(1),
						"map2": map[string]interface{}{
							"b": []interface{}{
								int64(1), int64(2), int64(3),
							}}},
				},
			}},
		},
		{
			name: "multiple capabilities",
			text: `
			@klotho::thing1 {
			  id = "thing1"
			  directive = "val1"
			  inline = {key = "val"}
			}
			@klotho::thing2 {
			  id = "thing2"
			  directive = "val2"
			}`,
			want: []*Capability{
				{
					Name: "thing1",
					ID:   "thing1",
					Directives: Directives{
						"directive": "val1",
						"id":        "thing1",
						"inline":    map[string]interface{}{"key": "val"},
					},
				},
				{
					Name: "thing2",
					ID:   "thing2",
					Directives: Directives{
						"directive": "val2",
						"id":        "thing2",
					},
				}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)

			got, err := ParseCapabilities(tt.text)
			if tt.wantErr {
				assert.Error(err)
			} else if assert.NoError(err) {
				assert.Equal(tt.want, got)
			}
		})
	}
}
