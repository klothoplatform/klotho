package construct2

type Resource struct {
	ID         ResourceId
	Properties Properties
}

// Id is a temporary bridge to the old Resource interface.
//
// Deprecated: remove in favour of direct ID field access.
func (r *Resource) Id() ResourceId {
	return r.ID
}

func CreateResource(id ResourceId) *Resource {
	return &Resource{
		ID:         id,
		Properties: make(Properties),
	}
}
