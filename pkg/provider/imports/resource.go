package imports

import "github.com/klothoplatform/klotho/pkg/core"

// Imported is an internal resource to signal that the resource that depends on this
// should be imported from `ID`.
//
//	OriginalResource -> Imported
//
// Any IaC rendering should replace the resource with
// this import and not render the original resource.
type Imported struct {
	ID string
}

func (imp Imported) KlothoConstructRef() []core.AnnotationKey {
	return nil
}

func (imp Imported) Id() core.ResourceId {
	return core.ResourceId{
		Provider: core.InternalProvider,
		Type:     "import",
		Name:     imp.ID,
	}
}
