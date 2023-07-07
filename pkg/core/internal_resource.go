package core

import (
	"github.com/klothoplatform/klotho/pkg/annotation"
)

type (
	InternalResource struct {
		Name string
	}
)

const KlothoPayloadName = "InternalKlothoPayloads"
const INTERNAL_TYPE = "internal"

func (p *InternalResource) Id() ResourceId {
	return ResourceId{
		Provider: AbstractConstructProvider,
		Type:     INTERNAL_TYPE,
		Name:     p.Name,
	}
}

func (p *InternalResource) AnnotationCapability() string {
	return annotation.InternalCapability
}

func (p *InternalResource) Functionality() Functionality {
	return Storage
}

func (p *InternalResource) Attributes() map[string]any {
	return map[string]any{
		"blob": nil,
	}
}
