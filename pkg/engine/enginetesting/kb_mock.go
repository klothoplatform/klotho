package enginetesting

import (
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base"
)

var MockKB = knowledgebase.Build(
	knowledgebase.EdgeBuilder[*mockResource1, *mockResource2]{},
	knowledgebase.EdgeBuilder[*mockResource1, *mockResource3]{},
	knowledgebase.EdgeBuilder[*mockResource1, *mockResource4]{},
	knowledgebase.EdgeBuilder[*mockResource2, *mockResource3]{},
)
