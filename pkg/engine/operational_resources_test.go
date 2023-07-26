package engine

import (
	"fmt"
	"testing"

	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/core/coretesting"
	"github.com/klothoplatform/klotho/pkg/engine/enginetesting"
	"github.com/klothoplatform/klotho/pkg/graph"
	"github.com/klothoplatform/klotho/pkg/provider"
	"github.com/stretchr/testify/assert"
)

func Test_handleOperationalRule(t *testing.T) {
	tests := []struct {
		name                 string
		rule                 core.OperationalRule
		resource             *enginetesting.MockResource5
		parent               core.Resource
		existingDependencies []graph.Edge[core.Resource]
		want                 coretesting.ResourcesExpectation
		check                func(assert *assert.Assertions, resource enginetesting.MockResource5)
		wantErr              []error
	}{
		{
			name: "only one none exists",
			rule: core.OperationalRule{
				Enforcement:   core.ExactlyOne,
				Direction:     core.Downstream,
				ResourceTypes: []string{"mock1"},
				SetField:      "Mock1",
				UnsatisfiedAction: core.UnsatisfiedAction{
					Operation: core.CreateUnsatisfiedResource,
				},
			},
			resource: &enginetesting.MockResource5{Name: "this"},
			wantErr: []error{&core.OperationalResourceError{
				Resource:  &enginetesting.MockResource5{Name: "this"},
				Needs:     []string{"mock1"},
				Direction: core.Downstream,
				Count:     1,
				Cause:     fmt.Errorf("rule with enforcement exactly one has less than the required number of resources of type [mock1]  or classifications [], 0 for resource mock:mock5:this"),
			}},
		},
		{
			name: "only one one exists",
			rule: core.OperationalRule{
				Enforcement:   core.ExactlyOne,
				Direction:     core.Downstream,
				ResourceTypes: []string{"mock1"},
				SetField:      "Mock1",
			},
			resource: &enginetesting.MockResource5{Name: "this"},
			existingDependencies: []graph.Edge[core.Resource]{
				{Source: &enginetesting.MockResource5{Name: "this"}, Destination: &enginetesting.MockResource1{Name: "that"}},
			},
			want: coretesting.ResourcesExpectation{
				Nodes: []string{"mock:mock1:that", "mock:mock5:this"},
				Deps: []coretesting.StringDep{
					{Source: "mock:mock5:this", Destination: "mock:mock1:that"},
				},
			},
			check: func(assert *assert.Assertions, resource enginetesting.MockResource5) {
				assert.Equal(&enginetesting.MockResource1{Name: "that"}, resource.Mock1)
			},
		},
		{
			name: "only one multiple exist error",
			rule: core.OperationalRule{
				Enforcement:   core.ExactlyOne,
				Direction:     core.Downstream,
				ResourceTypes: []string{"mock1"},
				SetField:      "Mock1",
			},
			resource: &enginetesting.MockResource5{Name: "this"},
			existingDependencies: []graph.Edge[core.Resource]{
				{Source: &enginetesting.MockResource5{Name: "this"}, Destination: &enginetesting.MockResource1{Name: "that"}},
				{Source: &enginetesting.MockResource5{Name: "this"}, Destination: &enginetesting.MockResource1{Name: "that2"}},
			},
			wantErr: []error{fmt.Errorf("rule with enforcement only_one has more than one resource for rule exactly_one [mock1] for resource mock:mock5:this")},
		},
		{
			name: "if one none exists",
			rule: core.OperationalRule{
				Enforcement:   core.Conditional,
				Direction:     core.Downstream,
				ResourceTypes: []string{"mock1"},
				SetField:      "Mock1",
			},
			resource: &enginetesting.MockResource5{Name: "this"},
		},
		{
			name: "if one one exists",
			rule: core.OperationalRule{
				Enforcement:   core.Conditional,
				Direction:     core.Downstream,
				ResourceTypes: []string{"mock1"},
				SetField:      "Mock1",
			},
			resource: &enginetesting.MockResource5{Name: "this"},
			existingDependencies: []graph.Edge[core.Resource]{
				{Source: &enginetesting.MockResource5{Name: "this"}, Destination: &enginetesting.MockResource1{Name: "that"}},
			},
			want: coretesting.ResourcesExpectation{
				Nodes: []string{"mock:mock1:that", "mock:mock5:this"},
				Deps: []coretesting.StringDep{
					{Source: "mock:mock5:this", Destination: "mock:mock1:that"},
				},
			},
			check: func(assert *assert.Assertions, resource enginetesting.MockResource5) {
				assert.Equal(&enginetesting.MockResource1{Name: "that"}, resource.Mock1)
			},
		},
		{
			name: "if one one exists with sub rules",
			rule: core.OperationalRule{
				Enforcement:            core.Conditional,
				Direction:              core.Downstream,
				ResourceTypes:          []string{"mock3"},
				RemoveDirectDependency: true,
				Rules: []core.OperationalRule{
					{
						Enforcement:   core.AnyAvailable,
						Direction:     core.Downstream,
						ResourceTypes: []string{"mock2"},
						SetField:      "Mock2s",
						NumNeeded:     2,
						UnsatisfiedAction: core.UnsatisfiedAction{
							Operation: core.CreateUnsatisfiedResource,
						},
					},
				},
			},
			resource: &enginetesting.MockResource5{Name: "this"},
			existingDependencies: []graph.Edge[core.Resource]{
				{Source: &enginetesting.MockResource5{Name: "this"}, Destination: &enginetesting.MockResource3{Name: "that"}},
			},
			wantErr: []error{&core.OperationalResourceError{
				Resource:  &enginetesting.MockResource5{Name: "this"},
				Count:     2,
				Direction: core.Downstream,
				Needs:     []string{"mock2"},
				Parent:    &enginetesting.MockResource3{Name: "that"},
				Cause:     fmt.Errorf("rule with enforcement any has less than the required number of resources of type [mock2]  or classifications [], 0 for resource mock:mock5:this"),
			}},
		},
		{
			name: "if one multiple exist error",
			rule: core.OperationalRule{

				Enforcement:   core.Conditional,
				Direction:     core.Downstream,
				ResourceTypes: []string{"mock1"},
				SetField:      "Mock1",
			},
			resource: &enginetesting.MockResource5{Name: "this"},
			existingDependencies: []graph.Edge[core.Resource]{
				{Source: &enginetesting.MockResource5{Name: "this"}, Destination: &enginetesting.MockResource1{Name: "that"}},
				{Source: &enginetesting.MockResource5{Name: "this"}, Destination: &enginetesting.MockResource1{Name: "that2"}},
			},
			want: coretesting.ResourcesExpectation{
				Nodes: []string{"mock:mock1:that", "mock:mock1:that2", "mock:mock5:this"},
				Deps: []coretesting.StringDep{
					{Source: "mock:mock5:this", Destination: "mock:mock1:that"},
					{Source: "mock:mock5:this", Destination: "mock:mock1:that2"},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			mp := &enginetesting.MockProvider{}
			engine := NewEngine(map[string]provider.Provider{
				mp.Name(): mp,
			}, enginetesting.MockKB, core.ListAllConstructs())
			engine.ClassificationDocument = enginetesting.BaseClassificationDocument

			dag := core.NewResourceGraph()
			for _, dep := range tt.existingDependencies {
				dag.AddDependency(dep.Source, dep.Destination)
			}

			err := engine.handleOperationalRule(tt.resource, tt.rule, dag, tt.parent)
			if tt.wantErr != nil {
				assert.Greater(len(err), 0)
				assert.Equal(err, tt.wantErr)
				return
			}
			if !assert.Len(err, 0) {
				return
			}
			tt.want.Assert(t, dag)
		})
	}
}

func Test_handleOperationalResourceError(t *testing.T) {
	tests := []struct {
		name                 string
		ore                  *core.OperationalResourceError
		existingDependencies []graph.Edge[core.Resource]
		want                 coretesting.ResourcesExpectation
		wantErr              bool
	}{
		{
			name: "needs one downstream",
			ore: &core.OperationalResourceError{
				Resource:  &enginetesting.MockResource5{Name: "this"},
				Direction: core.Downstream,
				Needs:     []string{"mock1"},
				Count:     1,
				Cause:     fmt.Errorf("0"),
			},
			want: coretesting.ResourcesExpectation{
				Nodes: []string{"mock:mock5:this", "mock:mock1:mock1-this"},
				Deps: []coretesting.StringDep{
					{Source: "mock:mock5:this", Destination: "mock:mock1:mock1-this"},
				},
			},
		},
		{
			name: "needs multiple downstream",
			ore: &core.OperationalResourceError{
				Resource:  &enginetesting.MockResource5{Name: "this"},
				Direction: core.Downstream,
				Needs:     []string{"mock2"},
				Count:     2,
				Cause:     fmt.Errorf("0"),
			},
			want: coretesting.ResourcesExpectation{
				Nodes: []string{"mock:mock5:this", "mock:mock2:mock2-this", "mock:mock2:mock2-this0"},
				Deps: []coretesting.StringDep{
					{Source: "mock:mock5:this", Destination: "mock:mock2:mock2-this"},
					{Source: "mock:mock5:this", Destination: "mock:mock2:mock2-this0"},
				},
			},
		},
		{
			name: "needs parents resource",
			ore: &core.OperationalResourceError{
				Resource:  &enginetesting.MockResource5{Name: "this"},
				Direction: core.Downstream,
				Needs:     []string{"mock1"},
				Count:     1,
				Parent:    &enginetesting.MockResource3{Name: "parent"},
				Cause:     fmt.Errorf("0"),
			},
			existingDependencies: []graph.Edge[core.Resource]{
				{Source: &enginetesting.MockResource1{Name: "child"}, Destination: &enginetesting.MockResource3{Name: "parent"}},
			},
			want: coretesting.ResourcesExpectation{
				Nodes: []string{"mock:mock5:this", "mock:mock1:child", "mock:mock3:parent"},
				Deps: []coretesting.StringDep{
					{Source: "mock:mock5:this", Destination: "mock:mock1:child"},
					{Source: "mock:mock1:child", Destination: "mock:mock3:parent"},
				},
			},
		},
		{
			name: "needs 2 but parent only has 1 resource",
			ore: &core.OperationalResourceError{
				Resource:  &enginetesting.MockResource5{Name: "this"},
				Direction: core.Downstream,
				Needs:     []string{"mock1"},
				Count:     2,
				Parent:    &enginetesting.MockResource3{Name: "parent"},
				Cause:     fmt.Errorf("0"),
			},
			existingDependencies: []graph.Edge[core.Resource]{
				{Source: &enginetesting.MockResource1{Name: "child"}, Destination: &enginetesting.MockResource3{Name: "parent"}},
				{Source: &enginetesting.MockResource1{Name: "child2"}, Destination: &enginetesting.MockResource3{Name: "parent2"}},
			},
			want: coretesting.ResourcesExpectation{
				Nodes: []string{"mock:mock5:this", "mock:mock1:mock1-this", "mock:mock1:child", "mock:mock1:child2", "mock:mock3:parent", "mock:mock3:parent2"},
				Deps: []coretesting.StringDep{
					{Source: "mock:mock5:this", Destination: "mock:mock1:child"},
					{Source: "mock:mock5:this", Destination: "mock:mock1:mock1-this"},
					{Source: "mock:mock1:child", Destination: "mock:mock3:parent"},
					{Source: "mock:mock1:mock1-this", Destination: "mock:mock3:parent"},
					{Source: "mock:mock1:child2", Destination: "mock:mock3:parent2"},
				},
			},
		},
		{
			name: "chooses existing resource to satisfy needs",
			ore: &core.OperationalResourceError{
				Resource:  &enginetesting.MockResource5{Name: "this"},
				Direction: core.Downstream,
				Needs:     []string{"mock1"},
				Count:     2,
				Cause:     fmt.Errorf("0"),
			},
			existingDependencies: []graph.Edge[core.Resource]{
				{Source: &enginetesting.MockResource1{Name: "child"}, Destination: &enginetesting.MockResource3{Name: "parent"}},
				{Source: &enginetesting.MockResource1{Name: "child2"}, Destination: &enginetesting.MockResource3{Name: "parent2"}},
			},
			want: coretesting.ResourcesExpectation{
				Nodes: []string{"mock:mock5:this", "mock:mock1:child", "mock:mock1:child2", "mock:mock3:parent", "mock:mock3:parent2"},
				Deps: []coretesting.StringDep{
					{Source: "mock:mock5:this", Destination: "mock:mock1:child"},
					{Source: "mock:mock5:this", Destination: "mock:mock1:child2"},
					{Source: "mock:mock1:child", Destination: "mock:mock3:parent"},
					{Source: "mock:mock1:child2", Destination: "mock:mock3:parent2"},
				},
			},
		},
		{
			name: "must create new resource to satisfy needs",
			ore: &core.OperationalResourceError{
				Resource:   &enginetesting.MockResource5{Name: "this"},
				Direction:  core.Downstream,
				Needs:      []string{"mock1"},
				Count:      2,
				MustCreate: true,
				Cause:      fmt.Errorf("0"),
			},
			existingDependencies: []graph.Edge[core.Resource]{
				{Source: &enginetesting.MockResource1{Name: "child"}, Destination: &enginetesting.MockResource3{Name: "parent"}},
				{Source: &enginetesting.MockResource1{Name: "child2"}, Destination: &enginetesting.MockResource3{Name: "parent2"}},
			},
			want: coretesting.ResourcesExpectation{
				Nodes: []string{"mock:mock5:this", "mock:mock1:mock1-this0", "mock:mock1:mock1-this", "mock:mock1:child", "mock:mock1:child2", "mock:mock3:parent", "mock:mock3:parent2"},
				Deps: []coretesting.StringDep{
					{Source: "mock:mock5:this", Destination: "mock:mock1:mock1-this0"},
					{Source: "mock:mock5:this", Destination: "mock:mock1:mock1-this"},
					{Source: "mock:mock1:child", Destination: "mock:mock3:parent"},
					{Source: "mock:mock1:child2", Destination: "mock:mock3:parent2"},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			mp := &enginetesting.MockProvider{}
			engine := NewEngine(map[string]provider.Provider{
				mp.Name(): mp,
			}, enginetesting.MockKB, core.ListAllConstructs())
			engine.ClassificationDocument = enginetesting.BaseClassificationDocument

			dag := core.NewResourceGraph()
			for _, dep := range tt.existingDependencies {
				dag.AddDependency(dep.Source, dep.Destination)
			}

			err := engine.handleOperationalResourceError(tt.ore, dag)
			if tt.wantErr {
				assert.Error(err)
				return
			}
			if !assert.NoError(err) {
				return
			}
			tt.want.Assert(t, dag)
		})
	}
}

func Test_TemplateConfigure(t *testing.T) {
	tests := []struct {
		name     string
		resource *enginetesting.MockResource6
		template core.ResourceTemplate
		want     *enginetesting.MockResource6
	}{
		{
			name:     "simple values",
			resource: &enginetesting.MockResource6{},
			template: core.ResourceTemplate{
				Configuration: []core.Configuration{
					{Field: "Field1", Value: 1},
					{Field: "Field2", Value: "two"},
					{Field: "Field3", Value: true},
				},
			},
			want: &enginetesting.MockResource6{
				Field1: 1,
				Field2: "two",
				Field3: true,
			},
		},
		{
			name:     "simple array",
			resource: &enginetesting.MockResource6{},
			template: core.ResourceTemplate{
				Configuration: []core.Configuration{
					{Field: "Arr1", Value: []string{"1", "2", "3"}},
				},
			},
			want: &enginetesting.MockResource6{
				Arr1: []string{"1", "2", "3"},
			},
		},
		{
			name:     "struct array",
			resource: &enginetesting.MockResource6{},
			template: core.ResourceTemplate{
				Configuration: []core.Configuration{
					{Field: "Arr2", Value: []map[string]interface{}{
						{
							"Field1": 1,
							"Field2": "two",
							"Field3": true,
						},
						{
							"Field1": 2,
							"Field2": "three",
							"Field3": false,
						},
					}},
				},
			},
			want: &enginetesting.MockResource6{
				Arr2: []enginetesting.TestRes1{
					{
						Field1: 1,
						Field2: "two",
						Field3: true,
					},
					{
						Field1: 2,
						Field2: "three",
						Field3: false,
					},
				},
			},
		},
		{
			name:     "pointer array",
			resource: &enginetesting.MockResource6{},
			template: core.ResourceTemplate{
				Configuration: []core.Configuration{
					{Field: "Arr3", Value: []map[string]interface{}{
						{
							"Field1": 1,
							"Field2": "two",
							"Field3": true,
						},
						{
							"Field1": 2,
							"Field2": "three",
							"Field3": false,
						},
					}},
				},
			},
			want: &enginetesting.MockResource6{
				Arr3: []*enginetesting.TestRes1{
					{
						Field1: 1,
						Field2: "two",
						Field3: true,
					},
					{
						Field1: 2,
						Field2: "three",
						Field3: false,
					},
				},
			},
		},
		{
			name:     "struct",
			resource: &enginetesting.MockResource6{},
			template: core.ResourceTemplate{
				Configuration: []core.Configuration{
					{Field: "Struct1", Value: map[string]interface{}{
						"Field1": 1,
						"Field2": "two",
						"Field3": true,
						"Arr1":   []string{"1", "2", "3"},
					}},
				},
			},
			want: &enginetesting.MockResource6{
				Struct1: enginetesting.TestRes1{
					Field1: 1,
					Field2: "two",
					Field3: true,
					Arr1:   []string{"1", "2", "3"},
				},
			},
		},
		{
			name:     "pointer",
			resource: &enginetesting.MockResource6{},
			template: core.ResourceTemplate{
				Configuration: []core.Configuration{
					{Field: "Struct2", Value: map[string]interface{}{
						"Field1": 1,
						"Field2": "two",
						"Field3": true,
						"Arr1":   []string{"1", "2", "3"},
					}},
				},
			},
			want: &enginetesting.MockResource6{
				Struct2: &enginetesting.TestRes1{
					Field1: 1,
					Field2: "two",
					Field3: true,
					Arr1:   []string{"1", "2", "3"},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)

			err := TemplateConfigure(tt.resource, tt.template)
			if !assert.NoError(err) {
				return
			}
			assert.Equal(tt.want, tt.resource)
		})
	}
}
