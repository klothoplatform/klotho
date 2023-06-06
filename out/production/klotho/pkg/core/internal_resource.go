package core

type (
	InternalResource struct {
		AnnotationKey
	}
)

const KlothoPayloadName = "InternalKlothoPayloads"

func (p *InternalResource) Provenance() AnnotationKey {
	return p.AnnotationKey
}

func (p *InternalResource) Id() ResourceId {
	return ConstructId(p.AnnotationKey).ToRid()
}
