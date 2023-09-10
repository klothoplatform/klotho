package knowledgebase

import (
	"testing"

	"github.com/klothoplatform/klotho/pkg/construct"
	"github.com/stretchr/testify/assert"
)

func TestConfigurationContext_Upstream(t *testing.T) {
	dag := construct.NewResourceGraph()
	testResource := &SimpleResource{Name: "test"}
	dag.AddDependency(&SimpleResource{Type: "a", Name: "a-upstream"}, testResource)
	dag.AddDependency(testResource, &SimpleResource{Type: "b", Name: "b-downstream"})

	ctx := &ConfigurationContext{
		dag:      dag,
		resource: testResource,
	}

	tests := []struct {
		name    string
		text    string
		want    string
		wantErr bool
	}{
		{
			name: "hit",
			text: "test:a",
			want: "test:a:a-upstream",
		},
		{
			name:    "miss not upstream",
			text:    "test:b",
			wantErr: true,
		},
		{
			name:    "miss not existing",
			text:    "test:resource:c",
			wantErr: true,
		},
		{
			name: "self is an upstream",
			text: "test:resource",
			want: "test:resource:test",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)

			got, err := ctx.Upstream(tt.text)
			if tt.wantErr {
				assert.Error(err, "got result %v", got)
				return
			}
			if !assert.NoError(err) {
				return
			}
			assert.Equal(tt.want, got)
		})
	}
}

func TestConfigurationContext_Downstream(t *testing.T) {
	ctx := &ConfigurationContext{
		dag:      construct.NewResourceGraph(),
		resource: &SimpleResource{Name: "test"},
	}
	ctx.dag.AddDependency(&SimpleResource{Type: "a", Name: "a-upstream"}, ctx.resource)
	ctx.dag.AddDependency(ctx.resource, &SimpleResource{Type: "b", Name: "b-downstream"})

	tests := []struct {
		name    string
		text    string
		want    string
		wantErr bool
	}{
		{
			name: "hit",
			text: "test:b",
			want: "test:b:b-downstream",
		},
		{
			name:    "miss not downstream",
			text:    "test:a",
			wantErr: true,
		},
		{
			name:    "miss not existing",
			text:    "test:resource:c",
			wantErr: true,
		},
		{
			name: "self is a downstream",
			text: "test:resource",
			want: "test:resource:test",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)

			got, err := ctx.Downstream(tt.text)
			if tt.wantErr {
				assert.Error(err, "got result %v", got)
				return
			}
			if !assert.NoError(err) {
				return
			}
			assert.Equal(tt.want, got)
		})
	}
}

func TestConfigurationContext_Split(t *testing.T) {
	tests := []struct {
		name    string
		delim   string
		value   string
		want    string
		wantErr bool
	}{
		{
			name:  "empty",
			delim: "/",
			value: "",
			want:  "[]",
		},
		{
			name:  "single",
			delim: "/",
			value: "a",
			want:  `["a"]`,
		},
		{
			name:  "multiple",
			delim: "/",
			value: "a/b/c",
			want:  `["a","b","c"]`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)

			ctx := &ConfigurationContext{}
			got, err := ctx.Split(tt.delim, tt.value)
			if tt.wantErr {
				assert.Error(err, "got result %v", got)
				return
			}
			if !assert.NoError(err) {
				return
			}
			assert.Equal(tt.want, got)
		})
	}
}

func TestConfigurationContext_FilterMatch(t *testing.T) {
	value := `[":valueA","something",":valueB"]`
	tests := []struct {
		name    string
		pattern string
		value   string
		want    string
		wantErr bool
	}{
		{
			name:    "empty pattern",
			pattern: "",
			value:   value,
			want:    value,
		},
		{
			name:    "empty value",
			pattern: "a",
			value:   "",
			want:    "[]",
		},
		{
			name:    "match",
			pattern: `^:\w*$`,
			value:   value,
			want:    `[":valueA",":valueB"]`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)

			ctx := &ConfigurationContext{}
			got, err := ctx.FilterMatch(tt.pattern, tt.value)
			if tt.wantErr {
				assert.Error(err, "got result %v", got)
				return
			}
			if !assert.NoError(err) {
				return
			}
			assert.Equal(tt.want, got)
		})
	}
}

func TestConfigurationContext_MapString(t *testing.T) {
	value := `["a","b","c"]`
	tests := []struct {
		name    string
		pattern string
		replace string
		value   string
		want    string
		wantErr bool
	}{
		{
			name:    "empty pattern",
			pattern: "",
			replace: "",
			value:   value,
			want:    value,
		},
		{
			name:    "empty value",
			pattern: "a",
			replace: "z",
			value:   "",
			want:    "[]",
		},
		{
			name:    "replace",
			pattern: `a`,
			replace: "z",
			value:   value,
			want:    `["z","b","c"]`,
		},
		{
			name:    "replace RE",
			pattern: `^(.)$`,
			replace: "$1$1",
			value:   value,
			want:    `["aa","bb","cc"]`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)

			ctx := &ConfigurationContext{}
			got, err := ctx.MapString(tt.pattern, tt.replace, tt.value)
			if tt.wantErr {
				assert.Error(err, "got result %v", got)
				return
			}
			if !assert.NoError(err) {
				return
			}
			assert.Equal(tt.want, got)
		})
	}
}

func TestConfigurationContext_ZipToMap(t *testing.T) {
	tests := []struct {
		name    string
		keys    string
		values  string
		want    string
		wantErr bool
	}{
		{
			name:   "empty",
			keys:   "",
			values: "",
			want:   "{}",
		},
		{
			name:   "simple",
			keys:   `["a","b","c"]`,
			values: `["1","2","3"]`,
			want:   `{"a":"1","b":"2","c":"3"}`,
		},
		{
			name:    "mismatch",
			keys:    `["a","b","c"]`,
			values:  `["1","2"]`,
			wantErr: true,
		},
		{
			name:    "non-string keys",
			keys:    `[1,2,3]`,
			wantErr: true,
		},
		{
			name:   "non-string values",
			keys:   `["a","b","c"]`,
			values: `[1,2,3]`,
			want:   `{"a":1,"b":2,"c":3}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)

			ctx := &ConfigurationContext{}
			got, err := ctx.ZipToMap(tt.keys, tt.values)
			if tt.wantErr {
				assert.Error(err, "got result %v", got)
				return
			}
			if !assert.NoError(err) {
				return
			}
			assert.Equal(tt.want, got)
		})
	}
}

func TestConfigurationContext_KeysToMapWithDefault(t *testing.T) {
	tests := []struct {
		name    string
		keys    string
		value   string
		want    string
		wantErr bool
	}{
		{
			name:  "empty",
			keys:  "",
			value: "",
			want:  "{}",
		},
		{
			name:  "simple string",
			keys:  `["a","b","c"]`,
			value: `"1"`,
			want:  `{"a":"1","b":"1","c":"1"}`,
		},
		{
			name:  "simple number",
			keys:  `["a","b","c"]`,
			value: `0`,
			want:  `{"a":0,"b":0,"c":0}`,
		},
		{
			name:  "simple bool",
			keys:  `["a","b","c"]`,
			value: `false`,
			want:  `{"a":false,"b":false,"c":false}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)

			ctx := &ConfigurationContext{}
			got, err := ctx.KeysToMapWithDefault(tt.value, tt.keys)
			if tt.wantErr {
				assert.Error(err, "got result %v", got)
				return
			}
			if !assert.NoError(err) {
				return
			}
			assert.Equal(tt.want, got)
		})
	}
}
