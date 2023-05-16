package core

import (
	"reflect"
	"testing"

	klothograph "github.com/klothoplatform/klotho/pkg/graph"

	"github.com/dominikbraun/graph"
	"github.com/stretchr/testify/assert"
)

var TestKnowledgeBase = EdgeKB{
	Edge{From: reflect.TypeOf(&A{}), To: reflect.TypeOf(&B{})}: EdgeDetails{
		ExpansionFunc: func(from, to Resource, dag *ResourceGraph, data EdgeData) error {
			dag.AddDependency(from, to)
			return nil
		},
		Configure: func(from, to Resource, data EdgeData) error {
			a := from.(*A)
			a.Name = "name"
			return nil
		},
	},
	Edge{From: reflect.TypeOf(&A{}), To: reflect.TypeOf(&E{})}: EdgeDetails{
		ExpansionFunc: func(from, to Resource, dag *ResourceGraph, data EdgeData) error {
			dag.AddDependency(from, to)
			return nil
		},
	},
	Edge{From: reflect.TypeOf(&B{}), To: reflect.TypeOf(&C{})}: EdgeDetails{
		ExpansionFunc: func(from, to Resource, dag *ResourceGraph, data EdgeData) error {
			dag.AddDependency(from, to)
			return nil
		},
	},
	Edge{From: reflect.TypeOf(&C{}), To: reflect.TypeOf(&D{})}: EdgeDetails{
		ExpansionFunc: func(from, to Resource, dag *ResourceGraph, data EdgeData) error {
			dag.AddDependency(from, to)
			return nil
		},
	},
	Edge{From: reflect.TypeOf(&C{}), To: reflect.TypeOf(&E{})}: EdgeDetails{
		ExpansionFunc: func(from, to Resource, dag *ResourceGraph, data EdgeData) error {
			dag.AddDependency(from, to)
			return nil
		},
	},
	Edge{From: reflect.TypeOf(&D{}), To: reflect.TypeOf(&B{})}: EdgeDetails{
		ExpansionFunc: func(from, to Resource, dag *ResourceGraph, data EdgeData) error {
			dag.AddDependency(from, to)
			return nil
		},
	},
	Edge{From: reflect.TypeOf(&D{}), To: reflect.TypeOf(&E{})}: EdgeDetails{
		ExpansionFunc: func(from, to Resource, dag *ResourceGraph, data EdgeData) error {
			dag.AddDependency(from, to)
			return nil
		},
	},
}

func Test_FindPaths(t *testing.T) {
	cases := []struct {
		name   string
		source Resource
		dest   Resource
		want   [][]Resource
	}{
		{
			name:   "paths from a",
			source: &A{},
			dest:   &E{},
			want: [][]Resource{
				{&A{}, &B{}, &C{}, &D{}, &E{}},
				{&A{}, &B{}, &C{}, &E{}},
				{&A{}, &E{}},
			},
		},
		{
			name:   "paths from b",
			source: &B{},
			dest:   &E{},
			want: [][]Resource{
				{&B{}, &C{}, &D{}, &E{}},
				{&B{}, &C{}, &E{}},
			},
		},
		{
			name:   "no paths from e",
			source: &E{},
			dest:   &A{},
			want:   [][]Resource{},
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)

			result := FindPaths(TestKnowledgeBase, reflect.TypeOf(tt.source), reflect.TypeOf(tt.dest))

			expectedResult := [][]reflect.Type{}
			for index, arr := range tt.want {
				expectedResult = append(expectedResult, []reflect.Type{})
				for _, res := range arr {
					expectedResult[index] = append(expectedResult[index], reflect.TypeOf(res))
				}
			}

			assert.ElementsMatch(expectedResult, result)
		})
	}
}

func Test_ConfigureFromEdgeData(t *testing.T) {
	cases := []struct {
		name   string
		source Resource
		dest   Resource
		data   EdgeData
		want   []klothograph.Edge[Resource]
	}{
		{
			name:   "node must and must not exist",
			source: &A{},
			dest:   &B{},
			want: []klothograph.Edge[Resource]{
				{Source: &A{Name: "name"}, Destination: &B{}, Properties: graph.EdgeProperties{Attributes: map[string]string{}, Data: EdgeData{}}},
			},
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			dag := NewResourceGraph()
			dag.AddDependencyWithData(tt.source, tt.dest, tt.data)
			err := ConfigureFromEdgeData(TestKnowledgeBase, dag)
			assert.NoError(err)
			assert.ElementsMatch(tt.want, dag.ListDependencies())
		})
	}
}

func Test_ExpandEdges(t *testing.T) {
	cases := []struct {
		name   string
		source Resource
		dest   Resource
		data   EdgeData
		want   []klothograph.Edge[Resource]
	}{
		{
			name:   "node must and must not exist",
			source: &A{},
			dest:   &E{},
			data: EdgeData{
				Constraint: EdgeConstraint{
					NodeMustExist:    &C{},
					NodeMustNotExist: &D{},
				},
			},
			want: []klothograph.Edge[Resource]{
				{Source: &A{}, Destination: &B{}, Properties: graph.EdgeProperties{Attributes: map[string]string{}}},
				{Source: &B{}, Destination: &C{}, Properties: graph.EdgeProperties{Attributes: map[string]string{}}},
				{Source: &C{}, Destination: &E{}, Properties: graph.EdgeProperties{Attributes: map[string]string{}}},
			},
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			dag := NewResourceGraph()
			dag.AddDependencyWithData(tt.source, tt.dest, tt.data)
			err := ExpandEdges(TestKnowledgeBase, dag)
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

func (f *A) Id() ResourceId                      { return ResourceId{Name: "A" + f.Name} }
func (f *A) Provider() string                    { return "DummyProvider" }
func (f *A) KlothoConstructRef() []AnnotationKey { return nil }

func (b B) Id() ResourceId                      { return ResourceId{Name: "B" + b.Name} }
func (f B) Provider() string                    { return "DummyProvider" }
func (f B) KlothoConstructRef() []AnnotationKey { return nil }

func (p *C) Id() ResourceId                      { return ResourceId{Name: "C" + p.Name} }
func (f *C) Provider() string                    { return "DummyProvider" }
func (f *C) KlothoConstructRef() []AnnotationKey { return nil }

func (p *D) Id() ResourceId                      { return ResourceId{Name: "D" + p.Name} }
func (f *D) Provider() string                    { return "DummyProvider" }
func (f *D) KlothoConstructRef() []AnnotationKey { return nil }

func (p *E) Id() ResourceId                      { return ResourceId{Name: "E" + p.Name} }
func (f *E) Provider() string                    { return "DummyProvider" }
func (f *E) KlothoConstructRef() []AnnotationKey { return nil }
