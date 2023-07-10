package engine

import "github.com/klothoplatform/klotho/pkg/core"

func (e *Engine) ListResources() []core.Resource {
	resources := []core.Resource{}
	for _, provider := range e.Providers {
		resources = append(resources, provider.ListResources()...)
	}
	return resources
}

func (e *Engine) ListResourcesByType() []string {
	resources := []string{}
	for _, res := range e.ListResources() {
		resources = append(resources, res.Id().String())
	}
	return resources
}

func (e *Engine) ListProviders() []string {
	providers := []string{core.AbstractConstructProvider}
	for _, provider := range e.Providers {
		providers = append(providers, provider.Name())
	}
	return providers
}

func (e *Engine) ListAttributes() []string {
	attributesMap := map[string]bool{}
	for _, classification := range e.ClassificationDocument.Classifications {
		for _, gives := range classification.Gives {
			attributesMap[gives.Attribute] = true
		}
		for _, is := range classification.Is {
			attributesMap[is] = true
		}
	}
	attributes := []string{}
	for attribute := range attributesMap {
		attributes = append(attributes, attribute)
	}
	return attributes
}
