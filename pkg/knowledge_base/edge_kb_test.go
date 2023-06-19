package knowledgebase

import (
	"reflect"
	"testing"

	"github.com/klothoplatform/klotho/pkg/core"

	klothograph "github.com/klothoplatform/klotho/pkg/graph"

	"github.com/dominikbraun/graph"
	"github.com/stretchr/testify/assert"
)

var TestKnowledgeBase = Build(
	EdgeBuilder[*A, *B]{
		Expand: func(a *A, b *B, dag *core.ResourceGraph, data EdgeData) error {
			b.Name = "B"
			dag.AddDependency(a, b)
			return nil
		},
		Configure: func(a *A, b *B, dag *core.ResourceGraph, data EdgeData) error {
			a.Name = "name"
			return nil
		},
		ValidDestinations: []core.Resource{&E{}},
	},
	EdgeBuilder[*A, *E]{
		Expand: func(a *A, e *E, dag *core.ResourceGraph, data EdgeData) error {
			dag.AddDependency(a, e)
			return nil
		},
	},
	EdgeBuilder[*B, *C]{
		Expand: func(b *B, c *C, dag *core.ResourceGraph, data EdgeData) error {
			dag.AddDependency(b, c)
			return nil
		},
		ValidDestinations: []core.Resource{&E{}},
	},
	EdgeBuilder[*C, *D]{
		Expand: func(c *C, d *D, dag *core.ResourceGraph, data EdgeData) error {
			dag.AddDependency(c, d)
			return nil
		},
		ValidDestinations: []core.Resource{&E{}},
	},
	EdgeBuilder[*C, *E]{
		Expand: func(c *C, e *E, dag *core.ResourceGraph, data EdgeData) error {
			dag.AddDependency(c, e)
			return nil
		},
	},
	EdgeBuilder[*D, *B]{
		Expand: func(d *D, b *B, dag *core.ResourceGraph, data EdgeData) error {
			dag.AddDependency(d, b)
			return nil
		},
		ValidDestinations: []core.Resource{&C{}},
	},
	EdgeBuilder[*D, *E]{
		Expand: func(d *D, e *E, dag *core.ResourceGraph, data EdgeData) error {
			dag.AddDependency(d, e)
			return nil
		},
	},
)

var typeA = reflect.TypeOf(&A{})
var typeB = reflect.TypeOf(&B{})
var typeC = reflect.TypeOf(&C{})
var typeD = reflect.TypeOf(&D{})
var typeE = reflect.TypeOf(&E{})

func Test_FindPaths(t *testing.T) {
	cases := []struct {
		name   string
		source core.Resource
		dest   core.Resource
		want   []Path
	}{
		{
			name:   "paths from a",
			source: &A{},
			dest:   &E{},
			want: []Path{
				{{typeA, typeB}, {typeB, typeC}, {typeC, typeD}, {typeD, typeE}},
				{{typeA, typeB}, {typeB, typeC}, {typeC, typeE}},
				{{typeA, typeE}},
			},
		},
		{
			name:   "paths from b",
			source: &B{},
			dest:   &E{},
			want: []Path{
				{{typeB, typeC}, {typeC, typeD}, {typeD, typeE}},
				{{typeB, typeC}, {typeC, typeE}},
			},
		},
		{
			name:   "paths from d to c",
			source: &D{},
			dest:   &C{},
			want: []Path{
				{{typeD, typeB}, {typeB, typeC}},
			},
		},
		{
			name:   "no paths from e",
			source: &E{},
			dest:   &A{},
			want:   []Path{},
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			result := TestKnowledgeBase.FindPaths(tt.source, tt.dest, EdgeConstraint{})
			assert.ElementsMatch(tt.want, result)
		})
	}
}

