package knowledgebase

import (
	"testing"

	"github.com/klothoplatform/klotho/pkg/construct"
	"github.com/stretchr/testify/assert"
)

type (
	TestResource[T any] struct {
		Provider string
		Type     string
		Name     string
		Field    T
	}
	SimpleResource = TestResource[struct{}]
)

func (t *TestResource[T]) Id() construct.ResourceId {
	id := construct.ResourceId{
		Provider: t.Provider,
		Type:     t.Type,
		Name:     t.Name,
	}
	if id.Provider == "" {
		id.Provider = "test"
	}
	if id.Type == "" {
		id.Type = "resource"
	}
	return id
}

func (t *TestResource[T]) BaseConstructRefs() construct.BaseConstructSet {
	return nil
}
func (t *TestResource[T]) DeleteContext() construct.DeleteContext {
	return construct.DeleteContext{}
}

func TestConfigurationStep_Apply(t *testing.T) {
	type complexField struct {
		A string
		B int
		M map[string]string
	}

	// Creates a test DAG of:
	// up[int] -> r[complexField] -> down[string]
	// where r is the resource being configured
	makeCtx := func() (*ConfigurationContext, *TestResource[complexField], *TestResource[int], *TestResource[string]) {
		r := &TestResource[complexField]{Name: "test"}
		up := &TestResource[int]{Type: "a", Name: "int"}
		down := &TestResource[string]{Type: "b", Name: "string"}
		ctx := &ConfigurationContext{
			dag:      construct.NewResourceGraph(),
			resource: r,
		}
		ctx.dag.AddDependency(up, r)
		ctx.dag.AddDependency(r, down)
		return ctx, r, up, down
	}

	tests := []struct {
		name       string
		object     string
		value      string
		inputValue any
		want       complexField
		wantUp     int
		wantDown   string
		wantErr    bool
	}{
		{
			name:       "set upstream",
			object:     `{{ upstream "test:a" }}`,
			inputValue: 1,
			wantUp:     1,
		},
		{
			name:       "set downstream",
			object:     `{{ downstream "test:b" }}`,
			inputValue: "hello",
			wantDown:   "hello",
		},
		{
			name: "set complex",
			value: `{{ zipToMap
				"[\"A\", \"B\", \"M\"]"
				"[\"a\", 2, {\"hello\": \"world\"}]"}}`,
			want: complexField{
				A: "a",
				B: 2,
				M: map[string]string{"hello": "world"},
			},
		},
		{
			name:    "bad value type",
			value:   `"hello"`,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)

			ctx, r, up, down := makeCtx()
			ctx.Value = tt.inputValue

			step := &ConfigurationStep{
				Object:   tt.object,
				Property: "Field",
				Value:    tt.value,
			}
			err := step.Apply(ctx)
			if tt.wantErr {
				assert.Error(err)
				return
			}
			if !assert.NoError(err) {
				return
			}
			assert.Equal(tt.want, r.Field, "resource.Field")
			assert.Equal(tt.wantUp, up.Field, "up.Field")
			assert.Equal(tt.wantDown, down.Field, "down.Field")
		})
	}
}
