package imports

import "github.com/klothoplatform/klotho/pkg/construct"

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

func (imp Imported) BaseConstructRefs() construct.BaseConstructSet {
	return nil
}

func (imp Imported) Id() construct.ResourceId {
	return construct.ResourceId{
		Provider: construct.InternalProvider,
		Type:     "import",
		Name:     imp.ID,
	}
}

func (imp Imported) DeleteContext() construct.DeleteContext {
	return construct.DeleteContext{
		RequiresNoUpstream: true,
	}
}