func Test_ConfigureFromEdgeData(t *testing.T) {
	cases := []struct {
		name   string
		source core.Resource
		dest   core.Resource
		data   EdgeData
		want   []klothograph.Edge[core.Resource]
	}{
		{
			name:   "node must and must not exist",
			source: &A{},
			dest:   &B{},
			want: []klothograph.Edge[core.Resource]{
				{Source: &A{Name: "name"}, Destination: &B{}, Properties: graph.EdgeProperties{Attributes: map[string]string{}, Data: EdgeData{}}},
			},
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			dag := core.NewResourceGraph()
			dag.AddDependencyWithData(tt.source, tt.dest, tt.data)
			err := TestKnowledgeBase.ConfigureFromEdgeData(dag)
			assert.NoError(err)
			assert.ElementsMatch(tt.want, dag.ListDependencies())
		})
	}
}

func Test_ExpandEdges(t *testing.T) {
	cases := []struct {
		name   string
		source core.Resource
		dest   core.Resource
		data   EdgeData
		want   []klothograph.Edge[core.Resource]
	}{
		{
			name:   "node must and must not exist",
			source: &A{Name: "A"},
			dest:   &E{Name: "E"},
			data: EdgeData{
				Constraint: EdgeConstraint{
					NodeMustExist:    []core.Resource{&C{}},
					NodeMustNotExist: []core.Resource{&D{}},
				},
			},
			want: []klothograph.Edge[core.Resource]{
				{Source: &A{Name: "A"}, Destination: &B{Name: "B"}, Properties: graph.EdgeProperties{Attributes: map[string]string{}}},
				{Source: &B{Name: "B"}, Destination: &C{}, Properties: graph.EdgeProperties{Attributes: map[string]string{}}},
				{Source: &C{}, Destination: &E{Name: "E"}, Properties: graph.EdgeProperties{Attributes: map[string]string{}}},
			},
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			dag := core.NewResourceGraph()
			dag.AddDependencyWithData(tt.source, tt.dest, tt.data)
			err := TestKnowledgeBase.ExpandEdges(dag, "my-app")
			assert.NoError(err)
			assert.ElementsMatch(tt.want, dag.ListDependencies())
		})
	}
}

type (
	A struct {
		Name string
	}
	B struct {
		Name string
	}
	C struct {
		Name string
	}
	D struct {
		Name string
	}
	E struct {
		Name string
	}
)

func (f *A) Id() core.ResourceId                      { return core.ResourceId{Type: "A", Name: "A" + f.Name} }
func (f *A) BaseConstructsRef() core.BaseConstructSet { return nil }
func (f *A) DeleteContext() core.DeleteContext {
	return core.DeleteContext{
		RequiresNoUpstream: true,
	}
}

func (b B) Id() core.ResourceId                      { return core.ResourceId{Type: "B", Name: "B" + b.Name} }
func (f B) BaseConstructsRef() core.BaseConstructSet { return nil }
func (f B) DeleteContext() core.DeleteContext {
	return core.DeleteContext{
		RequiresNoUpstream: true,
	}
}
func (p *C) Id() core.ResourceId                      { return core.ResourceId{Type: "C", Name: "C" + p.Name} }
func (f *C) BaseConstructsRef() core.BaseConstructSet { return nil }
func (f *C) DeleteContext() core.DeleteContext {
	return core.DeleteContext{
		RequiresNoUpstream: true,
	}
}
func (p *D) Id() core.ResourceId                      { return core.ResourceId{Type: "D", Name: "D" + p.Name} }
func (f *D) BaseConstructsRef() core.BaseConstructSet { return nil }
func (f *D) DeleteContext() core.DeleteContext {
	return core.DeleteContext{
		RequiresNoUpstream: true,
	}
}
func (p *E) Id() core.ResourceId                      { return core.ResourceId{Type: "E", Name: "E" + p.Name} }
func (f *E) BaseConstructsRef() core.BaseConstructSet { return nil }
func (f *E) DeleteContext() core.DeleteContext {
	return core.DeleteContext{
		RequiresNoUpstream: true,
	}
}
