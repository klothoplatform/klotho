package operational_rule

import (
	"testing"

	"github.com/dominikbraun/graph"
	construct "github.com/klothoplatform/klotho/pkg/construct2"
	"github.com/klothoplatform/klotho/pkg/construct2/graphtest"
	"github.com/klothoplatform/klotho/pkg/engine2/enginetesting"
	kbtesting "github.com/klothoplatform/klotho/pkg/engine2/enginetesting/test_kb"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func Test_getResourcesForStep(t *testing.T) {
	tests := []struct {
		name     string
		step     knowledgebase.OperationalStep
		resource *construct.Resource
		want     []construct.ResourceId
	}{
		{
			name:     "upstream resource types",
			resource: kbtesting.MockResource1("test1"),
			step: knowledgebase.OperationalStep{
				Direction: knowledgebase.DirectionUpstream,
				Resources: []knowledgebase.ResourceSelector{{Selector: "mock:resource4"}},
			},
			want: []construct.ResourceId{
				{Provider: "mock", Type: "resource4", Name: "test"},
				{Provider: "mock", Type: "resource4", Name: "test2"},
			},
		},
		{
			name:     "downstream resource types",
			resource: kbtesting.MockResource1("test1"),
			step: knowledgebase.OperationalStep{
				Direction: knowledgebase.DirectionDownstream,
				Resources: []knowledgebase.ResourceSelector{{Selector: "mock:resource4"}},
			},
			want: []construct.ResourceId{
				{Provider: "mock", Type: "resource4", Name: "test"},
				{Provider: "mock", Type: "resource4", Name: "test2"},
			},
		},
		{
			name:     "downstream classifications",
			resource: kbtesting.MockResource1("test1"),
			step: knowledgebase.OperationalStep{
				Direction: knowledgebase.DirectionDownstream,
				Resources: []knowledgebase.ResourceSelector{{Classifications: []string{"role"}}},
			},
			want: []construct.ResourceId{
				{Provider: "mock", Type: "resource4", Name: "test"},
				{Provider: "mock", Type: "resource4", Name: "test2"},
			},
		},
		{
			name:     "upstream classifications",
			resource: kbtesting.MockResource1("test1"),
			step: knowledgebase.OperationalStep{
				Direction: knowledgebase.DirectionUpstream,
				Resources: []knowledgebase.ResourceSelector{{Classifications: []string{"role"}}},
			},
			want: []construct.ResourceId{
				{Provider: "mock", Type: "resource4", Name: "test"},
				{Provider: "mock", Type: "resource4", Name: "test2"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			testSol := enginetesting.NewTestSolution()
			testSol.KB.On("GetEdgeTemplate", mock.Anything, mock.Anything).Return(&knowledgebase.EdgeTemplate{}, nil)
			testSol.KB.On("GetResourceTemplate", graphtest.ParseId(t, "mock:resource1")).Return(resource1, nil)
			testSol.KB.On("GetResourceTemplate", graphtest.ParseId(t, "mock:resource2")).Return(resource2, nil)
			testSol.KB.On("GetResourceTemplate", graphtest.ParseId(t, "mock:resource3")).Return(resource3, nil)
			testSol.KB.On("GetResourceTemplate", graphtest.ParseId(t, "mock:resource4")).Return(resource4, nil)
			testSol.KB.On("ListResources").Return([]*knowledgebase.ResourceTemplate{resource1, resource2, resource3, resource4}, nil)
			testResources := []*construct.Resource{
				kbtesting.MockResource4("test"),
				kbtesting.MockResource4("test2"),
				kbtesting.MockResource3("test3"),
			}
			testSol.RawView().AddVertex(tt.resource)
			for _, res := range testResources {
				testSol.RawView().AddVertex(res)
				if tt.step.Direction == knowledgebase.DirectionDownstream {
					testSol.RawView().AddEdge(tt.resource.ID, res.ID)
				} else {
					testSol.RawView().AddEdge(res.ID, tt.resource.ID)
				}
			}
			ctx := OperationalRuleContext{
				Solution: testSol,
			}

			result, err := ctx.getResourcesForStep(tt.step, tt.resource.ID)
			if err != nil {
				t.Fatal(err)
			}
			assert.ElementsMatch(tt.want, result)
		})
	}
}

func Test_addDependenciesFromProperty(t *testing.T) {
	tests := []struct {
		name         string
		step         knowledgebase.OperationalStep
		resource     *construct.Resource
		propertyName string
		initialState []any
		want         enginetesting.ExpectedGraphs
		wantIds      []construct.ResourceId
		wantEdges    []construct.Edge
	}{
		{
			name: "downstream",
			resource: &construct.Resource{
				ID: graphtest.ParseId(t, "a:a:a"),
				Properties: map[string]interface{}{
					"Res4": kbtesting.MockResource4("test").ID,
				}},
			step:         knowledgebase.OperationalStep{Direction: knowledgebase.DirectionDownstream},
			propertyName: "Res4",
			initialState: []any{"mock:resource4:test", "a:a:a"},
			want: enginetesting.ExpectedGraphs{
				Dataflow:   []any{"mock:resource4:test", "a:a:a", "a:a:a -> mock:resource4:test"},
				Deployment: []any{"mock:resource4:test", "a:a:a", "a:a:a -> mock:resource4:test"},
			},
			wantIds: []construct.ResourceId{
				graphtest.ParseId(t, "mock:resource4:test"),
			},
			wantEdges: []construct.Edge{
				{Source: graphtest.ParseId(t, "a:a:a"), Target: graphtest.ParseId(t, "mock:resource4:test")},
			},
		},
		{
			name: "upstream",
			resource: &construct.Resource{
				ID: graphtest.ParseId(t, "a:a:a"),
				Properties: map[string]interface{}{
					"Res4": kbtesting.MockResource4("test").ID,
				}},
			step:         knowledgebase.OperationalStep{Direction: knowledgebase.DirectionUpstream},
			propertyName: "Res4",
			initialState: []any{"mock:resource4:test", "a:a:a"},
			want: enginetesting.ExpectedGraphs{
				Dataflow:   []any{"mock:resource4:test", "a:a:a", "mock:resource4:test -> a:a:a"},
				Deployment: []any{"mock:resource4:test", "a:a:a", "mock:resource4:test -> a:a:a"},
			},
			wantIds: []construct.ResourceId{
				graphtest.ParseId(t, "mock:resource4:test"),
			},
			wantEdges: []construct.Edge{
				{Source: graphtest.ParseId(t, "mock:resource4:test"), Target: graphtest.ParseId(t, "a:a:a")},
			},
		},
		{
			name: "array",
			resource: &construct.Resource{
				ID: graphtest.ParseId(t, "a:a:a"),
				Properties: map[string]interface{}{
					"Res2s": []construct.ResourceId{kbtesting.MockResource2("test").ID, kbtesting.MockResource2("test2").ID},
				}},
			step:         knowledgebase.OperationalStep{Direction: knowledgebase.DirectionDownstream},
			propertyName: "Res2s",
			initialState: []any{"mock:resource2:test", "mock:resource2:test2", "a:a:a"},
			want: enginetesting.ExpectedGraphs{
				Dataflow:   []any{"mock:resource2:test", "mock:resource2:test2", "a:a:a", "a:a:a -> mock:resource2:test", "a:a:a -> mock:resource2:test2"},
				Deployment: []any{"mock:resource2:test", "mock:resource2:test2", "a:a:a", "a:a:a -> mock:resource2:test", "a:a:a -> mock:resource2:test2"},
			},
			wantIds: []construct.ResourceId{
				graphtest.ParseId(t, "mock:resource2:test"),
				graphtest.ParseId(t, "mock:resource2:test2"),
			},
			wantEdges: []construct.Edge{
				{Source: graphtest.ParseId(t, "a:a:a"), Target: graphtest.ParseId(t, "mock:resource2:test")},
				{Source: graphtest.ParseId(t, "a:a:a"), Target: graphtest.ParseId(t, "mock:resource2:test2")},
			},
		},
		{
			name: "array with existing",
			resource: &construct.Resource{
				ID: graphtest.ParseId(t, "a:a:a"),
				Properties: map[string]interface{}{
					"Res2s": []construct.ResourceId{kbtesting.MockResource2("test").ID, kbtesting.MockResource2("test2").ID},
				}},
			step:         knowledgebase.OperationalStep{Direction: knowledgebase.DirectionDownstream},
			propertyName: "Res2s",
			initialState: []any{"mock:resource2:test", "mock:resource2:test2", "a:a:a", "a:a:a -> mock:resource2:test"},
			want: enginetesting.ExpectedGraphs{
				Dataflow:   []any{"mock:resource2:test", "mock:resource2:test2", "a:a:a", "a:a:a -> mock:resource2:test", "a:a:a -> mock:resource2:test2"},
				Deployment: []any{"mock:resource2:test", "mock:resource2:test2", "a:a:a", "a:a:a -> mock:resource2:test", "a:a:a -> mock:resource2:test2"},
			},
			wantIds: []construct.ResourceId{
				graphtest.ParseId(t, "mock:resource2:test"),
				graphtest.ParseId(t, "mock:resource2:test2"),
			},
			wantEdges: []construct.Edge{
				{Source: graphtest.ParseId(t, "a:a:a"), Target: graphtest.ParseId(t, "mock:resource2:test2")},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			testSol := enginetesting.NewTestSolution()
			testSol.KB.On("GetEdgeTemplate", mock.Anything, mock.Anything).Return(&knowledgebase.EdgeTemplate{}, nil)
			testSol.LoadState(t, tt.initialState...)
			ctx := OperationalRuleContext{
				Solution: testSol,
			}

			ids, edges, err := ctx.addDependenciesFromProperty(tt.step, tt.resource, tt.propertyName)
			if !assert.NoError(err) {
				return
			}
			tt.want.AssertEqual(t, testSol)
			assert.ElementsMatch(tt.wantIds, ids)
			assert.ElementsMatch(tt.wantEdges, edges)
		})
	}
}

func Test_clearProperty(t *testing.T) {
	tests := []struct {
		name         string
		step         knowledgebase.OperationalStep
		resource     *construct.Resource
		propertyName string
		initialState []any
		want         enginetesting.ExpectedGraphs
	}{
		{
			name: "downstream",
			resource: &construct.Resource{
				ID: graphtest.ParseId(t, "a:a:a"),
				Properties: map[string]interface{}{
					"Res4": kbtesting.MockResource4("test").ID,
				},
			},
			step:         knowledgebase.OperationalStep{Direction: knowledgebase.DirectionDownstream},
			propertyName: "Res4",
			initialState: []any{"mock:resource4:test", "a:a:a", "a:a:a -> mock:resource4:test"},
			want: enginetesting.ExpectedGraphs{
				Dataflow:   []any{"mock:resource4:test", "a:a:a"},
				Deployment: []any{"mock:resource4:test", "a:a:a"},
			},
		},
		{
			name: "upstream",
			resource: &construct.Resource{
				ID: graphtest.ParseId(t, "a:a:a"),
				Properties: map[string]interface{}{
					"Res4": kbtesting.MockResource4("test").ID,
				},
			},
			step:         knowledgebase.OperationalStep{Direction: knowledgebase.DirectionUpstream},
			propertyName: "Res4",
			initialState: []any{"mock:resource4:test", "a:a:a", "mock:resource4:test -> a:a:a"},
			want: enginetesting.ExpectedGraphs{
				Dataflow:   []any{"mock:resource4:test", "a:a:a"},
				Deployment: []any{"mock:resource4:test", "a:a:a"},
			},
		},
		{
			name: "array",
			resource: &construct.Resource{
				ID: graphtest.ParseId(t, "a:a:a"),
				Properties: map[string]interface{}{
					"Res2s": []construct.ResourceId{kbtesting.MockResource2("test").ID, kbtesting.MockResource2("test2").ID},
				},
			},
			step:         knowledgebase.OperationalStep{Direction: knowledgebase.DirectionUpstream},
			propertyName: "Res2s",
			initialState: []any{"mock:resource2:test", "a:a:a", "mock:resource2:test -> a:a:a", "mock:resource2:test2", "mock:resource2:test2 -> a:a:a"},
			want: enginetesting.ExpectedGraphs{
				Dataflow:   []any{"mock:resource2:test", "mock:resource2:test2", "a:a:a"},
				Deployment: []any{"mock:resource2:test", "mock:resource2:test2", "a:a:a"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			testSol := enginetesting.NewTestSolution()
			testSol.KB.On("GetEdgeTemplate", mock.Anything, mock.Anything).Return(&knowledgebase.EdgeTemplate{}, nil)
			testSol.LoadState(t, tt.initialState...)
			ctx := OperationalRuleContext{
				Solution: testSol,
			}

			err := ctx.clearProperty(tt.step, tt.resource, tt.propertyName)
			if !assert.NoError(err) {
				return
			}

			val, err := tt.resource.GetProperty(tt.propertyName)
			if err != nil {
				t.Fatal(err)
			}
			assert.Nil(val, "property should be nil, but was %v", val)
			tt.want.AssertEqual(t, testSol)
		})
	}
}

func Test_addDependencyForDirection(t *testing.T) {
	tests := []struct {
		name     string
		to       *construct.Resource
		from     *construct.Resource
		step     knowledgebase.OperationalStep
		want     enginetesting.ExpectedGraphs
		wantEdge graph.Edge[construct.ResourceId]
	}{
		{
			name: "upstream",
			to:   kbtesting.MockResource1("test1"),
			from: kbtesting.MockResource4(""),
			step: knowledgebase.OperationalStep{
				Direction: knowledgebase.DirectionUpstream,
			},
			want: enginetesting.ExpectedGraphs{
				Dataflow:   []any{"mock:resource1:test1", "mock:resource4:", "mock:resource4: -> mock:resource1:test1"},
				Deployment: []any{"mock:resource1:test1", "mock:resource4:", "mock:resource4: -> mock:resource1:test1"},
			},
			wantEdge: graph.Edge[construct.ResourceId]{
				Source: graphtest.ParseId(t, "mock:resource4:"), Target: graphtest.ParseId(t, "mock:resource1:test1"),
			},
		},
		{
			name: "downstream",
			to:   kbtesting.MockResource1("test1"),
			from: kbtesting.MockResource4(""),
			step: knowledgebase.OperationalStep{
				Direction: knowledgebase.DirectionDownstream,
			},
			want: enginetesting.ExpectedGraphs{
				Dataflow:   []any{"mock:resource1:test1", "mock:resource4:", "mock:resource1:test1 -> mock:resource4:"},
				Deployment: []any{"mock:resource1:test1", "mock:resource4:", "mock:resource1:test1 -> mock:resource4:"},
			},
			wantEdge: graph.Edge[construct.ResourceId]{
				Source: graphtest.ParseId(t, "mock:resource1:test1"), Target: graphtest.ParseId(t, "mock:resource4:"),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			testSol := enginetesting.NewTestSolution()
			testSol.KB.On("GetEdgeTemplate", mock.Anything, mock.Anything).Return(&knowledgebase.EdgeTemplate{}, nil)
			testSol.LoadState(t, "mock:resource1:test1", "mock:resource4:")
			ctx := OperationalRuleContext{
				Solution: testSol,
			}

			edge, err := ctx.addDependencyForDirection(tt.step, tt.to, tt.from)
			if !assert.NoError(err) {
				return
			}
			tt.want.AssertEqual(t, testSol)
			assert.Equal(tt.wantEdge, edge)
		})
	}
}

func Test_removeDependencyForDirection(t *testing.T) {
	tests := []struct {
		name         string
		to           *construct.Resource
		from         *construct.Resource
		direction    knowledgebase.Direction
		initialState []any
	}{
		{
			name:         "upstream",
			to:           kbtesting.MockResource1("test1"),
			from:         kbtesting.MockResource4(""),
			direction:    knowledgebase.DirectionUpstream,
			initialState: []any{"mock:resource1:test1", "mock:resource4:", "mock:resource4: -> mock:resource1:test1"},
		},
		{
			name:         "downstream",
			to:           kbtesting.MockResource1("test1"),
			from:         kbtesting.MockResource4(""),
			direction:    knowledgebase.DirectionDownstream,
			initialState: []any{"mock:resource1:test1", "mock:resource4:", "mock:resource1:test1 -> mock:resource4:"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			testSol := enginetesting.NewTestSolution()
			testSol.KB.On("GetEdgeTemplate", mock.Anything, mock.Anything).Return(&knowledgebase.EdgeTemplate{}, nil)
			testSol.LoadState(t, tt.initialState...)
			ctx := OperationalRuleContext{
				Solution: testSol,
			}

			err := ctx.removeDependencyForDirection(tt.direction, tt.to.ID, tt.from.ID)
			if !assert.NoError(err) {
				return
			}
			expectedGraph := enginetesting.ExpectedGraphs{
				Dataflow:   []any{"mock:resource1:test1", "mock:resource4:"},
				Deployment: []any{"mock:resource1:test1", "mock:resource4:"},
			}
			expectedGraph.AssertEqual(t, testSol)
		})
	}
}

func Test_setField(t *testing.T) {
	res4ToReplace := kbtesting.MockResource4("thisWillBeReplaced")
	res4 := kbtesting.MockResource4("test4")
	tests := []struct {
		name          string
		resource      *construct.Resource
		resourceToSet *construct.Resource
		property      *knowledgebase.Property
		initialState  []any
		step          knowledgebase.OperationalStep
		wantResource  *construct.Resource
		wantGraph     enginetesting.ExpectedGraphs
	}{
		{
			name: "field is being replaced",
			resource: &construct.Resource{
				ID: construct.ResourceId{Provider: "mock", Type: "resource1", Name: "test1"},
				Properties: map[string]interface{}{
					"Res4": res4ToReplace.ID,
				},
			},
			resourceToSet: res4,
			property:      &knowledgebase.Property{Name: "Res4", Namespace: true, Path: "Res4"},
			step:          knowledgebase.OperationalStep{Direction: knowledgebase.DirectionDownstream},
			initialState:  []any{"mock:resource4:test4", "mock:resource1:test1", "mock:resource1:test1 -> mock:resource4:thisWillBeReplaced"},
			wantResource: &construct.Resource{
				ID: construct.ResourceId{Provider: "mock", Type: "resource1", Namespace: res4.ID.Name, Name: "test1"},
				Properties: map[string]interface{}{
					"Res4": res4.ID,
				},
			},
			wantGraph: enginetesting.ExpectedGraphs{
				Dataflow:   []any{"mock:resource1:test1", "mock:resource4:test4"},
				Deployment: []any{"mock:resource1:test1", "mock:resource4:test4"},
			},
		},
		{
			name: "field is being replaced, remove upstream",
			resource: &construct.Resource{
				ID: construct.ResourceId{Provider: "mock", Type: "resource1", Name: "test1"},
				Properties: map[string]interface{}{
					"Res4": res4ToReplace.ID,
				},
			},
			resourceToSet: res4,
			property:      &knowledgebase.Property{Name: "Res4", Namespace: true, Path: "Res4"},
			step:          knowledgebase.OperationalStep{Direction: knowledgebase.DirectionUpstream},
			initialState:  []any{"mock:resource4:test4", "mock:resource1:test1", "mock:resource4:thisWillBeReplaced -> mock:resource1:test1"},
			wantResource: &construct.Resource{
				ID: construct.ResourceId{Provider: "mock", Type: "resource1", Namespace: res4.ID.Name, Name: "test1"},
				Properties: map[string]interface{}{
					"Res4": res4.ID,
				},
			},
			wantGraph: enginetesting.ExpectedGraphs{
				Dataflow:   []any{"mock:resource1:test1", "mock:resource4:test4"},
				Deployment: []any{"mock:resource1:test1", "mock:resource4:test4"},
			},
		},
		{
			name: "set field on resource",
			resource: &construct.Resource{
				ID:         construct.ResourceId{Provider: "mock", Type: "resource1", Name: "test1"},
				Properties: make(map[string]interface{}),
			},
			resourceToSet: kbtesting.MockResource4("test4"),
			property:      &knowledgebase.Property{Name: "Res4", Namespace: true, Path: "Res4"},
			step:          knowledgebase.OperationalStep{Direction: knowledgebase.DirectionUpstream},
			initialState:  []any{"mock:resource1:test1", "mock:resource4:test4", "mock:resource4:test4 -> mock:resource1:test1"},
			wantResource: &construct.Resource{
				ID: construct.ResourceId{Provider: "mock", Type: "resource1", Namespace: res4.ID.Name, Name: "test1"},
				Properties: map[string]interface{}{
					"Res4": kbtesting.MockResource4("test4").ID,
				},
			},
			wantGraph: enginetesting.ExpectedGraphs{
				Dataflow:   []any{"mock:resource1:test1", "mock:resource4:test4", "mock:resource4:test4 -> mock:resource1:test1"},
				Deployment: []any{"mock:resource1:test1", "mock:resource4:test4", "mock:resource4:test4 -> mock:resource1:test1"},
			},
		},
		{
			name: "field is not being replaced",
			resource: &construct.Resource{
				ID: construct.ResourceId{Provider: "mock", Type: "resource1", Name: "test1"},
				Properties: map[string]interface{}{
					"Res4": res4.ID,
				},
			},
			initialState:  []any{"mock:resource4:test4", "mock:resource1:test1", "mock:resource4:test4 -> mock:resource1:test1"},
			resourceToSet: res4,
			property:      &knowledgebase.Property{Name: "Res4", Path: "Res4"},
			step:          knowledgebase.OperationalStep{Direction: knowledgebase.DirectionUpstream},
			wantResource: &construct.Resource{
				ID: construct.ResourceId{Provider: "mock", Type: "resource1", Name: "test1"},
				Properties: map[string]interface{}{
					"Res4": res4.ID,
				},
			},
			wantGraph: enginetesting.ExpectedGraphs{
				Dataflow:   []any{"mock:resource4:test4", "mock:resource1:test1", "mock:resource4:test4 -> mock:resource1:test1"},
				Deployment: []any{"mock:resource4:test4", "mock:resource1:test1", "mock:resource4:test4 -> mock:resource1:test1"},
			},
		},
		{
			name: "field is array",
			resource: &construct.Resource{
				ID: construct.ResourceId{Provider: "mock", Type: "resource1", Name: "test1"},
				Properties: map[string]interface{}{
					"Res2s": []construct.ResourceId{kbtesting.MockResource2("test").ID},
				},
			},
			resourceToSet: kbtesting.MockResource2("test2"),
			property:      &knowledgebase.Property{Name: "Res2s", Type: "list(resource)", Path: "Res2s"},
			step:          knowledgebase.OperationalStep{Direction: knowledgebase.DirectionUpstream},
			initialState:  []any{"mock:resource2:test", "mock:resource1:test1", "mock:resource2:test -> mock:resource1:test1"},
			wantResource: &construct.Resource{
				ID: construct.ResourceId{Provider: "mock", Type: "resource1", Name: "test1"},
				Properties: map[string]interface{}{
					"Res2s": []construct.ResourceId{kbtesting.MockResource2("test").ID, kbtesting.MockResource2("test2").ID},
				},
			},
			wantGraph: enginetesting.ExpectedGraphs{
				Dataflow:   []any{"mock:resource2:test", "mock:resource1:test1", "mock:resource2:test -> mock:resource1:test1"},
				Deployment: []any{"mock:resource2:test", "mock:resource1:test1", "mock:resource2:test -> mock:resource1:test1"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			testSol := enginetesting.NewTestSolution()
			testSol.KB.On("GetEdgeTemplate", mock.Anything, mock.Anything).Return(&knowledgebase.EdgeTemplate{}, nil)
			testSol.KB.On("GetResourceTemplate", mock.Anything).Return(&knowledgebase.ResourceTemplate{}, nil)
			testSol.LoadState(t, tt.initialState...)
			ctx := OperationalRuleContext{
				Solution: testSol,
				Property: tt.property,
			}

			err := ctx.setField(tt.resource, tt.resourceToSet, tt.step)
			if !assert.NoError(err) {
				return
			}
			assert.Equal(tt.wantResource, tt.resource)
			tt.wantGraph.AssertEqual(t, testSol)
		})
	}
}

func MockResource1(name string) *construct.Resource {
	return &construct.Resource{
		ID: construct.ResourceId{
			Provider: "mock",
			Type:     "resource1",
			Name:     name,
		},
		Properties: make(construct.Properties),
	}
}

func MockResource2(name string) *construct.Resource {
	return &construct.Resource{
		ID: construct.ResourceId{
			Provider: "mock",
			Type:     "resource2",
			Name:     name,
		},
		Properties: make(construct.Properties),
	}
}

func MockResource3(name string) *construct.Resource {
	return &construct.Resource{
		ID: construct.ResourceId{
			Provider: "mock",
			Type:     "resource3",
			Name:     name,
		},
		Properties: make(construct.Properties),
	}
}

func MockResource4(name string) *construct.Resource {
	return &construct.Resource{
		ID: construct.ResourceId{
			Provider: "mock",
			Type:     "resource4",
			Name:     name,
		},
		Properties: make(construct.Properties),
	}
}

// Defined are a set of resource teampltes that are used for testing
var resource1 = &knowledgebase.ResourceTemplate{
	QualifiedTypeName: "mock:resource1",
	Properties: map[string]knowledgebase.Property{
		"Name": {
			Name:      "Name",
			Type:      "string",
			Namespace: false,
		},
		"Res4": {
			Name:      "Res4",
			Type:      "resource",
			Namespace: true,
		},
		"Res2s": {
			Name: "Res2s",
			Type: "list(resource)",
		},
	},
	Classification: knowledgebase.Classification{
		Is: []string{},
	},
	DeleteContext: knowledgebase.DeleteContext{},
}

var resource2 = &knowledgebase.ResourceTemplate{
	QualifiedTypeName: "mock:resource2",
	Properties: map[string]knowledgebase.Property{
		"Name": {
			Name:      "Name",
			Type:      "string",
			Namespace: false,
		},
	},
	Classification: knowledgebase.Classification{
		Is: []string{},
	},
	DeleteContext: knowledgebase.DeleteContext{},
}

var resource3 = &knowledgebase.ResourceTemplate{
	QualifiedTypeName: "mock:resource3",
	Properties: map[string]knowledgebase.Property{
		"Name": {
			Name:      "Name",
			Type:      "string",
			Namespace: false,
		},
	},
	Classification: knowledgebase.Classification{
		Is: []string{},
	},
	DeleteContext: knowledgebase.DeleteContext{},
}

var resource4 = &knowledgebase.ResourceTemplate{
	QualifiedTypeName: "mock:resource4",
	Properties: map[string]knowledgebase.Property{
		"Name": {
			Name:      "Name",
			Type:      "string",
			Namespace: false,
		},
	},
	Classification: knowledgebase.Classification{
		Is: []string{"role"},
	},
	DeleteContext: knowledgebase.DeleteContext{},
}
