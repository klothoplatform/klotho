package aws

import (
	"sort"

	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/provider/aws/resources"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

func (a *AWS) Translate(result *core.ConstructGraph, dag *core.ResourceGraph) (Links []core.CloudResourceLink, err error) {
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
	for _, id := range constructIds {
		construct := result.GetConstruct(id)
		log.Debugf("Converting construct with id, %s, to aws resources", construct.Id())
		switch construct := construct.(type) {
		case *core.ExecutionUnit:
			err = a.GenerateExecUnitResources(construct, result, dag)
			if err != nil {
				return
			}
		case *core.StaticUnit:
			err = a.GenerateStaticUnitResources(construct, dag)
			if err != nil {
				return
			}
		case *core.Gateway:
			err = a.GenerateExposeResources(construct, result, dag)
			if err != nil {
				return
			}
		case *core.Fs:
			err = a.GenerateFsResources(construct, result, dag)
			if err != nil {
				return
			}
		case *core.Secrets:
			err = a.GenerateSecretsResources(construct, result, dag)
			if err != nil {
				return
			}
		case *core.Kv:
			err = a.GenerateKvResources(construct, result, dag)
			if err != nil {
				return
			}
		default:
			log.Warnf("Unsupported construct %s", construct.Id())
		}
	}

	err = a.convertExecUnitParams(result, dag)
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
		if cfg.Type == Kubernetes {
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
		keys = append(keys, "eks-cluster")
	}

	// Assign all units to the cluster that exists
	if len(unassignedUnits) != 0 {
		clusterIdToUnit[keys[0]] = append(clusterIdToUnit[keys[0]], unassignedUnits...)
	}

	vpc := resources.CreateNetwork(a.Config.AppName, dag)
	subnets := vpc.GetVpcSubnets(dag)
	for clusterId, units := range clusterIdToUnit {
		resources.CreateEksCluster(a.Config.AppName, clusterId, subnets, nil, units, dag)
	}
	return nil
}

func reverseInPlace[A any](a []A) {
	// taken from https://github.com/golang/go/wiki/SliceTricks/33793edcc2c7aee6448ed1dd0c36524eddfdf1e2#reversing
	for i := len(a)/2 - 1; i >= 0; i-- {
		opp := len(a) - 1 - i
		a[i], a[opp] = a[opp], a[i]
	}
}
