package aws

import (
	"fmt"

	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/provider/aws/resources"
)

func (a *AWS) expandRedisNode(dag *core.ResourceGraph, construct *core.RedisNode) error {
	redis, err := core.CreateResource[*resources.ElasticacheCluster](dag, resources.ElasticacheClusterCreateParams{
		AppName: a.Config.AppName,
		Refs:    core.AnnotationKeySetOf(construct.AnnotationKey),
		Name:    construct.ID,
	})
	if err != nil {
		return err
	}
	return a.MapResourceToConstruct(redis, construct)
}

func (a *AWS) getElasticacheConfiguration(result *core.ConstructGraph, refs core.AnnotationKeySet) (resources.ElasticacheClusterConfigureParams, error) {
	clusterConfig := resources.ElasticacheClusterConfigureParams{}
	ref, oneRef := refs.GetSingle()
	if !oneRef {
		return clusterConfig, fmt.Errorf("elasticache cluster must only have one construct reference")
	}
	construct := result.GetConstruct(ref.ToId())
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
