package operational_rule

import (
	"reflect"
	"testing"

	construct "github.com/klothoplatform/klotho/pkg/construct2"
	"github.com/klothoplatform/klotho/pkg/engine2/enginetesting"
	kbtesting "github.com/klothoplatform/klotho/pkg/engine2/enginetesting/test_kb"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func Test_handleOperationalResourceAction(t *testing.T) {
	tests := []struct {
		name     string
		mocks    []mock.Call
		resource *construct.Resource
		action   OperationalResourceAction
	}{
		{
			name:     "upstream exact resource that exists in the graph",
			resource: kbtesting.MockResource1("test1"),
			action: OperationalResourceAction{
				Step: &knowledgebase.OperationalStep{
					Direction: knowledgebase.Downstream,
					Resources: []string{"mock:resource4:test"},
					NumNeeded: 1,
				},
			},
			mocks: []mock.Call{
				{
					Method:          "GetResource",
					Arguments:       []interface{}{construct.ResourceId{Provider: "mock", Type: "resource4", Name: "test"}},
					ReturnArguments: mock.Arguments{kbtesting.MockResource4("test"), nil},
				},
				{
					Method:          "AddDependency",
					Arguments:       []interface{}{kbtesting.MockResource1("test1"), kbtesting.MockResource4("test")},
					ReturnArguments: mock.Arguments{nil},
				},
			},
		},
		{
			name:     "upstream unique resources from types",
			resource: kbtesting.MockResource1("test1"),
			action: OperationalResourceAction{
				Step: &knowledgebase.OperationalStep{
					Direction: knowledgebase.Downstream,
					Resources: []string{"mock:resource4"},
					NumNeeded: 2,
					Unique:    true,
				},
			},
			mocks: []mock.Call{
				{
					Method:          "AddDependency",
					Arguments:       []interface{}{kbtesting.MockResource1("test1"), kbtesting.MockResource4("resource4-test1-2")},
					ReturnArguments: mock.Arguments{nil},
				},
				{
					Method:          "AddDependency",
					Arguments:       []interface{}{kbtesting.MockResource1("test1"), kbtesting.MockResource4("resource4-test1-3")},
					ReturnArguments: mock.Arguments{nil},
				},
				{
					Method: "ListResources",
					ReturnArguments: mock.Arguments{
						[]*construct.Resource{
							kbtesting.MockResource4("test"),
							kbtesting.MockResource4("test2"),
						}, nil,
					},
				},
				{
					Method: "ListResources",
					ReturnArguments: mock.Arguments{
						[]*construct.Resource{
							kbtesting.MockResource4("test"),
							kbtesting.MockResource4("test2"),
							kbtesting.MockResource4("resource4-test1-2"),
						}, nil,
					},
				},
				{
					Method: "ListResources",
					ReturnArguments: mock.Arguments{
						[]*construct.Resource{
							kbtesting.MockResource4("test"),
							kbtesting.MockResource4("test2"),
							kbtesting.MockResource4("resource4-test1-2"),
						}, nil,
					},
				},
			},
		},
		{
			name:     "upstream resources from types, choose from available",
			resource: kbtesting.MockResource1("test1"),
			action: OperationalResourceAction{
				Step: &knowledgebase.OperationalStep{
					Direction: knowledgebase.Downstream,
					Resources: []string{"mock:resource4"},
					NumNeeded: 2,
				},
			},
			mocks: []mock.Call{
				{
					Method:          "AddDependency",
					Arguments:       []interface{}{kbtesting.MockResource1("test1"), kbtesting.MockResource4("test")},
					ReturnArguments: mock.Arguments{nil},
				},
				{
					Method:          "AddDependency",
					Arguments:       []interface{}{kbtesting.MockResource1("test1"), kbtesting.MockResource4("test2")},
					ReturnArguments: mock.Arguments{nil},
				},
				{
					Method: "ListResources",
					ReturnArguments: mock.Arguments{
						[]*construct.Resource{
							kbtesting.MockResource4("test"),
							kbtesting.MockResource4("test2"),
						}, nil,
					},
				},
			},
		},
		{
			name:     "upstream resources from types, none available, will create",
			resource: kbtesting.MockResource1("test1"),
			action: OperationalResourceAction{
				Step: &knowledgebase.OperationalStep{
					Direction: knowledgebase.Downstream,
					Resources: []string{"mock:resource4"},
					NumNeeded: 2,
				},
			},
			mocks: []mock.Call{
				{
					Method:          "AddDependency",
					Arguments:       []interface{}{kbtesting.MockResource1("test1"), kbtesting.MockResource4("resource4-0")},
					ReturnArguments: mock.Arguments{nil},
				},
				{
					Method:          "AddDependency",
					Arguments:       []interface{}{kbtesting.MockResource1("test1"), kbtesting.MockResource4("resource4-1")},
					ReturnArguments: mock.Arguments{nil},
				},
				{
					Method: "ListResources",
					ReturnArguments: mock.Arguments{
						[]*construct.Resource{}, nil,
					},
				},
				{
					Method: "ListResources",
					ReturnArguments: mock.Arguments{
						[]*construct.Resource{}, nil,
					},
				},
				{
					Method: "ListResources",
					ReturnArguments: mock.Arguments{
						[]*construct.Resource{
							kbtesting.MockResource4("resource4-0"),
						}, nil,
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			g := &enginetesting.MockGraph{}
			ctx := OperationalRuleContext{
				ConfigCtx: knowledgebase.ConfigTemplateContext{},
				Graph:     g,
				KB:        kbtesting.TestKB,
			}
			for _, mock := range tt.mocks {
				g.On(mock.Method, mock.Arguments...).Return(mock.ReturnArguments...).Once()

			}

			err := ctx.handleOperationalResourceAction(tt.resource, tt.action)
			if !assert.NoError(err) {
				return
			}
			for _, mock := range tt.mocks {
				g.AssertCalled(t, mock.Method, mock.Arguments...)
			}
			g.AssertExpectations(t)
		})
	}
}

func Test_findResourcesWhichSatisfyStepClassifications(t *testing.T) {
	tests := []struct {
		name     string
		step     knowledgebase.OperationalStep
		resource *construct.Resource
		want     []construct.ResourceId
	}{
		{
			name:     "upstream",
			resource: kbtesting.MockResource1("test1"),
			step: knowledgebase.OperationalStep{
				Direction:       knowledgebase.Upstream,
				Classifications: []string{"role"},
			},
			want: []construct.ResourceId{},
		},
		{
			name:     "downstream",
			resource: kbtesting.MockResource1("test1"),
			step: knowledgebase.OperationalStep{
				Direction:       knowledgebase.Downstream,
				Classifications: []string{"role"},
			},
			want: []construct.ResourceId{
				{Provider: "mock", Type: "resource4"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			g := &enginetesting.MockGraph{}
			ctx := OperationalRuleContext{
				Graph: g,
				KB:    kbtesting.TestKB,
			}

			result := ctx.findResourcesWhichSatisfyStepClassifications(&tt.step, tt.resource)
			assert.ElementsMatch(tt.want, result)
		})
	}
}

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
				Direction: knowledgebase.Upstream,
				Resources: []string{"mock:resource4"},
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
				Direction: knowledgebase.Downstream,
				Resources: []string{"mock:resource4"},
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
				Direction:       knowledgebase.Downstream,
				Classifications: []string{"role"},
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
				Direction:       knowledgebase.Upstream,
				Classifications: []string{"role"},
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
			g := &enginetesting.MockGraph{}
			ctx := OperationalRuleContext{
				Graph: g,
				KB:    kbtesting.TestKB,
			}

			g.On("Upstream", mock.Anything, mock.Anything).Return(
				[]*construct.Resource{
					kbtesting.MockResource4("test"),
					kbtesting.MockResource4("test2"),
					kbtesting.MockResource3("test3"),
				}, nil,
			)
			g.On("Downstream", mock.Anything, mock.Anything).Return(
				[]*construct.Resource{
					kbtesting.MockResource4("test"),
					kbtesting.MockResource4("test2"),
					kbtesting.MockResource3("test3"),
				}, nil,
			)

			result, err := ctx.getResourcesForStep(&tt.step, tt.resource)
			if err != nil {
				t.Fatal(err)
			}
			assert.ElementsMatch(tt.want, result)
			if tt.step.Direction == knowledgebase.Downstream {
				g.AssertCalled(t, "Downstream", tt.resource, 3)
			} else {
				g.AssertCalled(t, "Upstream", tt.resource, 3)
			}
		})
	}
}

