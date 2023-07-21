package resources

import (
	"fmt"

	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/sanitization/aws"
)

type (
	ElasticacheCluster struct {
		Name            string
		Engine          string
		CloudwatchGroup *LogGroup
		SubnetGroup     *ElasticacheSubnetgroup
		SecurityGroups  []*SecurityGroup
		ConstructRefs   core.BaseConstructSet `yaml:"-"`
		NodeType        string
		NumCacheNodes   int
	}

	ElasticacheSubnetgroup struct {
		Name          string
		Subnets       []*Subnet
		ConstructRefs core.BaseConstructSet `yaml:"-"`
	}
)

const (
	ELASTICACHE_CLUSTER_TYPE     = "elasticache_cluster"
	ELASTICACHE_SUBNETGROUP_TYPE = "elasticache_subnetgroup"
)

// BaseConstructRefs returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (ec *ElasticacheCluster) BaseConstructRefs() core.BaseConstructSet {
	return ec.ConstructRefs
}

// Id returns the id of the cloud resource
func (ec *ElasticacheCluster) Id() core.ResourceId {
	return core.ResourceId{
		Provider: AWS_PROVIDER,
		Type:     ELASTICACHE_CLUSTER_TYPE,
		Name:     ec.Name,
	}
}

func (ec *ElasticacheCluster) DeleteContext() core.DeleteContext {
	return core.DeleteContext{
		RequiresNoUpstream:     true,
		RequiresNoDownstream:   true,
		RequiresExplicitDelete: true,
	}
}

// BaseConstructRefs returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (ecsn *ElasticacheSubnetgroup) BaseConstructRefs() core.BaseConstructSet {
	return ecsn.ConstructRefs
}

// Id returns the id of the cloud resource
func (ecsn *ElasticacheSubnetgroup) Id() core.ResourceId {
	return core.ResourceId{
		Provider: AWS_PROVIDER,
		Type:     ELASTICACHE_SUBNETGROUP_TYPE,
		Name:     ecsn.Name,
	}
}

func (ecsn *ElasticacheSubnetgroup) DeleteContext() core.DeleteContext {
	return core.DeleteContext{
		RequiresNoUpstream: true,
	}
}

type ElasticacheClusterCreateParams struct {
	AppName string
	Refs    core.BaseConstructSet
	Name    string
}

func (ec *ElasticacheCluster) Create(dag *core.ResourceGraph, params ElasticacheClusterCreateParams) error {
	ec.Name = aws.ElasticacheClusterSanitizer.Apply(fmt.Sprintf("%s-%s", params.AppName, params.Name))
	ec.ConstructRefs = params.Refs.Clone()
	ec.SecurityGroups = make([]*SecurityGroup, 1)

	if existingCluster, ok := core.GetResource[*ElasticacheCluster](dag, ec.Id()); ok {
		existingCluster.ConstructRefs.AddAll(params.Refs)
		return nil
	}
	dag.AddResource(ec)
	return nil
}

type ElasticacheSubnetgroupCreateParams struct {
	Refs    core.BaseConstructSet
	AppName string
	Name    string
}

func (ecsn *ElasticacheSubnetgroup) Create(dag *core.ResourceGraph, params ElasticacheSubnetgroupCreateParams) error {
	ecsn.Name = aws.ElasticacheClusterSanitizer.Apply(fmt.Sprintf("%s-%s", params.AppName, params.Name))
	ecsn.ConstructRefs = params.Refs.Clone()
	if existingSubnetGroup, ok := core.GetResource[*ElasticacheSubnetgroup](dag, ecsn.Id()); ok {
		existingSubnetGroup.ConstructRefs.AddAll(params.Refs)
		return nil
	}
	dag.AddResource(ecsn)
	return nil
}
