package construct

// SortedIds is a helper type for sorting ResourceIds by purely their content, for use when deterministic ordering
// is desired (when no other sources of ordering are available).
type SortedIds []ResourceId

func (s SortedIds) Len() int {
	return len(s)
}

func ResourceIdLess(a, b ResourceId) bool {
	if a.Provider != b.Provider {
		return a.Provider < b.Provider
	}
	if a.Type != b.Type {
		return a.Type < b.Type
	}
	if a.Namespace != b.Namespace {
		return a.Namespace < b.Namespace
	}
	return a.Name < b.Name
}

func (s SortedIds) Less(i, j int) bool {
	return ResourceIdLess(s[i], s[j])
}

func (s SortedIds) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}
