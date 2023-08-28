package knowledgebase

import (
	"fmt"

	"github.com/klothoplatform/klotho/pkg/construct"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base"
	"github.com/klothoplatform/klotho/pkg/provider/aws/resources"
)

var ElasticacheKB = knowledgebase.Build(
	knowledgebase.EdgeBuilder[*resources.ElasticacheCluster, *resources.LogGroup]{
		Configure: func(cluster *resources.ElasticacheCluster, logGroup *resources.LogGroup, dag *construct.ResourceGraph, data knowledgebase.EdgeData) error {
			logGroup.LogGroupName = fmt.Sprintf("/aws/elasticache/%s", cluster.Name)
			return nil
		},
	},
)