func Test_addDependenciesFromProperty(t *testing.T) {
	tests := []struct {
		name         string
		step         knowledgebase.OperationalStep
		resource     *construct.Resource
		propertyName string
	}{
		{
			name: "downstream",
			resource: &construct.Resource{Properties: map[string]interface{}{
				"Res4": kbtesting.MockResource4("test"),
			}},
			step:         knowledgebase.OperationalStep{Direction: knowledgebase.Downstream},
			propertyName: "Res4",
		},
		{
			name: "upstream",
			resource: &construct.Resource{Properties: map[string]interface{}{
				"Res4": kbtesting.MockResource4("test"),
			}},
			step:         knowledgebase.OperationalStep{Direction: knowledgebase.Upstream},
			propertyName: "Res4",
		},
		{
			name: "array",
			resource: &construct.Resource{Properties: map[string]interface{}{
				"Res2s": []*construct.Resource{kbtesting.MockResource2("test"), kbtesting.MockResource2("test2")},
			}},
			step:         knowledgebase.OperationalStep{Direction: knowledgebase.Upstream},
			propertyName: "Res2s",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := &enginetesting.MockGraph{}
			ctx := OperationalRuleContext{
				Graph: g,
			}

			g.On("AddDependency", mock.Anything, mock.Anything).Return(nil)

			var currPropertyVal *construct.Resource
			var currPropertyArr []*construct.Resource
			val, err := tt.resource.GetProperty(tt.propertyName)
			if err != nil {
				t.Fatal(err)
			}
			fieldVal := reflect.ValueOf(val)
			if fieldVal.Kind() == reflect.Slice || fieldVal.Kind() == reflect.Array {
				currPropertyArr = fieldVal.Interface().([]*construct.Resource)
			} else {
				if !fieldVal.IsNil() {
					currPropertyVal = fieldVal.Interface().(*construct.Resource)
				}
			}

			ctx.addDependenciesFromProperty(&tt.step, tt.resource, tt.propertyName)

			if currPropertyVal != nil {
				if tt.step.Direction == knowledgebase.Upstream {
					g.AssertCalled(t, "AddDependency", currPropertyVal, tt.resource)
				} else {
					g.AssertCalled(t, "AddDependency", tt.resource, currPropertyVal)
				}
			} else {
				for _, res := range currPropertyArr {
					if tt.step.Direction == knowledgebase.Upstream {
						g.AssertCalled(t, "AddDependency", res, tt.resource)
					} else {
						g.AssertCalled(t, "AddDependency", tt.resource, res)
					}
				}
			}
		})
	}
}

