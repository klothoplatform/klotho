package resources

import (
	"fmt"

	"github.com/klothoplatform/klotho/pkg/config"
	"github.com/klothoplatform/klotho/pkg/core"
)

type (
	Elasticache struct {
		Name            string
		Engine          string
		CloudwatchGroup *LogGroup
		SubnetGroup     *ElasticacheSubnetgroup
		SecurityGroups  []*SecurityGroup
		ConstructsRef   []core.AnnotationKey
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

// Provider returns name of the provider the resource is correlated to
func (ec *Elasticache) Provider() string {
	return AWS_PROVIDER
}

// KlothoResource returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (ec *Elasticache) KlothoConstructRef() []core.AnnotationKey {
	return ec.ConstructsRef
}

// ID returns the id of the cloud resource
func (ec *Elasticache) Id() string {
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

func CreateElasticache(cfg *config.Application, dag *core.ResourceGraph, source core.Construct) *Elasticache {
	ec := &Elasticache{
		Name:            source.Id(),
		Engine:          "redis", // TODO determine this from the type of `source`
		CloudwatchGroup: NewLogGroup(cfg.AppName, fmt.Sprintf("/aws/elasticache/%s-%s-persist-redis", cfg.AppName, source.Id()), source.Provenance(), 0),
		SubnetGroup: &ElasticacheSubnetgroup{
			Name:          source.Id(),
			Subnets:       GetSubnets(cfg, dag), // TODO when we allow for segmented networks, need to determine which network (subnets) this lives in
			ConstructsRef: []core.AnnotationKey{source.Provenance()},
		},
		SecurityGroups: []*SecurityGroup{GetSecurityGroup(cfg, dag)},
		ConstructsRef:  []core.AnnotationKey{source.Provenance()},
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
