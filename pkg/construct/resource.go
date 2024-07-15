package construct

type Resource struct {
	ID         ResourceId
	Properties Properties
	Imported   bool
}

func (r Resource) Equals(other any) bool {
	switch other := other.(type) {
	case Resource:
		return r.ID == other.ID && r.Properties.Equals(other.Properties)
	case *Resource:
		return r.ID == other.ID && r.Properties.Equals(other.Properties)
	default:
		return false
	}
}
