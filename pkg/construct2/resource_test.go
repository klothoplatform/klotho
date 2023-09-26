package construct2

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_splitPath(t *testing.T) {
	tests := []struct {
		name string
		path string
		want []string
	}{
		{
			name: "empty",
			path: "",
			want: nil,
		},
		{
			name: "single",
			path: "foo",
			want: []string{"foo"},
		},
		{
			name: "dotted",
			path: "foo.bar",
			want: []string{"foo", ".bar"},
		},
		{
			name: "bracketed",
			path: "foo[bar]",
			want: []string{"foo", "[bar]"},
		},
		{
			name: "long mixed",
			path: "foo.bar[baz].qux",
			want: []string{"foo", ".bar", "[baz]", ".qux"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)

			got := splitPath(tt.path)
			assert.Equal(tt.want, got)
		})
	}
}

func TestResource_propertyValue(t *testing.T) {
	tests := []struct {
		name    string
		props   Properties
		path    string
		want    any
		wantErr bool
	}{
		{
			name:  "top-level field",
			props: Properties{"A": "foo"},
			path:  "A",
			want:  "foo",
		},
		{
			name:  "nested field",
			props: Properties{"A": Properties{"B": "foo"}},
			path:  "A.B",
			want:  "foo",
		},
		{
			name:  "array index",
			props: Properties{"A": []any{"foo", "bar"}},
			path:  "A[1]",
			want:  "bar",
		},
		{
			name:  "array index nested",
			props: Properties{"A": []any{"foo", Properties{"B": "bar"}}},
			path:  "A[1].B",
			want:  "bar",
		},
		{
			name:    "array index on map",
			props:   Properties{"A": Properties{"B": "foo"}},
			path:    "A[0]",
			wantErr: true,
		},
		{
			name:    "map key on array",
			props:   Properties{"A": []any{"foo", "bar"}},
			path:    "A.B",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)

			r := &Resource{Properties: tt.props}

			got, err := r.propertyValue(splitPath(tt.path))
			if tt.wantErr {
				assert.Error(err)
				return
			}
			if !assert.NoError(err) {
				return
			}
			assert.Equal(tt.want, got)
		})
	}
}
