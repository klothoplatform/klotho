package construct

type EdgeData struct {
	ConnectionType string `yaml:"connection_type,omitempty" json:"connection_type,omitempty"`
}

// Equals implements an interface used in [graph_addons.MemoryStore] to determine whether edges are equal
// to allow for idempotent edge addition.
func (ed EdgeData) Equals(other any) bool {
	if other, ok := other.(EdgeData); ok {
		return ed == other
	}

	return false
}