func Test_clearProperty(t *testing.T) {
	tests := []struct {
		name         string
		step         knowledgebase.OperationalStep
		resource     *construct.Resource
		propertyName string
	}{
		{
			name: "downstream",
			resource: &construct.Resource{Properties: map[string]interface{}{
				"Res4": kbtesting.MockResource4("test"),
			}},
			step:         knowledgebase.OperationalStep{Direction: knowledgebase.Downstream},
			propertyName: "Res4",
		},
		{
			name: "upstream",
			resource: &construct.Resource{Properties: map[string]interface{}{
				"Res4": kbtesting.MockResource4("test"),
			}},
			step:         knowledgebase.OperationalStep{Direction: knowledgebase.Upstream},
			propertyName: "Res4",
		},
		{
			name: "array",
			resource: &construct.Resource{Properties: map[string]interface{}{
				"Res2s": []*construct.Resource{kbtesting.MockResource2("test"), kbtesting.MockResource2("test2")},
			}},
			step:         knowledgebase.OperationalStep{Direction: knowledgebase.Upstream},
			propertyName: "Res2s",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := &enginetesting.MockGraph{}
			ctx := OperationalRuleContext{
				Graph: g,
			}

			g.On("RemoveDependency", mock.Anything, mock.Anything).Return(nil)

			originalId := tt.resource.ID
			var currPropertyVal *construct.Resource
			var currPropertyArr []*construct.Resource
			val, err := tt.resource.GetProperty(tt.propertyName)
			if err != nil {
				t.Fatal(err)
			}
			fieldVal := reflect.ValueOf(val)
			if fieldVal.Kind() == reflect.Slice || fieldVal.Kind() == reflect.Array {
				currPropertyArr = fieldVal.Interface().([]*construct.Resource)
			} else {
				if !fieldVal.IsNil() {
					currPropertyVal = fieldVal.Interface().(*construct.Resource)
				}
			}
			ctx.clearProperty(&tt.step, tt.resource, tt.propertyName)

			if currPropertyArr == nil && currPropertyVal == nil {
				assert.Fail(t, "property is nil")
			}
			if currPropertyVal != nil {
				if tt.step.Direction == knowledgebase.Upstream {
					g.AssertCalled(t, "RemoveDependency", currPropertyVal.ID, originalId)
				} else {
					g.AssertCalled(t, "RemoveDependency", originalId, currPropertyVal.ID)
				}
			} else {
				for _, res := range currPropertyArr {
					if tt.step.Direction == knowledgebase.Upstream {
						g.AssertCalled(t, "RemoveDependency", res.ID, originalId)
					} else {
						g.AssertCalled(t, "RemoveDependency", originalId, res.ID)
					}
				}
			}
		})
	}
}

