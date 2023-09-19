package knowledgebase

import (
	"reflect"
	"testing"

	"github.com/klothoplatform/klotho/pkg/construct"
	"github.com/klothoplatform/klotho/pkg/graph"
	"github.com/stretchr/testify/assert"
)

func TestConfigTemplateContext_ExecuteDecode(t *testing.T) {
	ctx := &ConfigTemplateContext{DAG: construct.NewResourceGraph()}

	// Set up graph (using test:resource qualified type unless otherwise specified):
	// a -> b -> test:x:y -> c -> d
	//      b -> m -> c
	a := &SimpleResource{Name: "a"}
	b := &SimpleResource{Name: "b"}
	ctx.DAG.AddDependency(a, b)
	xy := &TestResource[string]{Type: "x", Name: "y", Field: "my value"}
	ctx.DAG.AddDependency(b, xy)
	c := &SimpleResource{Name: "c"}
	ctx.DAG.AddDependency(xy, c)
	d := &SimpleResource{Name: "d"}
	ctx.DAG.AddDependency(c, d)
	m := &SimpleResource{Name: "m"}
	ctx.DAG.AddDependency(b, m)
	ctx.DAG.AddDependency(m, c)

	// Simple data is a convenience for specify the data similar to an edge configuration data
	// of xy -> c setting a property on xy.
	simpleData := ConfigTemplateData{
		Resource: xy.Id(),
		Edge:     graph.Edge[construct.ResourceId]{Source: xy.Id(), Destination: c.Id()},
	}

	tests := []struct {
		name    string
		tmpl    string
		data    ConfigTemplateData
		want    any
		wantErr bool
	}{
		{
			name: "string literal",
			tmpl: "value",
			want: "value",
		},
		{
			name: "bool literal",
			tmpl: "true",
			want: true,
		},
		{
			name: "ResourceId literal",
			tmpl: "test:resource:a",
			want: a.Id(),
		},
		{
			name: "IaCValue literal",
			tmpl: "test:resource:a#Property",
			want: construct.IaCValue{ResourceId: a.Id(), Property: "Property"},
		},

		// Data "field" access
		{
			name: "ResourceId Self",
			tmpl: `{{ .Self }}`,
			data: simpleData,
			want: xy.Id(),
		},
		{
			name: "ResourceId Source",
			tmpl: `{{ .Source }}`,
			data: simpleData,
			want: xy.Id(),
		},
		{
			name: "ResourceId Destination",
			tmpl: `{{ .Destination }}`,
			data: simpleData,
			want: c.Id(),
		},

		// DAG access
		{
			name: "upstream",
			tmpl: `{{ upstream "test:resource" .Self }}`,
			data: simpleData,
			want: b.Id(),
		},
		{
			name: "all upstream",
			tmpl: `{{ allUpstream "test:resource" .Self | toJson }}`,
			data: simpleData,
			want: []construct.ResourceId{b.Id(), a.Id()},
		},
		{
			name: "downstream",
			tmpl: `{{ downstream "test:resource" .Self }}`,
			data: simpleData,
			want: c.Id(),
		},
		{
			name: "all downstream",
			tmpl: `{{ allDownstream "test:resource" .Self | toJson }}`,
			data: simpleData,
			want: []construct.ResourceId{c.Id(), d.Id()},
		},

		// Resource access
		{
			name: "fieldValue",
			tmpl: `{{ fieldValue "Field" .Self }}`,
			data: simpleData,
			want: "my value",
		},
		{
			name: "fieldValue",
			tmpl: `{{ fieldRef "Field" .Self }}`,
			data: simpleData,
			want: construct.IaCValue{ResourceId: xy.Id(), Property: "Field"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)

			gotV := reflect.New(reflect.TypeOf(tt.want))
			err := ctx.ExecuteDecode(tt.tmpl, tt.data, gotV.Interface())
			if tt.wantErr {
				assert.Error(err)
				return
			}
			if !assert.NoError(err) {
				return
			}
			assert.Equal(tt.want, gotV.Elem().Interface())
		})
	}
}
