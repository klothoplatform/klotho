package construct2

import (
	"strings"
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

func TestResource_PropertyPath(t *testing.T) {
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
		{
			name:    "no dot after array for map index",
			props:   Properties{"A": []any{"foo", Properties{"B": "bar"}}},
			path:    "A[1]B",
			wantErr: true,
		},
		{
			name:    "array index out of bounds",
			props:   Properties{"A": []any{"foo", "bar"}},
			path:    "A[10]",
			wantErr: true,
		},
		{
			name:    "array index not a number",
			props:   Properties{"A": []any{"foo", "bar"}},
			path:    "A[blah]",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)

			r := &Resource{Properties: tt.props}

			path, err := r.PropertyPath(tt.path)
			if tt.wantErr {
				assert.Error(err)
				return
			}
			if !assert.NoError(err) {
				return
			}
			assert.Equal(tt.want, path.Get())

			// Test the last item's itemToPath instead of the path's Parts
			// because this will test both functions (itemToPath and Parts)
			last := path[len(path)-1]
			pathParts := itemToPath(last)
			assert.Equal(tt.path, strings.Join(pathParts, ""))
		})
	}
}

func TestResource_PropertyPath_ops(t *testing.T) {
	assert := assert.New(t)

	r := &Resource{
		Properties: Properties{
			"A": map[string]any{
				"foo":   "bar",
				"array": []any{"fox", "bat", "dog"},
			},
			"B": []any{[]any{1, 2, 3}},
			"C": map[string]any{
				"x": "y",
			},
		},
	}

	path := func(s string) PropertyPath {
		p, err := r.PropertyPath(s)
		if !assert.NoError(err, "path %q", s) {
			t.Fail()
		}
		return p
	}

	foo := path("A.foo")
	assert.Equal("bar", foo.Get())
	if assert.NoError(foo.Set("baz")) {
		assert.Equal("baz", foo.Get())
	}
	assert.Error(foo.Append("value"))

	if assert.NoError(foo.Remove(nil)) {
		assert.Nil(foo.Get())
		m := path("A").Get().(map[string]any)
		assert.NotContains(m, "foo")
	}

	arr := path("A.array")
	if assert.NoError(arr.Append("cat")) {
		assert.Equal([]any{"fox", "bat", "dog", "cat"}, arr.Get())
	}
	if assert.NoError(arr.Remove("bat")) {
		assert.Equal([]any{"fox", "dog", "cat"}, arr.Get())
	}

	fox := path("A.array[0]")
	assert.Equal("fox", fox.Get())
	if assert.NoError(fox.Set("wolf")) {
		assert.Equal("wolf", fox.Get())
		assert.Equal([]any{"wolf", "dog", "cat"}, arr.Get())
	}
	if assert.NoError(fox.Remove(nil)) {
		assert.Equal([]any{"dog", "cat"}, arr.Get())
		assert.Equal("dog", fox.Get()) // [0] now points to "dog"
	}

	two := path("B[0][1]")
	assert.Equal(2, two.Get())
	if assert.NoError(two.Remove(nil)) {
		assert.Equal([]any{1, 3}, path("B[0]").Get())
	}

	c := path("C")
	if assert.NoError(c.Append(map[string]string{"hello": "world"})) {
		assert.Equal(map[string]any{
			"x":     "y",
			"hello": "world",
		}, c.Get())
	}

	d := path("D")
	if assert.NoError(d.Set(map[string]string{"hello": "world"})) {
		assert.Error(d.Append(map[string]int{"foo": 1}))
		assert.Error(d.Append(map[int]string{1: "foo"}))
	}

	e := path("E")
	if assert.NoError(e.Set([]string{"one", "two"})) {
		assert.NoError(e.Append([]string{"three", "four"}))
		assert.Equal([]string{"one", "two", "three", "four"}, e.Get())
	}

	tmp := path("temp")
	if assert.NoError(tmp.Append("test")) {
		assert.Equal([]string{"test"}, tmp.Get())
		assert.NoError(tmp.Remove(nil))
	}
	assert.Nil(tmp.Get())

	if assert.NoError(tmp.Append(map[string]string{"hello": "world"})) {
		assert.Equal(map[string]string{"hello": "world"}, tmp.Get())
		assert.NoError(tmp.Remove(nil))
	}

	nested := path("deeply.nested.value")
	assert.Nil(nested.Get())
	assert.Nil(path("deeply").Get())
	if assert.NoError(nested.Set("test")) {
		assert.Equal(map[string]interface{}(map[string]interface{}{"nested": map[string]interface{}{"value": "test"}}), path("deeply").Get())
	}
}

func TestResource_Properties_ops(t *testing.T) {
	assert := assert.New(t)

	r := &Resource{
		Properties: Properties{
			"A": map[string]any{
				"foo":   "bar",
				"array": []string{"fox", "bat", "dog"},
			},
			"B": []any{[]int{1, 2, 3}},
			"C": map[string]any{
				"x": "y",
			},
		},
	}

	get := func(s string) any {
		v, err := r.GetProperty(s)
		if !assert.NoError(err, "path %q", s) {
			t.Fail()
		}
		return v
	}

	assert.Equal("bar", get("A.foo"))
	if assert.NoError(r.SetProperty("A.foo", "baz")) {
		assert.Equal("baz", get("A.foo"))
	}
	assert.Error(r.AppendProperty("A.foo", "value"))

	if assert.NoError(r.RemoveProperty("A.foo", nil)) {
		assert.Nil(get("A.foo"))
		m := get("A").(map[string]any)
		assert.NotContains(m, "foo")
	}

	if assert.NoError(r.AppendProperty("A.array", "cat")) {
		assert.Equal([]string{"fox", "bat", "dog", "cat"}, get("A.array"))
	}
	if assert.NoError(r.RemoveProperty("A.array", "bat")) {
		assert.Equal([]string{"fox", "dog", "cat"}, get("A.array"))
	}

	assert.Equal("fox", get("A.array[0]"))
	if assert.NoError(r.SetProperty("A.array[0]", "wolf")) {
		assert.Equal("wolf", get("A.array[0]"))
		assert.Equal([]string{"wolf", "dog", "cat"}, get("A.array"))
	}
	if assert.NoError(r.RemoveProperty("A.array[0]", nil)) {
		assert.Equal([]string{"dog", "cat"}, get("A.array"))
		assert.Equal("dog", get("A.array[0]"))
	}

	assert.Equal(2, get("B[0][1]"))
	if assert.NoError(r.RemoveProperty("B[0][1]", nil)) {
		assert.Equal([]int{1, 3}, get("B[0]"))
	}

	if assert.NoError(r.AppendProperty("C", map[string]string{"hello": "world"})) {
		assert.Equal(map[string]any{
			"x":     "y",
			"hello": "world",
		}, get("C"))
	}
}
