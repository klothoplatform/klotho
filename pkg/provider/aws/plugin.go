package aws

import (
	"sort"

	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/infra/kubernetes"
	"github.com/klothoplatform/klotho/pkg/multierr"
	"github.com/klothoplatform/klotho/pkg/provider/aws/resources"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

func (a *AWS) Translate(result *core.ConstructGraph, dag *core.ResourceGraph) (links []core.CloudResourceLink, err error) {
	log := zap.S()

	err = a.createEksClusters(result, dag)
	if err != nil {
		return
	}
	constructIds, err := result.TopologicalSort()
	if err != nil {
		return
	}
	// We want to reverse the list so that we start at the leaf nodes. This allows us to check downstream dependencies each time and process them.
	reverseInPlace(constructIds)
	var merr multierr.Error
	for _, id := range constructIds {
		construct := result.GetConstruct(id)
		log.Debugf("Converting construct with id, %s, to aws resources", construct.Id())
		switch construct := construct.(type) {
		case *core.ExecutionUnit:
			merr.Append(a.GenerateExecUnitResources(construct, result, dag))
		case *core.StaticUnit:
			merr.Append(a.GenerateStaticUnitResources(construct, dag))
		case *core.Gateway:
			merr.Append(a.GenerateExposeResources(construct, result, dag))
		case *core.Fs:
			merr.Append(a.GenerateFsResources(construct, result, dag))
		case *core.Secrets:
			merr.Append(a.GenerateSecretsResources(construct, result, dag))
		case *core.Kv:
			merr.Append(a.GenerateKvResources(construct, result, dag))
		case *core.RedisCluster, *core.RedisNode:
			merr.Append(a.GenerateRedisResources(construct, result, dag))
		default:
			// TODO convert to error once migration to ifc2 is complete
			log.Warnf("Unsupported resource %s", construct.Id())
		}
	}
	if err = merr.ErrOrNil(); err != nil {
		return
	}

	err = a.convertExecUnitParams(result, dag)
	if err != nil {
		return
	}
	err = a.createCDNs(result, dag)
	if err != nil {
		return
	}
	return
}

// createEksCluster determines whether any execution units have a type of EKS to determine whether a cluster needs to be created.
//
// If any units do have a type of EKS, the function will look at their configuration to determine how the cluster needs to be configured.
// The clusterId field within an execution units configuration, will determine which cluster the unit will belong to, helping klotho understands how many clusters to create.
// If the clusterId field is unassigned, the execution unit will be assigned to the first clusterId, if only one exists.
// If multiple clusters exist we will throw an error since we cannot determine which exec unit belongs to which cluster.
// If there are no clusterIds defined by any units, one cluster will be created for all units.
func (a *AWS) createEksClusters(result *core.ConstructGraph, dag *core.ResourceGraph) error {
	unassignedUnits := []*core.ExecutionUnit{}
	clusterIdToUnit := map[string][]*core.ExecutionUnit{}
	for _, unit := range core.GetResourcesOfType[*core.ExecutionUnit](result) {
		cfg := a.Config.GetExecutionUnit(unit.Provenance().ID)
		if cfg.Type == kubernetes.KubernetesType {
			params := cfg.GetExecutionUnitParamsAsKubernetes()
			if params.ClusterId == "" {
				unassignedUnits = append(unassignedUnits, unit)
				continue
			}
			clusterIdToUnit[params.ClusterId] = append(clusterIdToUnit[params.ClusterId], unit)

		}
	}

	//Assign unassigned units to the first key after sorted
	keys := []string{}
	for k := range clusterIdToUnit {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// If multiple clusters exist and there are unassigned units, error out since we can not determine where they should belong
	if len(keys) > 1 && len(unassignedUnits) != 0 {
		return errors.Errorf("Unable to determine which cluster, units %v belong to", unassignedUnits)
	}

	// If no units are defined in config, create a defaultly named one
	if len(keys) == 0 {
		keys = append(keys, resources.DEFAULT_CLUSTER_NAME)
	}

	// Assign all units to the cluster that exists
	if len(unassignedUnits) != 0 {
		clusterIdToUnit[keys[0]] = append(clusterIdToUnit[keys[0]], unassignedUnits...)
	}

	vpc := resources.CreateNetwork(a.Config, dag)
	sg := resources.GetSecurityGroup(a.Config, dag)
	sg.IngressRules = append(sg.IngressRules, resources.SecurityGroupRule{
		Description: "Allows ingress traffic from the EKS control plane",
		FromPort:    9443,
		Protocol:    "TCP",
		ToPort:      9443,
		CidrBlocks: []core.IaCValue{
			{Property: "0.0.0.0/0"},
		},
	})
	subnets := vpc.GetVpcSubnets(dag)

	var merr multierr.Error
	for clusterId, units := range clusterIdToUnit {
		merr.Append(resources.CreateEksCluster(a.Config.AppName, clusterId, subnets, nil, units, dag))
	}
	return merr.ErrOrNil()
}

func reverseInPlace[A any](a []A) {
	// taken from https://github.com/golang/go/wiki/SliceTricks/33793edcc2c7aee6448ed1dd0c36524eddfdf1e2#reversing
	for i := len(a)/2 - 1; i >= 0; i-- {
		opp := len(a) - 1 - i
		a[i], a[opp] = a[opp], a[i]
	}
}
