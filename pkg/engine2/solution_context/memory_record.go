package solution_context

type (
	MemoryRecord struct {
		records []record
	}

	record struct {
		context  []KV
		decision SolveDecision
	}
)

func (m *MemoryRecord) AddRecord(context []KV, decision SolveDecision) {
	m.records = append(m.records, record{context: context, decision: decision})
}

func (m *MemoryRecord) GetRecords() []SolveDecision {
	var decisions []SolveDecision
	for _, record := range m.records {
		decisions = append(decisions, record.decision)
	}
	return decisions
}