func Test_addDependencyForDirection(t *testing.T) {
	tests := []struct {
		name string
		to   *construct.Resource
		from *construct.Resource
		step knowledgebase.OperationalStep
	}{
		{
			name: "downstream",
			to:   kbtesting.MockResource1("test1"),
			from: kbtesting.MockResource4(""),
			step: knowledgebase.OperationalStep{
				Direction: knowledgebase.Downstream,
			},
		},
		{
			name: "upstream",
			to:   kbtesting.MockResource1("test1"),
			from: kbtesting.MockResource4(""),
			step: knowledgebase.OperationalStep{
				Direction: knowledgebase.Upstream,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := &enginetesting.MockGraph{}
			ctx := OperationalRuleContext{
				Graph: g,
			}

			g.On("AddDependency", mock.Anything, mock.Anything).Return(nil)

			ctx.addDependencyForDirection(&tt.step, tt.to, tt.from)

			if tt.step.Direction == knowledgebase.Upstream {
				g.AssertCalled(t, "AddDependency", tt.from, tt.to)
			} else {
				g.AssertCalled(t, "AddDependency", tt.to, tt.from)
			}
		})
	}
}

func Test_removeDependencyForDirection(t *testing.T) {
	tests := []struct {
		name      string
		to        *construct.Resource
		from      *construct.Resource
		direction knowledgebase.Direction
	}{
		{
			name:      "downstream",
			to:        kbtesting.MockResource1("test1"),
			from:      kbtesting.MockResource4(""),
			direction: knowledgebase.Downstream,
		},
		{
			name:      "upstream",
			to:        kbtesting.MockResource1("test1"),
			from:      kbtesting.MockResource4(""),
			direction: knowledgebase.Upstream,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := &enginetesting.MockGraph{}
			ctx := OperationalRuleContext{
				Graph: g,
			}

			g.On("RemoveDependency", mock.Anything, mock.Anything).Return(nil)

			ctx.removeDependencyForDirection(tt.direction, tt.to, tt.from)

			if tt.direction == knowledgebase.Upstream {
				g.AssertCalled(t, "RemoveDependency", tt.from.ID, tt.to.ID)
			} else {
				g.AssertCalled(t, "RemoveDependency", tt.to.ID, tt.from.ID)
			}
		})
	}
}

