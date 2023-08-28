package resources

import (
	"fmt"

	"github.com/klothoplatform/klotho/pkg/construct"
	"github.com/klothoplatform/klotho/pkg/sanitization/aws"
)

type (
	ElasticacheCluster struct {
		Name            string
		Engine          string
		CloudwatchGroup *LogGroup
		SubnetGroup     *ElasticacheSubnetgroup
		SecurityGroups  []*SecurityGroup
		ConstructRefs   construct.BaseConstructSet `yaml:"-"`
		NodeType        string
		NumCacheNodes   int
	}

	ElasticacheSubnetgroup struct {
		Name          string
		Subnets       []*Subnet
		ConstructRefs construct.BaseConstructSet `yaml:"-"`
	}
)

const (
	ELASTICACHE_CLUSTER_TYPE     = "elasticache_cluster"
	ELASTICACHE_SUBNETGROUP_TYPE = "elasticache_subnetgroup"
)

// BaseConstructRefs returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (ec *ElasticacheCluster) BaseConstructRefs() construct.BaseConstructSet {
	return ec.ConstructRefs
}

// Id returns the id of the cloud resource
func (ec *ElasticacheCluster) Id() construct.ResourceId {
	return construct.ResourceId{
		Provider: AWS_PROVIDER,
		Type:     ELASTICACHE_CLUSTER_TYPE,
		Name:     ec.Name,
	}
}

func (ec *ElasticacheCluster) DeleteContext() construct.DeleteContext {
	return construct.DeleteContext{
		RequiresNoUpstream:     true,
		RequiresNoDownstream:   true,
		RequiresExplicitDelete: true,
	}
}

// BaseConstructRefs returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (ecsn *ElasticacheSubnetgroup) BaseConstructRefs() construct.BaseConstructSet {
	return ecsn.ConstructRefs
}

// Id returns the id of the cloud resource
func (ecsn *ElasticacheSubnetgroup) Id() construct.ResourceId {
	return construct.ResourceId{
		Provider: AWS_PROVIDER,
		Type:     ELASTICACHE_SUBNETGROUP_TYPE,
		Name:     ecsn.Name,
	}
}

func (ecsn *ElasticacheSubnetgroup) DeleteContext() construct.DeleteContext {
	return construct.DeleteContext{
		RequiresNoUpstream: true,
	}
}

type ElasticacheClusterCreateParams struct {
	AppName string
	Refs    construct.BaseConstructSet
	Name    string
}

func (ec *ElasticacheCluster) Create(dag *construct.ResourceGraph, params ElasticacheClusterCreateParams) error {
	ec.Name = aws.ElasticacheClusterSanitizer.Apply(fmt.Sprintf("%s-%s", params.AppName, params.Name))
	ec.ConstructRefs = params.Refs.Clone()
	ec.SecurityGroups = make([]*SecurityGroup, 1)

	if existingCluster, ok := construct.GetResource[*ElasticacheCluster](dag, ec.Id()); ok {
		existingCluster.ConstructRefs.AddAll(params.Refs)
		return nil
	}
	dag.AddResource(ec)
	return nil
}

type ElasticacheSubnetgroupCreateParams struct {
	Refs    construct.BaseConstructSet
	AppName string
	Name    string
}

func (ecsn *ElasticacheSubnetgroup) Create(dag *construct.ResourceGraph, params ElasticacheSubnetgroupCreateParams) error {
	ecsn.Name = aws.ElasticacheClusterSanitizer.Apply(fmt.Sprintf("%s-%s", params.AppName, params.Name))
	ecsn.ConstructRefs = params.Refs.Clone()
	if existingSubnetGroup, ok := construct.GetResource[*ElasticacheSubnetgroup](dag, ecsn.Id()); ok {
		existingSubnetGroup.ConstructRefs.AddAll(params.Refs)
		return nil
	}
	dag.AddResource(ecsn)
	return nil
}
