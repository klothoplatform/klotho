package solution

import "sync"

type DecisionRecords struct {
	mu      sync.Mutex
	records []SolveDecision
}

func (r *DecisionRecords) RecordDecision(d SolveDecision) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.records == nil {
		r.records = []SolveDecision{d}
		return
	}
	r.records = append(r.records, d)
}

func (r *DecisionRecords) GetDecisions() []SolveDecision {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.records
}
