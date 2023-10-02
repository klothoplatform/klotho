package kbtesting

import (
	construct "github.com/klothoplatform/klotho/pkg/construct2"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base2"
)

func MockResource1(name string) *construct.Resource {
	return &construct.Resource{
		ID: construct.ResourceId{
			Provider: "mock",
			Type:     "resource1",
			Name:     name,
		},
		Properties: map[string]interface{}{},
	}
}

func MockResource2(name string) *construct.Resource {
	return &construct.Resource{
		ID: construct.ResourceId{
			Provider: "mock",
			Type:     "resource2",
			Name:     name,
		},
		Properties: map[string]interface{}{},
	}
}

func MockResource3(name string) *construct.Resource {
	return &construct.Resource{
		ID: construct.ResourceId{
			Provider: "mock",
			Type:     "resource3",
			Name:     name,
		},
		Properties: map[string]interface{}{},
	}
}

func MockResource4(name string) *construct.Resource {
	return &construct.Resource{
		ID: construct.ResourceId{
			Provider: "mock",
			Type:     "resource4",
			Name:     name,
		},
		Properties: map[string]interface{}{},
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
