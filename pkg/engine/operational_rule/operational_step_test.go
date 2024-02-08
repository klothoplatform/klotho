package operational_rule

import (
	"testing"

	"github.com/dominikbraun/graph"
	construct "github.com/klothoplatform/klotho/pkg/construct2"
	"github.com/klothoplatform/klotho/pkg/construct2/graphtest"
	"github.com/klothoplatform/klotho/pkg/engine/enginetesting"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base2"
	"github.com/klothoplatform/klotho/pkg/knowledge_base2/properties"
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
			resource: MockResource1("test1"),
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
			resource: MockResource1("test1"),
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
			resource: MockResource1("test1"),
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
			resource: MockResource1("test1"),
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
			AddMockKB(t, &testSol.KB)
			testResources := []*construct.Resource{
				MockResource4("test"),
				MockResource4("test2"),
				MockResource3("test3"),
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
	}{
		{
			name: "downstream",
			resource: &construct.Resource{
				ID: graphtest.ParseId(t, "a:a:a"),
				Properties: map[string]interface{}{
					"Res4": MockResource4("test").ID,
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
		},
		{
			name: "upstream",
			resource: &construct.Resource{
				ID: graphtest.ParseId(t, "a:a:a"),
				Properties: map[string]interface{}{
					"Res4": MockResource4("test").ID,
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
		},
		{
			name: "array",
			resource: &construct.Resource{
				ID: graphtest.ParseId(t, "a:a:a"),
				Properties: map[string]interface{}{
					"Res2s": []construct.ResourceId{MockResource2("test").ID, MockResource2("test2").ID},
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
		},
		{
			name: "array with existing",
			resource: &construct.Resource{
				ID: graphtest.ParseId(t, "a:a:a"),
				Properties: map[string]interface{}{
					"Res2s": []construct.ResourceId{MockResource2("test").ID, MockResource2("test2").ID},
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
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			testSol := enginetesting.NewTestSolution()
			AddMockKB(t, &testSol.KB)
			testSol.LoadState(t, tt.initialState...)
			ctx := OperationalRuleContext{
				Solution: testSol,
			}

			ids, err := ctx.addDependenciesFromProperty(tt.step, tt.resource, tt.propertyName)
			if !assert.NoError(err) {
				return
			}
			tt.want.AssertEqual(t, testSol)
			assert.ElementsMatch(tt.wantIds, ids)
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
					"Res4": MockResource4("test").ID,
				},
			},
			step:         knowledgebase.OperationalStep{Direction: knowledgebase.DirectionDownstream},
			propertyName: "Res4",
			initialState: []any{"mock:resource4:test", "a:a:a", "a:a:a -> mock:resource4:test"},
			want: enginetesting.ExpectedGraphs{
				Dataflow: []any{"a:a:a"},
			},
		},
		{
			name: "upstream",
			resource: &construct.Resource{
				ID: graphtest.ParseId(t, "a:a:a"),
				Properties: map[string]interface{}{
					"Res4": MockResource4("test").ID,
				},
			},
			step:         knowledgebase.OperationalStep{Direction: knowledgebase.DirectionUpstream},
			propertyName: "Res4",
			initialState: []any{"mock:resource4:test", "a:a:a", "mock:resource4:test -> a:a:a"},
			want: enginetesting.ExpectedGraphs{
				Dataflow: []any{"a:a:a"},
			},
		},
		{
			name: "array",
			resource: &construct.Resource{
				ID: graphtest.ParseId(t, "a:a:a"),
				Properties: map[string]interface{}{
					"Res2s": []construct.ResourceId{MockResource2("test").ID, MockResource2("test2").ID},
				},
			},
			step:         knowledgebase.OperationalStep{Direction: knowledgebase.DirectionUpstream},
			propertyName: "Res2s",
			initialState: []any{
				"mock:resource2:test", "a:a:a", "mock:resource2:test2",
				"mock:resource2:test -> a:a:a", "mock:resource2:test2 -> a:a:a",
			},
			want: enginetesting.ExpectedGraphs{
				Dataflow: []any{"a:a:a"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			testSol := enginetesting.NewTestSolution()
			AddMockKB(t, &testSol.KB)
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
			to:   MockResource1("test1"),
			from: MockResource4(""),
			step: knowledgebase.OperationalStep{
				Direction: knowledgebase.DirectionUpstream,
			},
			want: enginetesting.ExpectedGraphs{
				Dataflow: []any{"mock:resource1:test1", "mock:resource4:", "mock:resource4: -> mock:resource1:test1"},
			},
			wantEdge: graph.Edge[construct.ResourceId]{
				Source: graphtest.ParseId(t, "mock:resource4:"), Target: graphtest.ParseId(t, "mock:resource1:test1"),
			},
		},
		{
			name: "downstream",
			to:   MockResource1("test1"),
			from: MockResource4(""),
			step: knowledgebase.OperationalStep{
				Direction: knowledgebase.DirectionDownstream,
			},
			want: enginetesting.ExpectedGraphs{
				Dataflow: []any{"mock:resource1:test1", "mock:resource4:", "mock:resource1:test1 -> mock:resource4:"},
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
			AddMockKB(t, &testSol.KB)
			testSol.LoadState(t, "mock:resource1:test1", "mock:resource4:")
			ctx := OperationalRuleContext{
				Solution: testSol,
			}

			err := ctx.addDependencyForDirection(tt.step, tt.to, tt.from)
			if !assert.NoError(err) {
				return
			}
			tt.want.AssertEqual(t, testSol)
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
			to:           MockResource1("test1"),
			from:         MockResource4(""),
			direction:    knowledgebase.DirectionUpstream,
			initialState: []any{"mock:resource1:test1", "mock:resource4:", "mock:resource4: -> mock:resource1:test1"},
		},
		{
			name:         "downstream",
			to:           MockResource1("test1"),
			from:         MockResource4(""),
			direction:    knowledgebase.DirectionDownstream,
			initialState: []any{"mock:resource1:test1", "mock:resource4:", "mock:resource1:test1 -> mock:resource4:"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			testSol := enginetesting.NewTestSolution()
			AddMockKB(t, &testSol.KB)
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
	res4ToReplace := MockResource4("thisWillBeReplaced")
	res4 := MockResource4("test4")
	tests := []struct {
		name          string
		resource      *construct.Resource
		resourceToSet *construct.Resource
		property      knowledgebase.Property
		initialState  []any
		step          knowledgebase.OperationalStep
		wantResource  *construct.Resource
		wantGraph     enginetesting.ExpectedGraphs
		wantErr       bool
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
			property: &properties.ResourceProperty{
				PropertyDetails: knowledgebase.PropertyDetails{
					Name:      "Res4",
					Path:      "Res4",
					Namespace: true,
				},
			},
			step:         knowledgebase.OperationalStep{Direction: knowledgebase.DirectionDownstream},
			initialState: []any{"mock:resource4:test4", "mock:resource1:test1", "mock:resource1:test1 -> mock:resource4:thisWillBeReplaced"},
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
			property: &properties.ResourceProperty{
				PropertyDetails: knowledgebase.PropertyDetails{
					Name:      "Res4",
					Path:      "Res4",
					Namespace: true,
				},
			},
			step:         knowledgebase.OperationalStep{Direction: knowledgebase.DirectionUpstream},
			initialState: []any{"mock:resource4:test4", "mock:resource1:test1", "mock:resource4:thisWillBeReplaced -> mock:resource1:test1"},
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
			resourceToSet: MockResource4("test4"),
			property: &properties.ResourceProperty{
				PropertyDetails: knowledgebase.PropertyDetails{
					Name:      "Res4",
					Path:      "Res4",
					Namespace: true,
				},
			},
			step:         knowledgebase.OperationalStep{Direction: knowledgebase.DirectionUpstream},
			initialState: []any{"mock:resource1:test1", "mock:resource4:test4", "mock:resource4:test4 -> mock:resource1:test1"},
			wantResource: &construct.Resource{
				ID: construct.ResourceId{Provider: "mock", Type: "resource1", Namespace: res4.ID.Name, Name: "test1"},
				Properties: map[string]interface{}{
					"Res4": MockResource4("test4").ID,
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
			property: &properties.ResourceProperty{
				PropertyDetails: knowledgebase.PropertyDetails{
					Name: "Res4",
					Path: "Res4",
				},
			},
			step: knowledgebase.OperationalStep{Direction: knowledgebase.DirectionUpstream},
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
					"Res2s": []construct.ResourceId{MockResource2("test").ID},
				},
			},
			resourceToSet: MockResource2("test2"),
			property: &properties.ListProperty{
				ItemProperty: &properties.ResourceProperty{},
				PropertyDetails: knowledgebase.PropertyDetails{
					Name: "Res2s",
					Path: "Res2s",
				},
			},
			step:         knowledgebase.OperationalStep{Direction: knowledgebase.DirectionUpstream},
			initialState: []any{"mock:resource2:test", "mock:resource1:test1", "mock:resource2:test -> mock:resource1:test1"},
			wantResource: &construct.Resource{
				ID: construct.ResourceId{Provider: "mock", Type: "resource1", Name: "test1"},
				Properties: map[string]interface{}{
					"Res2s": []construct.ResourceId{MockResource2("test").ID, MockResource2("test2").ID},
				},
			},
			wantGraph: enginetesting.ExpectedGraphs{
				Dataflow:   []any{"mock:resource2:test", "mock:resource1:test1", "mock:resource2:test -> mock:resource1:test1"},
				Deployment: []any{"mock:resource2:test", "mock:resource1:test1", "mock:resource2:test -> mock:resource1:test1"},
			},
		},
		{
			name: "imported resource gets ignored",
			resource: &construct.Resource{
				ID: construct.ResourceId{Provider: "mock", Type: "resource1", Name: "test1"},
				Properties: map[string]interface{}{
					"Res4": res4.ID,
				},
				Imported: true,
			},
			resourceToSet: res4,
			property: &properties.ResourceProperty{
				PropertyDetails: knowledgebase.PropertyDetails{
					Name:      "Res4",
					Path:      "Res4",
					Namespace: true,
				},
			},
			step:         knowledgebase.OperationalStep{Direction: knowledgebase.DirectionDownstream},
			initialState: []any{"mock:resource4:test4", "mock:resource1:test1"},
			wantResource: &construct.Resource{
				ID: construct.ResourceId{Provider: "mock", Type: "resource1", Namespace: res4.ID.Name, Name: "test1"},
				Properties: map[string]interface{}{
					"Res4": res4.ID,
				},
				Imported: true,
			},
			wantGraph: enginetesting.ExpectedGraphs{
				Dataflow:   []any{"mock:resource1:test1", "mock:resource4:test4"},
				Deployment: []any{"mock:resource1:test1", "mock:resource4:test4"},
			},
		},
		{
			name: "imported resource throws error if field is changing",
			resource: &construct.Resource{
				ID: construct.ResourceId{Provider: "mock", Type: "resource1", Name: "test1"},
				Properties: map[string]interface{}{
					"Res4": res4ToReplace.ID,
				},
				Imported: true,
			},
			resourceToSet: res4,
			property: &properties.ResourceProperty{
				PropertyDetails: knowledgebase.PropertyDetails{
					Name:      "Res4",
					Path:      "Res4",
					Namespace: true,
				},
			},
			step:         knowledgebase.OperationalStep{Direction: knowledgebase.DirectionDownstream},
			initialState: []any{"mock:resource4:test4", "mock:resource1:test1", "mock:resource1:test1 -> mock:resource4:thisWillBeReplaced"},
			wantErr:      true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			testSol := enginetesting.NewTestSolution()
			AddMockKB(t, &testSol.KB)
			testSol.LoadState(t, tt.initialState...)
			ctx := OperationalRuleContext{
				Solution: testSol,
				Property: tt.property,
			}

			err := ctx.SetField(tt.resource, tt.resourceToSet, tt.step)
			if tt.wantErr {
				assert.Error(err)
				return
			}
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
	Properties: knowledgebase.Properties{
		"Name": &properties.StringProperty{
			PropertyDetails: knowledgebase.PropertyDetails{
				Name: "Name",
			},
		},
		"Res4": &properties.ResourceProperty{
			PropertyDetails: knowledgebase.PropertyDetails{
				Name:      "Res4",
				Path:      "Res4",
				Namespace: true,
			},
		},
		"Res2s": &properties.ListProperty{
			ItemProperty: &properties.ResourceProperty{},
			PropertyDetails: knowledgebase.PropertyDetails{
				Name: "Res2s",
				Path: "Res2s",
			},
		},
	},
	Classification: knowledgebase.Classification{
		Is: []string{},
	},
	DeleteContext: knowledgebase.DeleteContext{},
}

var resource2 = &knowledgebase.ResourceTemplate{
	QualifiedTypeName: "mock:resource2",
	Properties: knowledgebase.Properties{
		"Name": &properties.StringProperty{
			PropertyDetails: knowledgebase.PropertyDetails{
				Name: "Name",
			},
		},
	},
	Classification: knowledgebase.Classification{
		Is: []string{},
	},
	DeleteContext: knowledgebase.DeleteContext{},
}

var resource3 = &knowledgebase.ResourceTemplate{
	QualifiedTypeName: "mock:resource3",
	Properties: knowledgebase.Properties{
		"Name": &properties.StringProperty{
			PropertyDetails: knowledgebase.PropertyDetails{
				Name: "Name",
			},
		},
	},
	Classification: knowledgebase.Classification{
		Is: []string{},
	},
	DeleteContext: knowledgebase.DeleteContext{},
}

var resource4 = &knowledgebase.ResourceTemplate{
	QualifiedTypeName: "mock:resource4",
	Properties: knowledgebase.Properties{
		"Name": &properties.StringProperty{
			PropertyDetails: knowledgebase.PropertyDetails{
				Name: "Name",
			},
		},
	},
	Classification: knowledgebase.Classification{
		Is: []string{"role"},
	},
	DeleteContext: knowledgebase.DeleteContext{},
}

func AddMockKB(t *testing.T, kb *enginetesting.MockKB) {
	kb.On("GetEdgeTemplate", mock.Anything, mock.Anything).Return(&knowledgebase.EdgeTemplate{}, nil)
	kb.On("GetResourceTemplate", mock.MatchedBy(graphtest.ParseId(t, "mock:resource1").Matches)).Return(resource1, nil)
	kb.On("GetResourceTemplate", mock.MatchedBy(graphtest.ParseId(t, "mock:resource2").Matches)).Return(resource2, nil)
	kb.On("GetResourceTemplate", mock.MatchedBy(graphtest.ParseId(t, "mock:resource3").Matches)).Return(resource3, nil)
	kb.On("GetResourceTemplate", mock.MatchedBy(graphtest.ParseId(t, "mock:resource4").Matches)).Return(resource4, nil)
	// a:a:a is used in a few tests as the base resource
	kb.On("GetResourceTemplate", graphtest.ParseId(t, "a:a:a")).Return(&knowledgebase.ResourceTemplate{}, nil)
	kb.On("ListResources").Return([]*knowledgebase.ResourceTemplate{resource1, resource2, resource3, resource4}, nil)
}
