package operational_eval

type internalVertex struct {
	Internal string
}

func (v *internalVertex) Key() Key {
	return Key{Internal: v.Internal}
}

func (v *internalVertex) Evaluate(eval *Evaluator) error {
	return nil
}

func (v *internalVertex) UpdateFrom(other Vertex) {}

func (v *internalVertex) Dependencies(eval *Evaluator) (graphChanges, error) {
	return graphChanges{}, nil
}
