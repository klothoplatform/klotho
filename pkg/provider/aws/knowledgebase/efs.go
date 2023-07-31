package knowledgebase

import (
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base"
	"github.com/klothoplatform/klotho/pkg/provider/aws/resources"
)

var EfsKB = knowledgebase.Build(
	knowledgebase.EdgeBuilder[*resources.EfsAccessPoint, *resources.EfsFileSystem]{},
	knowledgebase.EdgeBuilder[*resources.EfsMountTarget, *resources.EfsFileSystem]{},
	knowledgebase.EdgeBuilder[*resources.EfsAccessPoint, *resources.EfsMountTarget]{},
	knowledgebase.EdgeBuilder[*resources.EfsMountTarget, *resources.Subnet]{},
	knowledgebase.EdgeBuilder[*resources.EfsMountTarget, *resources.SecurityGroup]{},
	knowledgebase.EdgeBuilder[*resources.EfsFileSystem, *resources.KmsKey]{},
	knowledgebase.EdgeBuilder[*resources.EfsFileSystem, *resources.AvailabilityZones]{},
)
