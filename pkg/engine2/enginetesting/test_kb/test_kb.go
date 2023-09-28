package kbtesting

import knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base2"

func CreateTestKB() *knowledgebase.KnowledgeBase {
	testKB := knowledgebase.NewKB()
	err := testKB.AddResourceTemplate(resource1)
	if err != nil {
		panic(err)
	}
	err = testKB.AddResourceTemplate(resource2)
	if err != nil {
		panic(err)
	}
	err = testKB.AddResourceTemplate(resource3)
	if err != nil {
		panic(err)
	}
	err = testKB.AddResourceTemplate(resource4)
	if err != nil {
		panic(err)
	}
	err = testKB.AddEdgeTemplate(edge1)
	if err != nil {
		panic(err)
	}
	err = testKB.AddEdgeTemplate(edge2)
	if err != nil {
		panic(err)
	}
	err = testKB.AddEdgeTemplate(edge3)
	if err != nil {
		panic(err)
	}
	return testKB
}

var TestKB = CreateTestKB()
