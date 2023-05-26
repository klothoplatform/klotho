package knowledgebase

import (
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base"
	"github.com/klothoplatform/klotho/pkg/provider/aws/resources"
)

var S3KB = knowledgebase.Build(
	knowledgebase.EdgeBuilder[*resources.S3Object, *resources.S3Bucket]{},
)
