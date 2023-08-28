package types

import (
	"github.com/klothoplatform/klotho/pkg/annotation"
	"github.com/klothoplatform/klotho/pkg/construct"
)

type (
	InternalResource struct {
		Name string
	}
)

const KlothoPayloadName = "InternalKlothoPayloads"
const INTERNAL_TYPE = "internal"

func (p *InternalResource) Id() construct.ResourceId {
	return construct.ResourceId{
		Provider: construct.AbstractConstructProvider,
		Type:     INTERNAL_TYPE,
		Name:     p.Name,
	}
}

func (p *InternalResource) AnnotationCapability() string {
	return annotation.InternalCapability
}

func (p *InternalResource) Functionality() construct.Functionality {
	return construct.Storage
}

func (p *InternalResource) Attributes() map[string]any {
	return map[string]any{
		"blob": nil,
	}
}
