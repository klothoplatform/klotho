package engine

import "github.com/klothoplatform/klotho/pkg/core"

func (e *Engine) ListResources() []core.Resource {
	resources := []core.Resource{}
	for _, provider := range e.Providers {
		resources = append(resources, provider.ListResources()...)
	}
	return resources
}
