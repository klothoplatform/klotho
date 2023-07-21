package enginetesting

import (
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base"
)

var MockKB = knowledgebase.Build(
	knowledgebase.EdgeBuilder[*MockResource1, *MockResource2]{},
	knowledgebase.EdgeBuilder[*MockResource1, *MockResource3]{},
	knowledgebase.EdgeBuilder[*MockResource1, *MockResource4]{},
	knowledgebase.EdgeBuilder[*MockResource2, *MockResource3]{},
	// used for operational resource testing
	knowledgebase.EdgeBuilder[*MockResource5, *MockResource1]{},
	knowledgebase.EdgeBuilder[*MockResource5, *MockResource2]{},
)
