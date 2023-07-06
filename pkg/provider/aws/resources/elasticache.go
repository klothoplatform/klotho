package resources

import (
	"fmt"

	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/engine/classification"
	"github.com/klothoplatform/klotho/pkg/sanitization/aws"
)

type (
	ElasticacheCluster struct {
		Name            string
		Engine          string
		CloudwatchGroup *LogGroup
		SubnetGroup     *ElasticacheSubnetgroup
		SecurityGroups  []*SecurityGroup
		ConstructsRef   core.BaseConstructSet `yaml:"-"`
		NodeType        string
		NumCacheNodes   int
	}

	ElasticacheSubnetgroup struct {
		Name          string
		Subnets       []*Subnet
		ConstructsRef core.BaseConstructSet `yaml:"-"`
	}
)

const (
	ELASTICACHE_CLUSTER_TYPE     = "elasticache_cluster"
	ELASTICACHE_SUBNETGROUP_TYPE = "elasticache_subnetgroup"
)

// BaseConstructsRef returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (ec *ElasticacheCluster) BaseConstructsRef() core.BaseConstructSet {
	return ec.ConstructsRef
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

// BaseConstructsRef returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (ecsn *ElasticacheSubnetgroup) BaseConstructsRef() core.BaseConstructSet {
	return ecsn.ConstructsRef
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
	ec.ConstructsRef = params.Refs.Clone()
	ec.SecurityGroups = make([]*SecurityGroup, 1)

	if existingCluster, ok := core.GetResource[*ElasticacheCluster](dag, ec.Id()); ok {
		existingCluster.ConstructsRef.AddAll(params.Refs)
		return nil
	}
	dag.AddResource(ec)
	return nil
}

func (cluster *ElasticacheCluster) MakeOperational(dag *core.ResourceGraph, appName string, classifier classification.Classifier) error {
	if cluster.CloudwatchGroup == nil {
		logGroups := core.GetDownstreamResourcesOfType[*LogGroup](dag, cluster)
		if len(logGroups) == 0 {
			logGroup, err := core.CreateResource[*LogGroup](dag, CloudwatchLogGroupCreateParams{
				AppName: appName,
				Name:    fmt.Sprintf("%s-loggroup", cluster.Name),
				Refs:    core.BaseConstructSetOf(cluster),
			})
			if err != nil {
				return err
			}
			cluster.CloudwatchGroup = logGroup
		} else if len(logGroups) > 1 {
			return fmt.Errorf("elasticache cluster %s has more than one log group downstream", cluster.Id())
		} else {
			cluster.CloudwatchGroup = logGroups[0]
		}
		dag.AddDependenciesReflect(cluster)
	}

	if cluster.SubnetGroup == nil {
		subnetGroups := core.GetDownstreamResourcesOfType[*ElasticacheSubnetgroup](dag, cluster)
		if len(subnetGroups) == 0 {
			vpc, err := getSingleUpstreamVpc(dag, cluster)
			if err != nil {
				return err
			}
			subnetGroup, err := core.CreateResource[*ElasticacheSubnetgroup](dag, ElasticacheSubnetgroupCreateParams{
				AppName: appName,
				Name:    fmt.Sprintf("%s-subnetgroup", cluster.Name),
				Refs:    core.BaseConstructSetOf(cluster),
			})
			if err != nil {
				return err
			}
			if vpc != nil {
				dag.AddDependency(subnetGroup, vpc)
			}
			err = subnetGroup.MakeOperational(dag, appName, classifier)
			if err != nil {
				return err
			}
			cluster.SubnetGroup = subnetGroup
		} else if len(subnetGroups) > 1 {
			return fmt.Errorf("elasticache cluster %s has more than one subnet group downstream", cluster.Id())
		} else {
			cluster.SubnetGroup = subnetGroups[0]
		}
		dag.AddDependenciesReflect(cluster)
	}

	if len(cluster.SecurityGroups) == 0 {
		securityGroups, err := getSecurityGroupsOperational(dag, cluster, appName)
		if err != nil {
			return err
		}
		cluster.SecurityGroups = securityGroups
		dag.AddDependenciesReflect(cluster)
	}

	return nil
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
	Refs    core.BaseConstructSet
	AppName string
	Name    string
}

func (ecsn *ElasticacheSubnetgroup) Create(dag *core.ResourceGraph, params ElasticacheSubnetgroupCreateParams) error {
	ecsn.Name = aws.ElasticacheClusterSanitizer.Apply(fmt.Sprintf("%s-%s", params.AppName, params.Name))
	ecsn.ConstructsRef = params.Refs.Clone()
	if existingSubnetGroup, ok := core.GetResource[*ElasticacheSubnetgroup](dag, ecsn.Id()); ok {
		existingSubnetGroup.ConstructsRef.AddAll(params.Refs)
		return nil
	}
	dag.AddResource(ecsn)
	return nil
}

func (subnetGroup *ElasticacheSubnetgroup) MakeOperational(dag *core.ResourceGraph, appName string, classifier classification.Classifier) error {
	if len(subnetGroup.Subnets) == 0 {
		subnets, err := getSubnetsOperational(dag, subnetGroup, appName)
		if err != nil {
			return err
		}
		for _, subnet := range subnets {
			if subnet.Type == PrivateSubnet {
				subnetGroup.Subnets = append(subnetGroup.Subnets, subnet)
			}
		}
		dag.AddDependenciesReflect(subnetGroup)
	}
	return nil
}
