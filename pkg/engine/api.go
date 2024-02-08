package engine

import (
	"github.com/klothoplatform/klotho/pkg/collectionutil"
	construct "github.com/klothoplatform/klotho/pkg/construct2"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base2"
)

func (e *Engine) ListResources() []construct.ResourceId {
	resourceTemplates := e.Kb.ListResources()
	resources := []construct.ResourceId{}
	for _, res := range resourceTemplates {
		resources = append(resources, res.Id())
	}
	return resources
}

func (e *Engine) ListProviders() []string {
	resourceTemplates := e.Kb.ListResources()
	providers := []string{}
	for _, res := range resourceTemplates {
		provider := res.Id().Provider
		if !collectionutil.Contains(providers, provider) {
			providers = append(providers, provider)
		}
	}
	return providers
}

func (e *Engine) ListFunctionalities() []knowledgebase.Functionality {
	functionalities := []knowledgebase.Functionality{}
	resourceTemplates := e.Kb.ListResources()
	for _, res := range resourceTemplates {
		functionality := res.GetFunctionality()
		if !collectionutil.Contains(functionalities, functionality) {
			functionalities = append(functionalities, functionality)
		}

	}
	return functionalities
}

func (e *Engine) ListAttributes() []string {
	attributes := []string{}
	resourceTemplates := e.Kb.ListResources()
	for _, res := range resourceTemplates {
		for _, is := range res.Classification.Is {
			if !collectionutil.Contains(attributes, is) {
				attributes = append(attributes, is)
			}
		}
		for _, gives := range res.Classification.Gives {
			if !collectionutil.Contains(attributes, gives.Attribute) {
				attributes = append(attributes, gives.Attribute)
			}
		}
	}
	return attributes
}
