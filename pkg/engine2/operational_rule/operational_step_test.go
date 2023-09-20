package operational_rule

import (
	"reflect"
	"testing"

	"github.com/klothoplatform/klotho/pkg/construct"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base2"
	kbtesting "github.com/klothoplatform/klotho/pkg/knowledge_base2/kb_testing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func Test_handleOperationalResourceAction(t *testing.T) {
	tests := []struct {
		name     string
		mocks    []mock.Call
		resource construct.Resource
		action   OperationalResourceAction
	}{
		{
			name:     "upstream exact resource that exists in the graph",
			resource: &kbtesting.MockResource1{Name: "test1"},
			action: OperationalResourceAction{
				Step: knowledgebase.OperationalStep{
					Direction: knowledgebase.Downstream,
					Resources: []string{"mock:resource4:test"},
					NumNeeded: 1,
				},
			},
			mocks: []mock.Call{
				{
					Method:          "GetResource",
					Arguments:       []interface{}{construct.ResourceId{Provider: "mock", Type: "resource4", Name: "test"}},
					ReturnArguments: mock.Arguments{&kbtesting.MockResource4{Name: "test"}},
				},
				{
					Method:          "AddDependency",
					Arguments:       []interface{}{&kbtesting.MockResource1{Name: "test1"}, &kbtesting.MockResource4{Name: "test"}},
					ReturnArguments: mock.Arguments{nil},
				},
			},
		},
		{
			name:     "upstream unique resources from types",
			resource: &kbtesting.MockResource1{Name: "test1"},
			action: OperationalResourceAction{
				Step: knowledgebase.OperationalStep{
					Direction: knowledgebase.Downstream,
					Resources: []string{"mock:resource4"},
					NumNeeded: 2,
					Unique:    true,
				},
			},
			mocks: []mock.Call{
				{
					Method:          "AddDependency",
					Arguments:       []interface{}{&kbtesting.MockResource1{Name: "test1"}, &kbtesting.MockResource4{Name: "resource4-test1-2"}},
					ReturnArguments: mock.Arguments{nil},
				},
				{
					Method:          "AddDependency",
					Arguments:       []interface{}{&kbtesting.MockResource1{Name: "test1"}, &kbtesting.MockResource4{Name: "resource4-test1-3"}},
					ReturnArguments: mock.Arguments{nil},
				},
				{
					Method: "ListResources",
					ReturnArguments: mock.Arguments{
						[]construct.Resource{
							&kbtesting.MockResource4{Name: "test"},
							&kbtesting.MockResource4{Name: "test2"},
						},
					},
				},
				{
					Method: "ListResources",
					ReturnArguments: mock.Arguments{
						[]construct.Resource{
							&kbtesting.MockResource4{Name: "test"},
							&kbtesting.MockResource4{Name: "test2"},
							&kbtesting.MockResource4{Name: "resource4-test1-2"},
						},
					},
				},
				{
					Method: "ListResources",
					ReturnArguments: mock.Arguments{
						[]construct.Resource{
							&kbtesting.MockResource4{Name: "test"},
							&kbtesting.MockResource4{Name: "test2"},
							&kbtesting.MockResource4{Name: "resource4-test1-2"},
						},
					},
				},
			},
		},
		{
			name:     "upstream resources from types, choose from available",
			resource: &kbtesting.MockResource1{Name: "test1"},
			action: OperationalResourceAction{
				Step: knowledgebase.OperationalStep{
					Direction: knowledgebase.Downstream,
					Resources: []string{"mock:resource4"},
					NumNeeded: 2,
				},
			},
			mocks: []mock.Call{
				{
					Method:          "AddDependency",
					Arguments:       []interface{}{&kbtesting.MockResource1{Name: "test1"}, &kbtesting.MockResource4{Name: "test"}},
					ReturnArguments: mock.Arguments{nil},
				},
				{
					Method:          "AddDependency",
					Arguments:       []interface{}{&kbtesting.MockResource1{Name: "test1"}, &kbtesting.MockResource4{Name: "test2"}},
					ReturnArguments: mock.Arguments{nil},
				},
				{
					Method: "ListResources",
					ReturnArguments: mock.Arguments{
						[]construct.Resource{
							&kbtesting.MockResource4{Name: "test"},
							&kbtesting.MockResource4{Name: "test2"},
						},
					},
				},
			},
		},
		{
			name:     "upstream resources from types, none available, will create",
			resource: &kbtesting.MockResource1{Name: "test1"},
			action: OperationalResourceAction{
				Step: knowledgebase.OperationalStep{
					Direction: knowledgebase.Downstream,
					Resources: []string{"mock:resource4"},
					NumNeeded: 2,
				},
			},
			mocks: []mock.Call{
				{
					Method:          "AddDependency",
					Arguments:       []interface{}{&kbtesting.MockResource1{Name: "test1"}, &kbtesting.MockResource4{Name: "resource4-0"}},
					ReturnArguments: mock.Arguments{nil},
				},
				{
					Method:          "AddDependency",
					Arguments:       []interface{}{&kbtesting.MockResource1{Name: "test1"}, &kbtesting.MockResource4{Name: "resource4-1"}},
					ReturnArguments: mock.Arguments{nil},
				},
				{
					Method: "ListResources",
					ReturnArguments: mock.Arguments{
						[]construct.Resource{},
					},
				},
				{
					Method: "ListResources",
					ReturnArguments: mock.Arguments{
						[]construct.Resource{},
					},
				},
				{
					Method: "ListResources",
					ReturnArguments: mock.Arguments{
						[]construct.Resource{
							&kbtesting.MockResource4{Name: "resource4-0"},
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			g := &MockGraph{}
			ctx := OperationalRuleContext{
				ConfigCtx: knowledgebase.ConfigTemplateContext{},
				Graph:     g,
				KB:        kbtesting.TestKB,
				CreateResourcefromId: func(id construct.ResourceId) construct.Resource {
					return &kbtesting.MockResource4{Name: id.Name}
				},
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
		resource construct.Resource
		want     []construct.ResourceId
	}{
		{
			name:     "upstream",
			resource: &kbtesting.MockResource1{Name: "test1"},
			step: knowledgebase.OperationalStep{
				Direction:       knowledgebase.Upstream,
				Classifications: []string{"role"},
			},
			want: []construct.ResourceId{},
		},
		{
			name:     "downstream",
			resource: &kbtesting.MockResource1{Name: "test1"},
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
			g := &MockGraph{}
			ctx := OperationalRuleContext{
				Graph: g,
				KB:    kbtesting.TestKB,
			}

			result := ctx.findResourcesWhichSatisfyStepClassifications(tt.step, tt.resource)
			assert.ElementsMatch(tt.want, result)
		})
	}
}

func Test_getResourcesForStep(t *testing.T) {
	tests := []struct {
		name     string
		step     knowledgebase.OperationalStep
		resource construct.Resource
		want     []construct.ResourceId
	}{
		{
			name:     "upstream resource types",
			resource: &kbtesting.MockResource1{Name: "test1"},
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
			resource: &kbtesting.MockResource1{Name: "test1"},
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
			resource: &kbtesting.MockResource1{Name: "test1"},
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
			resource: &kbtesting.MockResource1{Name: "test1"},
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
			g := &MockGraph{}
			ctx := OperationalRuleContext{
				Graph: g,
				KB:    kbtesting.TestKB,
			}

			g.On("GetFunctionalUpstreamResources", mock.Anything).Return(
				[]construct.Resource{
					&kbtesting.MockResource4{Name: "test"},
					&kbtesting.MockResource4{Name: "test2"},
					&kbtesting.MockResource3{Name: "test3"},
				},
			)
			g.On("GetFunctionalDownstreamResources", mock.Anything).Return(
				[]construct.Resource{
					&kbtesting.MockResource4{Name: "test"},
					&kbtesting.MockResource4{Name: "test2"},
					&kbtesting.MockResource3{Name: "test3"},
				},
			)

			result, err := ctx.getResourcesForStep(tt.step, tt.resource)
			if err != nil {
				t.Fatal(err)
			}
			assert.ElementsMatch(tt.want, result)
			if tt.step.Direction == knowledgebase.Downstream {
				g.AssertCalled(t, "GetFunctionalDownstreamResources", tt.resource)
			} else {
				g.AssertCalled(t, "GetFunctionalUpstreamResources", tt.resource)
			}
		})
	}
}

func Test_addDependenciesFromProperty(t *testing.T) {
	tests := []struct {
		name         string
		step         knowledgebase.OperationalStep
		resource     construct.Resource
		propertyName string
	}{
		{
			name:         "downstream",
			resource:     &kbtesting.MockResource1{Name: "test1", Res4: &kbtesting.MockResource4{Name: "test"}},
			step:         knowledgebase.OperationalStep{Direction: knowledgebase.Downstream},
			propertyName: "Res4",
		},
		{
			name:         "upstream",
			resource:     &kbtesting.MockResource1{Name: "test1", Res4: &kbtesting.MockResource4{Name: "test"}},
			step:         knowledgebase.OperationalStep{Direction: knowledgebase.Upstream},
			propertyName: "Res4",
		},
		{
			name: "array",
			resource: &kbtesting.MockResource1{
				Name:  "test1",
				Res2s: []construct.Resource{&kbtesting.MockResource2{Name: "test"}, &kbtesting.MockResource2{Name: "test2"}},
			},
			step:         knowledgebase.OperationalStep{Direction: knowledgebase.Upstream},
			propertyName: "Res2s",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := &MockGraph{}
			ctx := OperationalRuleContext{
				Graph: g,
			}

			g.On("AddDependency", mock.Anything, mock.Anything).Return(nil)

			var currPropertyVal construct.Resource
			var currPropertyArr []construct.Resource
			fieldVal := reflect.ValueOf(tt.resource).Elem().FieldByName(tt.propertyName)
			if fieldVal.Kind() == reflect.Slice || fieldVal.Kind() == reflect.Array {
				currPropertyArr = fieldVal.Interface().([]construct.Resource)
			} else {
				if !fieldVal.IsNil() {
					currPropertyVal = fieldVal.Interface().(construct.Resource)
				}
			}

			ctx.addDependenciesFromProperty(tt.step, tt.resource, tt.propertyName)

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
		resource     construct.Resource
		propertyName string
	}{
		{
			name:         "downstream",
			resource:     &kbtesting.MockResource1{Name: "test1", Res4: &kbtesting.MockResource4{Name: "test"}},
			step:         knowledgebase.OperationalStep{Direction: knowledgebase.Downstream},
			propertyName: "Res4",
		},
		{
			name:         "upstream",
			resource:     &kbtesting.MockResource1{Name: "test1", Res4: &kbtesting.MockResource4{Name: "test"}},
			step:         knowledgebase.OperationalStep{Direction: knowledgebase.Upstream},
			propertyName: "Res4",
		},
		{
			name: "array",
			resource: &kbtesting.MockResource1{
				Name:  "test1",
				Res2s: []construct.Resource{&kbtesting.MockResource2{Name: "test"}, &kbtesting.MockResource2{Name: "test2"}},
			},
			step:         knowledgebase.OperationalStep{Direction: knowledgebase.Upstream},
			propertyName: "Res2s",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := &MockGraph{}
			ctx := OperationalRuleContext{
				Graph: g,
			}

			g.On("RemoveDependency", mock.Anything, mock.Anything).Return(nil)

			originalId := tt.resource.Id()
			var currPropertyVal construct.Resource
			var currPropertyArr []construct.Resource
			fieldVal := reflect.ValueOf(tt.resource).Elem().FieldByName(tt.propertyName)
			if fieldVal.Kind() == reflect.Slice || fieldVal.Kind() == reflect.Array {
				currPropertyArr = fieldVal.Interface().([]construct.Resource)
			} else {
				if !fieldVal.IsNil() {
					currPropertyVal = fieldVal.Interface().(construct.Resource)
				}
			}
			ctx.clearProperty(tt.step, tt.resource, tt.propertyName)

			if currPropertyArr == nil && currPropertyVal == nil {
				assert.Fail(t, "property is nil")
			}
			if currPropertyVal != nil {
				if tt.step.Direction == knowledgebase.Upstream {
					g.AssertCalled(t, "RemoveDependency", currPropertyVal.Id(), originalId)
				} else {
					g.AssertCalled(t, "RemoveDependency", originalId, currPropertyVal.Id())
				}
			} else {
				for _, res := range currPropertyArr {
					if tt.step.Direction == knowledgebase.Upstream {
						g.AssertCalled(t, "RemoveDependency", res.Id(), originalId)
					} else {
						g.AssertCalled(t, "RemoveDependency", originalId, res.Id())
					}
				}
			}
		})
	}
}

func Test_addDependencyForDirection(t *testing.T) {
	tests := []struct {
		name      string
		to        construct.Resource
		from      construct.Resource
		direction knowledgebase.Direction
	}{
		{
			name:      "downstream",
			to:        &kbtesting.MockResource1{Name: "test1"},
			from:      &kbtesting.MockResource4{},
			direction: knowledgebase.Downstream,
		},
		{
			name:      "upstream",
			to:        &kbtesting.MockResource1{Name: "test1"},
			from:      &kbtesting.MockResource4{},
			direction: knowledgebase.Downstream,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := &MockGraph{}
			ctx := OperationalRuleContext{
				Graph: g,
			}

			g.On("AddDependency", mock.Anything, mock.Anything).Return(nil)

			ctx.addDependencyForDirection(tt.direction, tt.to, tt.from)

			if tt.direction == knowledgebase.Upstream {
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
		to        construct.Resource
		from      construct.Resource
		direction knowledgebase.Direction
	}{
		{
			name:      "downstream",
			to:        &kbtesting.MockResource1{Name: "test1"},
			from:      &kbtesting.MockResource4{},
			direction: knowledgebase.Downstream,
		},
		{
			name:      "upstream",
			to:        &kbtesting.MockResource1{Name: "test1"},
			from:      &kbtesting.MockResource4{},
			direction: knowledgebase.Downstream,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := &MockGraph{}
			ctx := OperationalRuleContext{
				Graph: g,
			}

			g.On("RemoveDependency", mock.Anything, mock.Anything).Return(nil)

			ctx.removeDependencyForDirection(tt.direction, tt.to, tt.from)

			if tt.direction == knowledgebase.Upstream {
				g.AssertCalled(t, "RemoveDependency", tt.from.Id(), tt.to.Id())
			} else {
				g.AssertCalled(t, "RemoveDependency", tt.to.Id(), tt.from.Id())
			}
		})
	}
}

func Test_generateResourceName(t *testing.T) {
	tests := []struct {
		name          string
		resource      construct.Resource
		resourceToSet construct.Resource
		unique        bool
		want          string
	}{
		{
			name:          "resource name is not unique",
			resource:      &kbtesting.MockResource1{Name: "test1"},
			resourceToSet: &kbtesting.MockResource4{},
			unique:        false,
			want:          "resource4-0",
		},
		{
			name:          "resource name is unique",
			resource:      &kbtesting.MockResource1{Name: "test1"},
			resourceToSet: &kbtesting.MockResource4{},
			unique:        true,
			want:          "resource4-test1-0",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			g := &MockGraph{}
			ctx := OperationalRuleContext{
				Graph: g,
			}

			g.On("ListResources").Return([]construct.Resource{tt.resource, tt.resource})

			ctx.generateResourceName(tt.resourceToSet, tt.resource, tt.unique)

			g.AssertCalled(t, "ListResources")
			assert.Equal(tt.want, tt.resourceToSet.Id().Name)
		})
	}

}

func Test_setField(t *testing.T) {
	tests := []struct {
		name          string
		resource      construct.Resource
		resourceToSet construct.Resource
		property      *knowledgebase.Property
		step          knowledgebase.OperationalStep
		shouldReplace bool
		want          construct.Resource
	}{
		{
			name:          "field is being replaced",
			resource:      &kbtesting.MockResource1{Name: "test1", Res4: &kbtesting.MockResource4{Name: "thisWillBeReplaced"}},
			resourceToSet: &kbtesting.MockResource4{Name: "test4"},
			property:      &knowledgebase.Property{Name: "Res4"},
			step:          knowledgebase.OperationalStep{Direction: knowledgebase.Downstream},
			shouldReplace: true,
			want:          &kbtesting.MockResource1{Name: "test1", Res4: &kbtesting.MockResource4{Name: "test4"}},
		},
		{
			name:          "field is being replaced, remove upstream",
			resource:      &kbtesting.MockResource1{Name: "test1", Res4: &kbtesting.MockResource4{Name: "thisWillBeReplaced"}},
			resourceToSet: &kbtesting.MockResource4{Name: "test4"},
			property:      &knowledgebase.Property{Name: "Res4"},
			step:          knowledgebase.OperationalStep{Direction: knowledgebase.Upstream},
			shouldReplace: true,
			want:          &kbtesting.MockResource1{Name: "test1", Res4: &kbtesting.MockResource4{Name: "test4"}},
		},
		{
			name:          "set field on resource",
			resource:      &kbtesting.MockResource1{Name: "test1"},
			resourceToSet: &kbtesting.MockResource4{Name: "test4"},
			property:      &knowledgebase.Property{Name: "Res4"},
			step:          knowledgebase.OperationalStep{Direction: knowledgebase.Upstream},
			shouldReplace: true,
			want:          &kbtesting.MockResource1{Name: "test1", Res4: &kbtesting.MockResource4{Name: "test4"}},
		},
		{
			name:          "field is not being replaced",
			resource:      &kbtesting.MockResource1{Name: "test1", Res4: &kbtesting.MockResource4{Name: "test4"}},
			resourceToSet: &kbtesting.MockResource4{Name: "test4"},
			property:      &knowledgebase.Property{Name: "Res4"},
			step:          knowledgebase.OperationalStep{Direction: knowledgebase.Upstream},
		},
		{
			name:          "field is array",
			resource:      &kbtesting.MockResource1{Name: "test1", Res2s: []construct.Resource{&kbtesting.MockResource3{Name: "test"}}},
			resourceToSet: &kbtesting.MockResource2{Name: "test2"},
			property:      &knowledgebase.Property{Name: "Res2s"},
			step:          knowledgebase.OperationalStep{Direction: knowledgebase.Upstream},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			g := &MockGraph{}
			testKb := kbtesting.TestKB
			ctx := OperationalRuleContext{
				Property: tt.property,
				KB:       testKb,
				Graph:    g,
			}

			resId := tt.resource.Id()
			var currPropertyVal construct.Resource
			var currPropertyArr []construct.Resource
			if tt.property != nil {

				fieldVal := reflect.ValueOf(tt.resource).Elem().FieldByName(tt.property.Name)
				if fieldVal.Kind() == reflect.Slice || fieldVal.Kind() == reflect.Array {
					currPropertyArr = fieldVal.Interface().([]construct.Resource)
				} else {
					if !fieldVal.IsNil() {
						currPropertyVal = fieldVal.Interface().(construct.Resource)
					}
				}
			}
			if tt.shouldReplace {
				g.On("RemoveDependency", mock.Anything, mock.Anything).Return(nil)
				g.On("RemoveResource", mock.Anything, mock.Anything).Return(nil)
				g.On("ReplaceResourceId", mock.Anything, mock.Anything).Return(nil)
			}

			err := ctx.setField(tt.resource, tt.resourceToSet, tt.step)
			if !assert.NoError(err) {
				return
			}

			if tt.property != nil {
				propertyVal := reflect.ValueOf(tt.resource).Elem().FieldByName(tt.property.Name).Interface()
				if reflect.ValueOf(tt.resource).Elem().FieldByName(tt.property.Name).Kind() == reflect.Slice || reflect.ValueOf(tt.resource).Elem().FieldByName(tt.property.Name).Kind() == reflect.Array {
					assert.Equal(propertyVal.([]construct.Resource), append(currPropertyArr, tt.resourceToSet))
				} else {
					assert.Equal(propertyVal, tt.resourceToSet)
				}
			}

			if tt.shouldReplace {
				if currPropertyVal != nil {
					if tt.step.Direction == knowledgebase.Upstream {
						g.AssertCalled(t, "RemoveDependency", currPropertyVal.Id(), resId)
					} else {
						g.AssertCalled(t, "RemoveDependency", resId, currPropertyVal.Id())
					}
					g.AssertCalled(t, "RemoveResource", currPropertyVal, false)
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

type MockGraph struct {
	mock.Mock
}

func (g *MockGraph) ListResources() []construct.Resource {
	args := g.Called()
	return args.Get(0).([]construct.Resource)
}
func (g *MockGraph) AddResource(resource construct.Resource) {
}
func (g *MockGraph) RemoveResource(resource construct.Resource, explicit bool) error {
	args := g.Called(resource, explicit)
	return args.Error(0)
}
func (g *MockGraph) AddDependency(from construct.Resource, to construct.Resource) error {
	args := g.Called(from, to)
	return args.Error(0)
}
func (g *MockGraph) RemoveDependency(from construct.ResourceId, to construct.ResourceId) error {
	args := g.Called(from, to)
	return args.Error(0)
}
func (g *MockGraph) GetResource(id construct.ResourceId) construct.Resource {
	args := g.Called(id)
	return args.Get(0).(construct.Resource)
}
func (g *MockGraph) GetFunctionalDownstreamResourcesOfType(resource construct.Resource, qualifiedType construct.ResourceId) []construct.Resource {
	args := g.Called(resource, qualifiedType)
	return args.Get(0).([]construct.Resource)
}
func (g *MockGraph) GetFunctionalDownstreamResources(resource construct.Resource) []construct.Resource {
	args := g.Called(resource)
	return args.Get(0).([]construct.Resource)
}
func (g *MockGraph) GetFunctionalUpstreamResourcesOfType(resource construct.Resource, qualifiedType construct.ResourceId) []construct.Resource {
	args := g.Called(resource, qualifiedType)
	return args.Get(0).([]construct.Resource)
}
func (g *MockGraph) GetFunctionalUpstreamResources(resource construct.Resource) []construct.Resource {
	args := g.Called(resource)
	return args.Get(0).([]construct.Resource)
}
func (g *MockGraph) ReplaceResourceId(oldId construct.ResourceId, resource construct.Resource) error {
	args := g.Called(oldId, resource)
	return args.Error(0)
}
func (g *MockGraph) ConfigureResource(resource construct.Resource, configuration knowledgebase.Configuration, data knowledgebase.ConfigTemplateData) error {
	args := g.Called(resource, configuration, data)
	return args.Error(0)
}
