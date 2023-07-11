package engine

import (
	"fmt"
	"github.com/klothoplatform/klotho/pkg/core"
)

func (e *Engine) ListResources() []core.Resource {
	var resources []core.Resource
	for _, provider := range e.Providers {
		resources = append(resources, provider.ListResources()...)
	}
	return resources
}

func (e *Engine) ListResourcesByType() []string {
	var resources []string
	for _, res := range e.ListResources() {
		id := res.Id()
		resources = append(resources, fmt.Sprintf("%s:%s", id.Provider, id.Type))
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
	var attributes []string
	for attribute := range attributesMap {
		attributes = append(attributes, attribute)
	}
	return attributes
}
