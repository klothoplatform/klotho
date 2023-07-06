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
		Configure: func(a *A, b *B, dag *core.ResourceGraph, data EdgeData) error {
			b.Name = "B"
			a.Name = "name"
			return nil
		},
	},
	EdgeBuilder[*A, *E]{},
	EdgeBuilder[*B, *C]{},
	EdgeBuilder[*C, *D]{},
	EdgeBuilder[*C, *E]{},
	EdgeBuilder[*D, *B]{},
	EdgeBuilder[*D, *E]{},
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

func Test_ConfigureEdge(t *testing.T) {
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
				{Source: &A{Name: "name"}, Destination: &B{Name: "B"}, Properties: graph.EdgeProperties{Attributes: map[string]string{}, Data: EdgeData{}}},
			},
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			dag := core.NewResourceGraph()
			dag.AddDependencyWithData(tt.source, tt.dest, tt.data)
			edge := dag.GetDependency(tt.source.Id(), tt.dest.Id())
			err := TestKnowledgeBase.ConfigureEdge(edge, dag)
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
		path   Path
		want   []klothograph.Edge[core.Resource]
	}{
		{
			name:   "node must and must not exist",
			source: &A{},
			dest:   &E{},
			data: EdgeData{
				Constraint: EdgeConstraint{
					NodeMustExist:    []core.Resource{&C{}},
					NodeMustNotExist: []core.Resource{&D{}},
				},
			},
			path: Path{
				{typeA, typeB}, {typeB, typeC}, {typeC, typeE},
			},
			want: []klothograph.Edge[core.Resource]{
				{Source: &A{}, Destination: &B{Name: "B_A_E"}},
				{Source: &B{Name: "B_A_E"}, Destination: &C{}},
				{Source: &C{}, Destination: &E{}},
			},
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			dag := core.NewResourceGraph()
			dag.AddDependencyWithData(tt.source, tt.dest, tt.data)
			edge := dag.GetDependency(tt.source.Id(), tt.dest.Id())
			err := TestKnowledgeBase.ExpandEdge(edge, dag, tt.path, tt.data)

			var result []klothograph.Edge[core.Resource]
			for _, dep := range dag.ListDependencies() {
				result = append(result, klothograph.Edge[core.Resource]{Source: dep.Source, Destination: dep.Destination})
			}
			assert.NoError(err)
			assert.ElementsMatch(tt.want, result)

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
