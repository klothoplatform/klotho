package kbtesting

import (
	"testing"

	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base2"
)

func CreateTestKB(t *testing.T) *knowledgebase.KnowledgeBase {
	testKB := knowledgebase.NewKB()
	must := func(err error) {
		if err != nil {
			t.Fatal(err)
		}
	}
	must(testKB.AddResourceTemplate(resource1))
	must(testKB.AddResourceTemplate(resource2))
	must(testKB.AddResourceTemplate(resource3))
	must(testKB.AddResourceTemplate(resource4))
	must(testKB.AddEdgeTemplate(edge1))
	must(testKB.AddEdgeTemplate(edge2))
	must(testKB.AddEdgeTemplate(edge3))
	return testKB
}
