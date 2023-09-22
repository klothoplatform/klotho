package kbtesting

import (
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base2"
)

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
	DeleteContext: knowledgebase.DeleteContext{},
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
	DeleteContext: knowledgebase.DeleteContext{},
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
	DeleteContext: knowledgebase.DeleteContext{},
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
	DeleteContext: knowledgebase.DeleteContext{},
}
