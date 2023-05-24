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
		ConstructsRef   []core.AnnotationKey
		NodeType        string
		NumCacheNodes   int
	}

	ElasticacheSubnetgroup struct {
		Name          string
		Subnets       []*Subnet
		ConstructsRef []core.AnnotationKey
	}
)

const (
	EC_TYPE   = "elasticache"
	ECSN_TYPE = "elasticache_subnetgroup"
)

// KlothoConstructRef returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (ec *ElasticacheCluster) KlothoConstructRef() []core.AnnotationKey {
	return ec.ConstructsRef
}

// Id returns the id of the cloud resource
func (ec *ElasticacheCluster) Id() core.ResourceId {
	return core.ResourceId{
		Provider: AWS_PROVIDER,
		Type:     EC_TYPE,
		Name:     ec.Name,
	}
}

// KlothoConstructRef returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (ecsn *ElasticacheSubnetgroup) KlothoConstructRef() []core.AnnotationKey {
	return ecsn.ConstructsRef
}

// Id returns the id of the cloud resource
func (ecsn *ElasticacheSubnetgroup) Id() core.ResourceId {
	return core.ResourceId{
		Provider: AWS_PROVIDER,
		Type:     ECSN_TYPE,
		Name:     ecsn.Name,
	}
}

type ElasticacheClusterCreateParams struct {
	AppName string
	Refs    []core.AnnotationKey
	Name    string
}

func (ec *ElasticacheCluster) Create(dag *core.ResourceGraph, params ElasticacheClusterCreateParams) error {
	ec.Name = aws.ElasticacheClusterSanitizer.Apply(fmt.Sprintf("%s-%s", params.AppName, params.Name))
	ec.ConstructsRef = params.Refs
	ec.SecurityGroups = make([]*SecurityGroup, 1)

	if existingCluster, ok := core.GetResource[*ElasticacheCluster](dag, ec.Id()); ok {
		existingCluster.ConstructsRef = core.DedupeAnnotationKeys(append(existingCluster.KlothoConstructRef(), params.Refs...))
	}

	subParams := map[string]any{
		"CloudwatchGroup": params,
		"SubnetGroup": ElasticacheSubnetgroupCreateParams{
			AppName: params.AppName,
			Name:    fmt.Sprintf("%s-subnetgroup", params.Name),
			Refs:    params.Refs,
		},
		"SecurityGroups": []SecurityGroupCreateParams{{
			AppName: params.AppName,
			Refs:    params.Refs,
		}},
	}

	err := dag.CreateDependencies(ec, subParams)
	return err
}

type ElasticacheClusterConfigureParams struct {
	Engine        string
	NodeType      string
	NumCacheNodes int
}

func (ec *ElasticacheCluster) Configure(params ElasticacheClusterConfigureParams) error {
	ec.Engine = params.Engine
	ec.NodeType = params.NodeType
	ec.NumCacheNodes = params.NumCacheNodes
	return nil
}

type ElasticacheSubnetgroupCreateParams struct {
	Refs    []core.AnnotationKey
	AppName string
	Name    string
}

func (ecsn *ElasticacheSubnetgroup) Create(dag *core.ResourceGraph, params ElasticacheSubnetgroupCreateParams) error {
	ecsn.Name = aws.ElasticacheClusterSanitizer.Apply(fmt.Sprintf("%s-%s", params.AppName, params.Name))
	ecsn.ConstructsRef = params.Refs
	ecsn.Subnets = make([]*Subnet, 2)
	if existingSubnetGroup, ok := core.GetResource[*ElasticacheSubnetgroup](dag, ecsn.Id()); ok {
		existingSubnetGroup.ConstructsRef = core.DedupeAnnotationKeys(append(existingSubnetGroup.KlothoConstructRef(), params.Refs...))
	}

	subParams := map[string]any{
		"Subnets": []SubnetCreateParams{
			{
				AppName: params.AppName,
				Refs:    params.Refs,
				AZ:      "0",
				Type:    PrivateSubnet,
			},
			{
				AppName: params.AppName,
				Refs:    params.Refs,
				AZ:      "1",
				Type:    PrivateSubnet,
			},
		},
	}

	err := dag.CreateDependencies(ecsn, subParams)
	return err
}
