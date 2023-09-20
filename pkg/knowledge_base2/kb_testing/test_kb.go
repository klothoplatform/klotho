package kbtesting

import knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base2"

func CreateTestKB() *knowledgebase.KnowledgeBase {
	testKB := knowledgebase.NewKB()
	testKB.AddResourceTemplate(resource1)
	testKB.AddResourceTemplate(resource2)
	testKB.AddResourceTemplate(resource3)
	testKB.AddResourceTemplate(resource4)
	testKB.AddEdgeTemplate(edge1)
	testKB.AddEdgeTemplate(edge2)
	testKB.AddEdgeTemplate(edge3)
	return testKB
}

var TestKB = CreateTestKB()
