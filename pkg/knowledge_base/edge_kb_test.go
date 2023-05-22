package knowledgebase

import (
	"reflect"
	"testing"

	"github.com/klothoplatform/klotho/pkg/core"

	klothograph "github.com/klothoplatform/klotho/pkg/graph"

	"github.com/dominikbraun/graph"
	"github.com/stretchr/testify/assert"
)

var TestKnowledgeBase = &EdgeKB{
	Edge{Source: reflect.TypeOf(&A{}), Destination: reflect.TypeOf(&B{})}: EdgeDetails{
		ExpansionFunc: func(source, dest core.Resource, dag *core.ResourceGraph, data EdgeData) error {
			b := dest.(*B)
			b.Name = "B"
			dag.AddDependency(source, dest)
			return nil
		},
		Configure: func(source, dest core.Resource, dag *core.ResourceGraph, data EdgeData) error {
			a := source.(*A)
			a.Name = "name"
			return nil
		},
	},
	Edge{Source: reflect.TypeOf(&A{}), Destination: reflect.TypeOf(&E{})}: EdgeDetails{
		ExpansionFunc: func(source, dest core.Resource, dag *core.ResourceGraph, data EdgeData) error {
			dag.AddDependency(source, dest)
			return nil
		},
		ValidDestinations: []reflect.Type{typeE},
	},
	Edge{Source: reflect.TypeOf(&B{}), Destination: reflect.TypeOf(&C{})}: EdgeDetails{
		ExpansionFunc: func(source, dest core.Resource, dag *core.ResourceGraph, data EdgeData) error {
			dag.AddDependency(source, dest)
			return nil
		},
	},
	Edge{Source: reflect.TypeOf(&C{}), Destination: reflect.TypeOf(&D{})}: EdgeDetails{
		ExpansionFunc: func(source, dest core.Resource, dag *core.ResourceGraph, data EdgeData) error {
			dag.AddDependency(source, dest)
			return nil
		},
		ValidDestinations: []reflect.Type{typeE, typeD},
	},
	Edge{Source: reflect.TypeOf(&C{}), Destination: reflect.TypeOf(&E{})}: EdgeDetails{
		ExpansionFunc: func(source, dest core.Resource, dag *core.ResourceGraph, data EdgeData) error {
			dag.AddDependency(source, dest)
			return nil
		},
		ValidDestinations: []reflect.Type{typeE},
	},
	Edge{Source: reflect.TypeOf(&D{}), Destination: reflect.TypeOf(&B{})}: EdgeDetails{
		ExpansionFunc: func(source, dest core.Resource, dag *core.ResourceGraph, data EdgeData) error {
			dag.AddDependency(source, dest)
			return nil
		},
		ValidDestinations: []reflect.Type{typeC},
	},
	Edge{Source: reflect.TypeOf(&D{}), Destination: reflect.TypeOf(&E{})}: EdgeDetails{
		ExpansionFunc: func(source, dest core.Resource, dag *core.ResourceGraph, data EdgeData) error {
			dag.AddDependency(source, dest)
			return nil
		},
		ValidDestinations: []reflect.Type{typeE},
	},
}

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
				{{typeA, typeB}, {typeA, typeE}},
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

			result := TestKnowledgeBase.FindPaths(reflect.TypeOf(tt.source), reflect.TypeOf(tt.dest))
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
			err := TestKnowledgeBase.ExpandEdges(dag)
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

func (f *A) Id() core.ResourceId                      { return core.ResourceId{Name: "A" + f.Name} }
func (f *A) KlothoConstructRef() []core.AnnotationKey { return nil }

func (b B) Id() core.ResourceId                      { return core.ResourceId{Name: "B" + b.Name} }
func (f B) KlothoConstructRef() []core.AnnotationKey { return nil }

func (p *C) Id() core.ResourceId                      { return core.ResourceId{Name: "C" + p.Name} }
func (f *C) KlothoConstructRef() []core.AnnotationKey { return nil }

func (p *D) Id() core.ResourceId                      { return core.ResourceId{Name: "D" + p.Name} }
func (f *D) KlothoConstructRef() []core.AnnotationKey { return nil }

func (p *E) Id() core.ResourceId                      { return core.ResourceId{Name: "E" + p.Name} }
func (f *E) KlothoConstructRef() []core.AnnotationKey { return nil }
