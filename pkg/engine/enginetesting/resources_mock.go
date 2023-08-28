package enginetesting

import (
	"github.com/klothoplatform/klotho/pkg/construct"
)

type (
	MockResource1 struct {
		Name          string
		ConstructRefs construct.BaseConstructSet `yaml:"-"`
	}
	MockResource2 struct {
		Name          string
		ConstructRefs construct.BaseConstructSet `yaml:"-"`
	}
	MockResource3 struct {
		Name          string
		ConstructRefs construct.BaseConstructSet `yaml:"-"`
	}
	MockResource4 struct {
		Name          string
		ConstructRefs construct.BaseConstructSet `yaml:"-"`
	}

	// this is solely used for operational testing at the moment
	MockResource5 struct {
		Name          string
		ConstructRefs construct.BaseConstructSet `yaml:"-"`
		Mock1         *MockResource1
		Mock2s        []*MockResource2
	}

	TestRes1 struct {
		Field1 int
		Field2 string
		Field3 bool
		Arr1   []string
	}

	// this is solely used for configuration testing at the moment
	MockResource6 struct {
		Name          string
		ConstructRefs construct.BaseConstructSet `yaml:"-"`
		Field1        int
		Field2        string
		Field3        bool
		Arr1          []string
		Arr2          []TestRes1
		Arr3          []*TestRes1
		Struct1       TestRes1
		Struct2       *TestRes1
		Map1          map[string]construct.IaCValue
	}
)

func (f *MockResource1) Id() construct.ResourceId {
	return construct.ResourceId{Provider: "mock", Type: "mock1", Name: f.Name}
}
func (f *MockResource1) BaseConstructRefs() construct.BaseConstructSet { return f.ConstructRefs }
func (f *MockResource1) DeleteContext() construct.DeleteContext {
	return construct.DeleteContext{
		RequiresNoUpstream:   true,
		RequiresNoDownstream: true,
	}
}
func (f *MockResource2) Id() construct.ResourceId {
	return construct.ResourceId{Provider: "mock", Type: "mock2", Name: f.Name}
}
func (f *MockResource2) BaseConstructRefs() construct.BaseConstructSet { return f.ConstructRefs }
func (f *MockResource2) DeleteContext() construct.DeleteContext {
	return construct.DeleteContext{
		RequiresNoUpstream:   true,
		RequiresNoDownstream: true,
	}
}
func (f *MockResource3) Id() construct.ResourceId {
	return construct.ResourceId{Provider: "mock", Type: "mock3", Name: f.Name}
}
func (f *MockResource3) BaseConstructRefs() construct.BaseConstructSet { return f.ConstructRefs }
func (f *MockResource3) DeleteContext() construct.DeleteContext {
	return construct.DeleteContext{
		RequiresNoUpstream:   true,
		RequiresNoDownstream: true,
	}
}

func (f *MockResource4) Id() construct.ResourceId {
	return construct.ResourceId{Provider: "mock", Type: "mock4", Name: f.Name}
}
func (f *MockResource4) BaseConstructRefs() construct.BaseConstructSet { return f.ConstructRefs }
func (f *MockResource4) DeleteContext() construct.DeleteContext {
	return construct.DeleteContext{
		RequiresNoUpstream:   true,
		RequiresNoDownstream: true,
	}
}

func (f *MockResource5) Id() construct.ResourceId {
	return construct.ResourceId{Provider: "mock", Type: "mock5", Name: f.Name}
}
func (f *MockResource5) BaseConstructRefs() construct.BaseConstructSet { return f.ConstructRefs }
func (f *MockResource5) DeleteContext() construct.DeleteContext {
	return construct.DeleteContext{
		RequiresNoUpstream:   true,
		RequiresNoDownstream: true,
	}
}
func (f *MockResource6) Id() construct.ResourceId {
	return construct.ResourceId{Provider: "mock", Type: "mock6", Name: f.Name}
}
func (f *MockResource6) BaseConstructRefs() construct.BaseConstructSet { return f.ConstructRefs }
func (f *MockResource6) DeleteContext() construct.DeleteContext {
	return construct.DeleteContext{
		RequiresNoUpstream:   true,
		RequiresNoDownstream: true,
	}
}
