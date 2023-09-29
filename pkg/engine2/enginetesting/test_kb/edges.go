package kbtesting

import knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base2"

var edge1 = &knowledgebase.EdgeTemplate{
	Source: resource1.Id(),
	Target: resource2.Id(),
}

var edge2 = &knowledgebase.EdgeTemplate{
	Source: resource2.Id(),
	Target: resource3.Id(),
}

var edge3 = &knowledgebase.EdgeTemplate{
	Source: resource1.Id(),
	Target: resource4.Id(),
}
