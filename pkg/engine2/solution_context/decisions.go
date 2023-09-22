package solution_context

type (
	KV struct {
		key   string
		value any
	}

	DecisionRecords interface {
		// AddRecord stores each decision (the what) with the context (the why) in some datastore
		AddRecord(context []KV, decision SolveDecision)
		// // FindDecision returns the context (the why) for a given decision (the what)
		// FindDecision(decision SolveDecision) []KV
		// // FindContext returns the various decisions (the what) for a given context (the why)
		// FindContext(key string, value any) []SolveDecision
	}

	SolveDecision interface {
		// having a private method here prevents other packages from implementing this interface
		// not necessary, but could prevent some accidental bad practices from emerging
		internal()
	}
)
