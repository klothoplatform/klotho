package knowledgebase

import (
	"github.com/klothoplatform/klotho/pkg/core"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base"
	"github.com/klothoplatform/klotho/pkg/provider/aws/resources"
)

var LbKB = knowledgebase.Build(
	knowledgebase.EdgeBuilder[*resources.TargetGroup, *resources.Vpc]{},
	knowledgebase.EdgeBuilder[*resources.Listener, *resources.TargetGroup]{},
	knowledgebase.EdgeBuilder[*resources.Listener, *resources.LoadBalancer]{
		ValidDestinations: []core.Resource{&resources.TargetGroup{}},
	},
	knowledgebase.EdgeBuilder[*resources.LoadBalancer, *resources.Subnet]{},
	knowledgebase.EdgeBuilder[*resources.LoadBalancer, *resources.SecurityGroup]{},
)
