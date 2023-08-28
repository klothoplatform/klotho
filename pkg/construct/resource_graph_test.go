package construct

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

type NestedResources struct {
	Resource           Resource
	SpecificDependency *testResource
	ResourceArray      []Resource
	ResourceMap        map[string]Resource
}

type testResource struct {
	Name string

	NestedResources NestedResources

	SingleDependency   Resource
	SpecificDependency *testResource

	DependencyArray  []Resource
	SpecificDepArray []*testResource

	DependencyMap  map[string]Resource
	SpecificDepMap map[string]*testResource
}

// BaseConstructRefs returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (tr *testResource) BaseConstructRefs() BaseConstructSet {
	return nil
}

func (tr *testResource) DeleteContext() DeleteContext {
	return DeleteContext{}
}

type testResourceParams struct {
	Name string
}

func (tr *testResource) Create(dag *ResourceGraph, metadata testResourceParams) error {
	currName := metadata.Name + uuid.NewString()
	*tr = testResource{Name: currName}
	return nil
}

// Id returns the id of the cloud resource
func (tr *testResource) Id() ResourceId {
	return ResourceId{
		Provider: "test-provider",
		Type:     "test-type",
		Name:     tr.Name,
	}
}

func TestResourceGraph_AddDependenciesReflect(t *testing.T) {
	tr := &testResource{
		Name: "id",

		NestedResources: NestedResources{
			Resource:      &testResource{Name: "nested"},
			ResourceArray: []Resource{&testResource{Name: "nested_arr1"}, &testResource{Name: "nested_arr2"}},
			ResourceMap: map[string]Resource{
				"one": &testResource{Name: "nested_map1"},
				"two": &testResource{Name: "nested_map2"},
			},
		},

		SingleDependency:   &testResource{Name: "single", NestedResources: NestedResources{Resource: &testResource{Name: "nested_single"}}},
		SpecificDependency: &testResource{Name: "single_specific"},

		DependencyArray:  []Resource{&testResource{Name: "arr1"}, &testResource{Name: "arr2"}},
		SpecificDepArray: []*testResource{{Name: "spec_arr1"}, {Name: "spec_arr2"}},

		DependencyMap: map[string]Resource{
			"one": &testResource{Name: "map1"},
			"two": &testResource{Name: "map2"},
		},
		SpecificDepMap: map[string]*testResource{
			"one": {Name: "spec_map1"},
			"two": {Name: "spec_map2"},
		},
	}

	dag := NewResourceGraph()

	dag.AddDependenciesReflect(tr)

	assert := assert.New(t)

	for _, target := range []string{
		"single", "single_specific",
		"arr1", "arr2",
		"spec_arr1", "spec_arr2",
		"map1", "map2",
		"spec_map1", "spec_map2",
		"nested",
		"nested_arr1", "nested_arr2",
		"nested_map1", "nested_map2",
	} {
		assert.NotNil(dag.GetDependency(tr.Id(), (&testResource{Name: target}).Id()), "source -> %s", target)
	}
	assert.Nil(dag.GetDependency(tr.Id(), (&testResource{Name: "nested_single"}).Id()))
}
