package core

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

	IacValue       IaCValue
	IacValuePtr    *IaCValue
	IacValueArr    []IaCValue
	IacValuePtrArr []*IaCValue
	IacValueMap    map[string]IaCValue
	IacValuePtrMap map[string]*IaCValue
}

// KlothoConstructRef returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (tr *testResource) KlothoConstructRef() []AnnotationKey {
	return nil
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

		IacValue:    IaCValue{Resource: &testResource{Name: "value1"}},
		IacValuePtr: &IaCValue{Resource: &testResource{Name: "value2"}},
		IacValueArr: []IaCValue{
			{Resource: &testResource{Name: "value_arr1"}},
			{Resource: &testResource{Name: "value_arr2"}},
		},
		IacValuePtrArr: []*IaCValue{
			{Resource: &testResource{Name: "value_ptr_arr1"}},
			{Resource: &testResource{Name: "value_ptr_arr2"}},
		},
		IacValueMap: map[string]IaCValue{
			"one": {Resource: &testResource{Name: "value_map1"}},
			"two": {Resource: &testResource{Name: "value_map2"}},
		},
		IacValuePtrMap: map[string]*IaCValue{
			"one": {Resource: &testResource{Name: "value_ptr_map1"}},
			"two": {Resource: &testResource{Name: "value_ptr_map2"}},
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
		"value1", "value2",
		"value_arr1", "value_arr2",
		"value_ptr_arr1", "value_ptr_arr2",
		"value_map1", "value_map2",
		"value_ptr_map1", "value_ptr_map2",
		"nested",
		"nested_arr1", "nested_arr2",
		"nested_map1", "nested_map2",
	} {
		assert.NotNil(dag.GetDependencyByVertexIds(tr.Id().String(), (&testResource{Name: target}).Id().String()), "source -> %s", target)
	}
	assert.Nil(dag.GetDependencyByVertexIds(tr.Id().String(), (&testResource{Name: "nested_single"}).Id().String()))
}

func TestResourceGraph_CreateDependencies(t *testing.T) {
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

		SingleDependency: &testResource{Name: "single", NestedResources: NestedResources{Resource: &testResource{Name: "nested_single"}}},

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

		IacValue:    IaCValue{Resource: &testResource{Name: "value1"}},
		IacValuePtr: &IaCValue{Resource: &testResource{Name: "value2"}},
		IacValueArr: []IaCValue{
			{Resource: &testResource{Name: "value_arr1"}},
			{Resource: &testResource{Name: "value_arr2"}},
		},
		IacValuePtrArr: []*IaCValue{
			{Resource: &testResource{Name: "value_ptr_arr1"}},
			{Resource: &testResource{Name: "value_ptr_arr2"}},
		},
		IacValueMap: map[string]IaCValue{
			"one": {Resource: &testResource{Name: "value_map1"}},
			"two": {Resource: &testResource{Name: "value_map2"}},
		},
		IacValuePtrMap: map[string]*IaCValue{
			"one": {Resource: &testResource{Name: "value_ptr_map1"}},
			"two": {Resource: &testResource{Name: "value_ptr_map2"}},
		},
	}

	subParams := map[string]any{
		"NestedResources": map[string]any{
			"Resource":           testResourceParams{Name: "nested"},
			"SpecificDependency": testResourceParams{Name: "nested_specific"},
			"ResourceArray":      []testResourceParams{{Name: "nested_arr1"}, {Name: "nested_arr2"}},
			"ResourceMap": map[string]testResourceParams{
				"one": {Name: "nested_map1"},
				"two": {Name: "nested_map2"},
			},
		},

		"SingleDependency":   testResourceParams{Name: "single"},
		"SpecificDependency": testResourceParams{Name: "specific"},
		"DependencyArray":    []testResourceParams{{Name: "arr1"}, {Name: "arr2"}},
		"SpecificDepArray":   []testResourceParams{{Name: "spec_arr1"}, {Name: "spec_arr2"}},

		"DependencyMap": map[string]testResourceParams{
			"one": {Name: "map1"},
			"two": {Name: "map2"},
		},
		"SpecificDepMap": map[string]testResourceParams{
			"one": {Name: "spec_map1"},
			"two": {Name: "spec_map2"},
		},

		"IacValue":    testResourceParams{Name: "value1"},
		"IacValuePtr": testResourceParams{Name: "value2"},
		"IacValueArr": []testResourceParams{
			{Name: "value_arr1"},
			{Name: "value_arr2"},
		},
		"IacValuePtrArr": []testResourceParams{
			{Name: "value_ptr_arr1"},
			{Name: "value_ptr_arr2"},
		},
		"IacValueMap": map[string]testResourceParams{
			"one": {Name: "value_map1"},
			"two": {Name: "value_map2"},
		},
		"IacValuePtrMap": map[string]testResourceParams{
			"one": {Name: "value_ptr_map1"},
			"two": {Name: "value_ptr_map2"},
		},
	}

	dag := NewResourceGraph()

	err := dag.CreateDependencies(tr, subParams)

	assert := assert.New(t)

	if !assert.NoError(err) {
		return
	}
	assert.Len(dag.ListResources(), 27)
	assert.Len(dag.ListDependencies(), 26)
	assert.NotNil(tr.SpecificDependency)
	assert.NotNil(tr.NestedResources.SpecificDependency)
	for _, target := range []string{
		"single", "single_specific",
		"arr1", "arr2",
		"spec_arr1", "spec_arr2",
		"map1", "map2",
		"spec_map1", "spec_map2",
		"value1", "value2",
		"value_arr1", "value_arr2",
		"value_ptr_arr1", "value_ptr_arr2",
		"value_map1", "value_map2",
		"value_ptr_map1", "value_ptr_map2",
		"nested",
		"nested_arr1", "nested_arr2",
		"nested_map1", "nested_map2",
	} {
		assert.Nil(dag.GetDependencyByVertexIds(tr.Id().String(), (&testResource{Name: target}).Id().String()), "source -> %s", target)
	}
}
