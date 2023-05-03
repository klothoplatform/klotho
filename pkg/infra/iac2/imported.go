package iac2

import "github.com/klothoplatform/klotho/pkg/core"

// Imported is an internal resource to signal that the resource that depends on this
// should be imported from `ID`.
type Imported struct {
	ID string
}

func (imp Imported) KlothoConstructRef() []core.AnnotationKey {
	return nil
}

func (imp Imported) Id() core.ResourceId {
	return core.ResourceId{
		Provider: "pulumi",
		Type:     "import",
		Name:     imp.ID,
	}
}
