package resources

import (
	"fmt"

	"github.com/klothoplatform/klotho/pkg/config"
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

var (
	elasticacheClusterSanitizer = aws.ElasticacheClusterSanitizer
)

// Provider returns name of the provider the resource is correlated to
func (ec *ElasticacheCluster) Provider() string {
	return AWS_PROVIDER
}

// KlothoResource returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (ec *ElasticacheCluster) KlothoConstructRef() []core.AnnotationKey {
	return ec.ConstructsRef
}

// ID returns the id of the cloud resource
func (ec *ElasticacheCluster) Id() string {
	return fmt.Sprintf("%s:%s:%s", ec.Provider(), EC_TYPE, ec.Name)
}

// Provider returns name of the provider the resource is correlated to
func (ecsn *ElasticacheSubnetgroup) Provider() string {
	return AWS_PROVIDER
}

// KlothoResource returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (ecsn *ElasticacheSubnetgroup) KlothoConstructRef() []core.AnnotationKey {
	return ecsn.ConstructsRef
}

// ID returns the id of the cloud resource
func (ecsn *ElasticacheSubnetgroup) Id() string {
	return fmt.Sprintf("%s:%s:%s", ecsn.Provider(), ECSN_TYPE, ecsn.Name)
}

func CreateElasticache(cfg *config.Application, dag *core.ResourceGraph, source core.Construct) *ElasticacheCluster {
	ec := &ElasticacheCluster{
		Name:            elasticacheClusterSanitizer.Apply(fmt.Sprintf("%s-%s", cfg.AppName, source.Provenance().ID)),
		Engine:          "redis", // TODO determine this from the type of `source`
		CloudwatchGroup: NewLogGroup(cfg.AppName, fmt.Sprintf("/aws/elasticache/%s-%s-persist-redis", cfg.AppName, source.Id()), source.Provenance(), 0),
		SubnetGroup: &ElasticacheSubnetgroup{
			Name:          elasticacheClusterSanitizer.Apply(fmt.Sprintf("%s-%s", cfg.AppName, source.Provenance().ID)),
			Subnets:       GetSubnets(cfg, dag), // TODO when we allow for segmented networks, need to determine which network (subnets) this lives in
			ConstructsRef: []core.AnnotationKey{source.Provenance()},
		},
		SecurityGroups: []*SecurityGroup{GetSecurityGroup(cfg, dag)},
		ConstructsRef:  []core.AnnotationKey{source.Provenance()},
		NodeType:       "cache.t3.micro",
		NumCacheNodes:  1,
	}
	dag.AddResource(ec)
	dag.AddResource(ec.CloudwatchGroup)
	dag.AddResource(ec.SubnetGroup)
	dag.AddDependency(ec, ec.CloudwatchGroup)
	dag.AddDependency(ec, ec.SubnetGroup)

	for _, sg := range ec.SecurityGroups {
		sg.ConstructsRef = append(sg.ConstructsRef, source.Provenance())
		dag.AddDependency(ec, sg)
	}
	for _, sn := range ec.SubnetGroup.Subnets {
		sn.ConstructsRef = append(sn.ConstructsRef, source.Provenance())
		dag.AddDependency(ec.SubnetGroup, sn)
	}

	return ec
}
