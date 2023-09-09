package knowledgebase

import (
	"reflect"
	"testing"

	"github.com/dominikbraun/graph"
	"github.com/klothoplatform/klotho/pkg/construct"
	klothograph "github.com/klothoplatform/klotho/pkg/graph"
	"github.com/stretchr/testify/assert"
)

var TestKnowledgeBase = Build(
	EdgeBuilder[*A, *B]{
		Configure: func(a *A, b *B, dag *construct.ResourceGraph, data EdgeData) error {
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
var typeE = reflect.TypeOf(&E{})

func Test_ConfigureEdge(t *testing.T) {
	cases := []struct {
		name   string
		source construct.Resource
		dest   construct.Resource
		data   EdgeData
		want   []klothograph.Edge[construct.Resource]
	}{
		{
			name:   "node must and must not exist",
			source: &A{},
			dest:   &B{},
			want: []klothograph.Edge[construct.Resource]{
				{Source: &A{Name: "name"}, Destination: &B{Name: "B"}, Properties: graph.EdgeProperties{Attributes: map[string]string{}, Data: EdgeData{}}},
			},
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			dag := construct.NewResourceGraph()
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
		source construct.Resource
		dest   construct.Resource
		data   EdgeData
		path   Path
		want   []klothograph.Edge[construct.Resource]
	}{
		{
			name:   "node must and must not exist",
			source: &A{},
			dest:   &E{},
			data: EdgeData{
				Constraint: EdgeConstraint{
					NodeMustExist:    []construct.Resource{&C{}},
					NodeMustNotExist: []construct.Resource{&D{}},
				},
			},
			path: Path{
				{typeA, typeB}, {typeB, typeC}, {typeC, typeE},
			},
			want: []klothograph.Edge[construct.Resource]{
				{Source: &A{}, Destination: &B{Name: "B_A_E"}},
				{Source: &B{Name: "B_A_E"}, Destination: &C{}},
				{Source: &C{}, Destination: &E{}},
			},
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			dag := construct.NewResourceGraph()
			dag.AddDependencyWithData(tt.source, tt.dest, tt.data)
			edge := dag.GetDependency(tt.source.Id(), tt.dest.Id())
			edges := TestKnowledgeBase.ExpandEdge(edge, dag, tt.path, tt.data)
			assert.ElementsMatch(tt.want, edges)

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

func (f *A) Id() construct.ResourceId                      { return construct.ResourceId{Type: "A", Name: "A" + f.Name} }
func (f *A) BaseConstructRefs() construct.BaseConstructSet { return nil }
func (f *A) DeleteContext() construct.DeleteContext {
	return construct.DeleteContext{
		RequiresNoUpstream: true,
	}
}

func (b B) Id() construct.ResourceId                      { return construct.ResourceId{Type: "B", Name: "B" + b.Name} }
func (f B) BaseConstructRefs() construct.BaseConstructSet { return nil }
func (f B) DeleteContext() construct.DeleteContext {
	return construct.DeleteContext{
		RequiresNoUpstream: true,
	}
}
func (p *C) Id() construct.ResourceId                      { return construct.ResourceId{Type: "C", Name: "C" + p.Name} }
func (f *C) BaseConstructRefs() construct.BaseConstructSet { return nil }
func (f *C) DeleteContext() construct.DeleteContext {
	return construct.DeleteContext{
		RequiresNoUpstream: true,
	}
}
func (p *D) Id() construct.ResourceId                      { return construct.ResourceId{Type: "D", Name: "D" + p.Name} }
func (f *D) BaseConstructRefs() construct.BaseConstructSet { return nil }
func (f *D) DeleteContext() construct.DeleteContext {
	return construct.DeleteContext{
		RequiresNoUpstream: true,
	}
}
func (p *E) Id() construct.ResourceId                      { return construct.ResourceId{Type: "E", Name: "E" + p.Name} }
func (f *E) BaseConstructRefs() construct.BaseConstructSet { return nil }
func (f *E) DeleteContext() construct.DeleteContext {
	return construct.DeleteContext{
		RequiresNoUpstream: true,
	}
}

func TestEdge_SourceTypeName(t *testing.T) {
	edge := NewEdge[*A, *B]()
	a := &A{}
	assert.Equal(t, a.Id(), edge.SourceId())
	b := &B{}
	assert.Equal(t, b.Id(), edge.DestinationId())
}
