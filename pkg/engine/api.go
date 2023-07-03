package engine

import "github.com/klothoplatform/klotho/pkg/core"

func (e *Engine) ListResources() []core.Resource {
	resources := []core.Resource{}
	for _, provider := range e.Providers {
		for _, res := range provider.ListResources() {
			resources = append(resources, res)
		}
	}
	return resources
}
