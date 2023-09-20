package kbtesting

import (
	"github.com/klothoplatform/klotho/pkg/construct"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base2"
)

type (
	MockResource1 struct {
		Name          string
		ConstructRefs construct.BaseConstructSet `yaml:"-"`
		Res4          *MockResource4
		Res2s         []construct.Resource
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
)

func (f *MockResource1) Id() construct.ResourceId {
	if f.Res4 == nil {
		return construct.ResourceId{Provider: "mock", Type: "resource1", Name: f.Name}
	}
	return construct.ResourceId{Provider: "mock", Type: "resource1", Namespace: f.Res4.Name, Name: f.Name}
}
func (f *MockResource1) BaseConstructRefs() construct.BaseConstructSet { return f.ConstructRefs }
func (f *MockResource1) DeleteContext() construct.DeleteContext {
	return construct.DeleteContext{
		RequiresNoUpstream:   true,
		RequiresNoDownstream: true,
	}
}
func (f *MockResource2) Id() construct.ResourceId {
	return construct.ResourceId{Provider: "mock", Type: "resource2", Name: f.Name}
}
func (f *MockResource2) BaseConstructRefs() construct.BaseConstructSet { return f.ConstructRefs }
func (f *MockResource2) DeleteContext() construct.DeleteContext {
	return construct.DeleteContext{
		RequiresNoUpstream:   true,
		RequiresNoDownstream: true,
	}
}
func (f *MockResource3) Id() construct.ResourceId {
	return construct.ResourceId{Provider: "mock", Type: "resource3", Name: f.Name}
}
func (f *MockResource3) BaseConstructRefs() construct.BaseConstructSet { return f.ConstructRefs }
func (f *MockResource3) DeleteContext() construct.DeleteContext {
	return construct.DeleteContext{
		RequiresNoUpstream:   true,
		RequiresNoDownstream: true,
	}
}

func (f *MockResource4) Id() construct.ResourceId {
	return construct.ResourceId{Provider: "mock", Type: "resource4", Name: f.Name}
}
func (f *MockResource4) BaseConstructRefs() construct.BaseConstructSet { return f.ConstructRefs }
func (f *MockResource4) DeleteContext() construct.DeleteContext {
	return construct.DeleteContext{
		RequiresNoUpstream:   true,
		RequiresNoDownstream: true,
	}
}

// Defined are a set of resource teampltes that are used for testing
var resource1 = &knowledgebase.ResourceTemplate{
	QualifiedTypeName: "mock:resource1",
	Properties: map[string]knowledgebase.Property{
		"Name": {
			Name:             "Name",
			Type:             "string",
			Namespace:        false,
			UserConfigurable: false,
		},
		"Res4": {
			Name:             "Res4",
			Type:             "mock:resource4",
			Namespace:        true,
			UserConfigurable: false,
		},
	},
	Classification: knowledgebase.Classification{
		Is: []string{},
	},
	DeleteContext: construct.DeleteContext{},
}

var resource2 = &knowledgebase.ResourceTemplate{
	QualifiedTypeName: "mock:resource2",
	Properties: map[string]knowledgebase.Property{
		"Name": {
			Name:             "Name",
			Type:             "string",
			Namespace:        false,
			UserConfigurable: false,
		},
	},
	Classification: knowledgebase.Classification{
		Is: []string{},
	},
	DeleteContext: construct.DeleteContext{},
}

var resource3 = &knowledgebase.ResourceTemplate{
	QualifiedTypeName: "mock:resource3",
	Properties: map[string]knowledgebase.Property{
		"Name": {
			Name:             "Name",
			Type:             "string",
			Namespace:        false,
			UserConfigurable: false,
		},
	},
	Classification: knowledgebase.Classification{
		Is: []string{},
	},
	DeleteContext: construct.DeleteContext{},
}

var resource4 = &knowledgebase.ResourceTemplate{
	QualifiedTypeName: "mock:resource4",
	Properties: map[string]knowledgebase.Property{
		"Name": {
			Name:             "Name",
			Type:             "string",
			Namespace:        false,
			UserConfigurable: false,
		},
	},
	Classification: knowledgebase.Classification{
		Is: []string{"role"},
	},
	DeleteContext: construct.DeleteContext{},
}
