package solution_context

type (
	memoryRecord struct {
		records []record
	}

	record struct {
		context  []KV
		decision SolveDecision
	}
)

func (m *memoryRecord) AddRecord(context []KV, decision SolveDecision) {
	m.records = append(m.records, record{context: context, decision: decision})
}
