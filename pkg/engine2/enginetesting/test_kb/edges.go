package kbtesting

import knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base2"

var edge1 = &knowledgebase.EdgeTemplate{
	Source:      resource1.Id(),
	Destination: resource2.Id(),
}

var edge2 = &knowledgebase.EdgeTemplate{
	Source:      resource2.Id(),
	Destination: resource3.Id(),
}

var edge3 = &knowledgebase.EdgeTemplate{
	Source:      resource1.Id(),
	Destination: resource4.Id(),
}