func Test_generateResourceName(t *testing.T) {
	tests := []struct {
		name          string
		resource      *construct.Resource
		resourceToSet *construct.Resource
		unique        bool
		want          string
	}{

		{
			name:          "resource name is not unique",
			resource:      kbtesting.MockResource1("test1"),
			resourceToSet: kbtesting.MockResource4(""),
			unique:        false,
			want:          "resource4-0",
		},
		{
			name:          "resource name is unique",
			resource:      kbtesting.MockResource1("test1"),
			resourceToSet: kbtesting.MockResource4(""),
			unique:        true,
			want:          "resource4-test1-0",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			g := &enginetesting.MockGraph{}
			ctx := OperationalRuleContext{
				Graph: g,
			}

			g.On("ListResources").Return([]*construct.Resource{tt.resource, tt.resource}, nil)

			ctx.generateResourceName(tt.resourceToSet, tt.resource, tt.unique)

			g.AssertCalled(t, "ListResources")
			assert.Equal(tt.want, tt.resourceToSet.ID.Name)
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
		step          knowledgebase.OperationalStep
		shouldReplace bool
		want          *construct.Resource
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
			step:          knowledgebase.OperationalStep{Direction: knowledgebase.Downstream},
			shouldReplace: true,
			want: &construct.Resource{
				ID: construct.ResourceId{Provider: "mock", Type: "resource1", Namespace: res4.ID.Name, Name: "test1"},
				Properties: map[string]interface{}{
					"Res4": res4.ID,
				},
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
			step:          knowledgebase.OperationalStep{Direction: knowledgebase.Upstream},
			shouldReplace: true,
			want: &construct.Resource{
				ID: construct.ResourceId{Provider: "mock", Type: "resource1", Namespace: res4.ID.Name, Name: "test1"},
				Properties: map[string]interface{}{
					"Res4": res4.ID,
				},
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
			step:          knowledgebase.OperationalStep{Direction: knowledgebase.Upstream},
			shouldReplace: true,
			want: &construct.Resource{
				ID: construct.ResourceId{Provider: "mock", Type: "resource1", Namespace: res4.ID.Name, Name: "test1"},
				Properties: map[string]interface{}{
					"Res4": kbtesting.MockResource4("test4").ID,
				},
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
			resourceToSet: res4,
			property:      &knowledgebase.Property{Name: "Res4", Path: "Res4"},
			step:          knowledgebase.OperationalStep{Direction: knowledgebase.Upstream},
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
			step:          knowledgebase.OperationalStep{Direction: knowledgebase.Upstream},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			g := &enginetesting.MockGraph{}
			testKb := kbtesting.TestKB
			ctx := OperationalRuleContext{
				Property: tt.property,
				KB:       testKb,
				Graph:    g,
			}

			resId := tt.resource.ID
			var currPropertyVal construct.ResourceId
			var currPropertyArr []construct.ResourceId
			if tt.property != nil {
				val, err := tt.resource.GetProperty(tt.property.Name)
				if err != nil {
					t.Fatal(err)
				}
				fieldVal := reflect.ValueOf(val)
				if fieldVal.Kind() == reflect.Slice || fieldVal.Kind() == reflect.Array {
					currPropertyArr = fieldVal.Interface().([]construct.ResourceId)
				} else {
					if fieldVal.IsValid() && !fieldVal.IsZero() {
						currPropertyVal = fieldVal.Interface().(construct.ResourceId)
					}
				}
			}
			if tt.shouldReplace {
				g.On("GetResource", mock.Anything).Return(res4ToReplace, nil)
				g.On("RemoveDependency", mock.Anything, mock.Anything).Return(nil)
				g.On("RemoveResource", mock.Anything, mock.Anything).Return(nil)
				g.On("ReplaceResourceId", mock.Anything, mock.Anything).Return(nil)
			}

			err := ctx.setField(tt.resource, tt.resourceToSet, &tt.step)
			if !assert.NoError(err) {
				return
			}

			if tt.property != nil {
				propertyVal, err := tt.resource.GetProperty(tt.property.Name)
				if err != nil {
					t.Fatal(err)
				}
				if reflect.ValueOf(propertyVal).Kind() == reflect.Slice || reflect.ValueOf(propertyVal).Kind() == reflect.Array {
					assert.Equal(propertyVal.([]construct.ResourceId), append(currPropertyArr, tt.resourceToSet.ID))
				} else {
					assert.Equal(propertyVal, tt.resourceToSet.ID)
				}
			}

			if tt.shouldReplace {
				if !currPropertyVal.IsZero() {
					if tt.step.Direction == knowledgebase.Upstream {
						g.AssertCalled(t, "RemoveDependency", currPropertyVal, resId)
					} else {
						g.AssertCalled(t, "RemoveDependency", resId, currPropertyVal)
					}
					g.AssertCalled(t, "GetResource", currPropertyVal)
					g.AssertCalled(t, "RemoveResource", res4ToReplace, false)
				} else {
					g.AssertNotCalled(t, "RemoveDependency", mock.Anything, mock.Anything)
					g.AssertNotCalled(t, "RemoveResource", mock.Anything, mock.Anything)
				}
				g.AssertCalled(t, "ReplaceResourceId", resId, tt.want)
			} else {
				g.AssertNotCalled(t, "RemoveDependency", mock.Anything, mock.Anything)
				g.AssertNotCalled(t, "RemoveResource", mock.Anything, mock.Anything)
				g.AssertNotCalled(t, "ReplaceResourceId", mock.Anything, mock.Anything)
			}
		})
	}
}
