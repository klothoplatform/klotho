package aws

import (
	"fmt"

	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/provider/aws/resources"
)

func (a *AWS) expandRedisNode(dag *core.ResourceGraph, construct *core.RedisNode) error {
	redis, err := core.CreateResource[*resources.ElasticacheCluster](dag, resources.ElasticacheClusterCreateParams{
		AppName: a.Config.AppName,
		Refs:    core.BaseConstructSetOf(construct),
		Name:    construct.Name,
	})
	if err != nil {
		return err
	}
	a.MapResourceDirectlyToConstruct(redis, construct)
	return nil
}

func (a *AWS) getElasticacheConfiguration(result *core.ConstructGraph, refs core.BaseConstructSet) (resources.ElasticacheClusterConfigureParams, error) {
	clusterConfig := resources.ElasticacheClusterConfigureParams{}
	if len(refs) > 1 || len(refs) == 0 {
		return clusterConfig, fmt.Errorf("elasticache cluster must only have one construct reference")
	}
	var ref core.ResourceId
	for r := range refs {
		ref = r
	}
	construct := result.GetConstruct(ref)
	if construct == nil {
		return clusterConfig, fmt.Errorf("construct with id %s does not exist", ref)
	}
	if _, ok := construct.(*core.RedisNode); !ok {
		return clusterConfig, fmt.Errorf("elasticache cluster must only have a construct reference to a redis node")
	}

	return resources.ElasticacheClusterConfigureParams{
		NumCacheNodes: 1,
		NodeType:      "cache.t3.micro",
		Engine:        "redis",
	}, nil
}
