package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type testResource struct {
	ID string

	SingleDependency   Resource
	SpecificDependency *testResource

	DependencyArray  []Resource
	SpecificDepArray []*testResource

	DependencyMap  map[string]Resource
	SpecificDepMap map[string]*testResource
}

// Provider returns name of the provider the resource is correlated to
func (tr *testResource) Provider() string {
	return "test"
}

// KlothoResource returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (tr *testResource) KlothoConstructRef() []AnnotationKey {
	return nil
}

// ID returns the id of the cloud resource
func (tr *testResource) Id() string {
	return tr.ID
}

func TestResourceGraph_AddDependenciesReflect(t *testing.T) {
	tr := &testResource{
		ID: "source",

		SingleDependency:   &testResource{ID: "single"},
		SpecificDependency: &testResource{ID: "single_specific"},

		DependencyArray:  []Resource{&testResource{ID: "arr1"}, &testResource{ID: "arr2"}},
		SpecificDepArray: []*testResource{{ID: "spec_arr1"}, {ID: "spec_arr2"}},

		DependencyMap: map[string]Resource{
			"one": &testResource{ID: "map1"},
			"two": &testResource{ID: "map2"},
		},
		SpecificDepMap: map[string]*testResource{
			"one": {ID: "spec_map1"},
			"two": {ID: "spec_map2"},
		},
	}

	dag := NewResourceGraph()

	dag.AddDependenciesReflect(tr)

	assert := assert.New(t)

	for _, target := range []string{"single", "single_specific", "arr1", "arr2", "spec_arr1", "spec_arr2", "map1", "map2", "spec_map1", "spec_map2"} {
		assert.NotNil(dag.GetDependency(tr.ID, target), "source -> %s", target)
	}
}
