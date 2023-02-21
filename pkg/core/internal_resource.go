package core

type (
	InternalResource struct {
		Name string
	}
)

const KlothoPayloadName = "InternalKlothoPayloads"

const InternalKind = "internal"

func (p *InternalResource) Key() ResourceKey {
	return ResourceKey{
		Name: p.Name,
		Kind: InternalKind,
	}
}
