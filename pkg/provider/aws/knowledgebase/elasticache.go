package knowledgebase

import (
	"fmt"
	"github.com/klothoplatform/klotho/pkg/core"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base"
	"github.com/klothoplatform/klotho/pkg/provider/aws/resources"
)

var ElasticacheKB = knowledgebase.Build(
	knowledgebase.EdgeBuilder[*resources.ElasticacheCluster, *resources.ElasticacheSubnetgroup]{
		Configure: func(cluster *resources.ElasticacheCluster, subnetgroup *resources.ElasticacheSubnetgroup, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			cluster.SubnetGroup = subnetgroup
			return nil
		},
	},
	knowledgebase.EdgeBuilder[*resources.ElasticacheCluster, *resources.SecurityGroup]{},
	knowledgebase.EdgeBuilder[*resources.ElasticacheCluster, *resources.LogGroup]{
		Configure: func(cluster *resources.ElasticacheCluster, logGroup *resources.LogGroup, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			logGroup.LogGroupName = fmt.Sprintf("/aws/elasticache/%s", cluster.Name)
			logGroup.RetentionInDays = 5
			return nil
		},
	},
	knowledgebase.EdgeBuilder[*resources.ElasticacheSubnetgroup, *resources.Subnet]{},
)
