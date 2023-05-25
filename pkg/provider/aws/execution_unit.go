package aws

import (
	"fmt"

	"github.com/klothoplatform/klotho/pkg/config"
	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/infra/kubernetes"
	"github.com/klothoplatform/klotho/pkg/provider/aws/knowledgebase"
	"github.com/klothoplatform/klotho/pkg/provider/aws/resources"
)

// expandExecutionUnit takes in a single execution unit and expands the generic construct into a set of resource's based on the units configuration.
func (a *AWS) expandExecutionUnit(dag *core.ResourceGraph, unit *core.ExecutionUnit) error {
	switch a.Config.GetExecutionUnit(unit.ID).Type {
	case Lambda:
		lambda, err := core.CreateResource[*resources.LambdaFunction](dag, resources.LambdaCreateParams{
			AppName: a.Config.AppName,
			Refs:    core.AnnotationKeySetOf(unit.AnnotationKey),
			Name:    unit.ID,
		})
		if err != nil {
			return err
		}
		a.MapResourceDirectlyToConstruct(lambda, unit)
	case kubernetes.KubernetesType:
		params := config.ConvertFromInfraParams[config.KubernetesTypeParams](a.Config.GetExecutionUnit(unit.ID).InfraParams)
		clusterName := params.ClusterId
		if clusterName == "" {
			clusterName = "cluster"
		}
		var fargateProfile *resources.EksFargateProfile
		if params.NodeType == "fargate" {
			fargateProfile = &resources.EksFargateProfile{}
			err := fargateProfile.Create(dag, resources.EksFargateProfileCreateParams{
				Name:        "klotho-fargate-profile",
				ClusterName: clusterName,
				Refs:        core.AnnotationKeySetOf(unit.AnnotationKey),
				AppName:     a.Config.AppName,
				NetworkType: a.Config.GetExecutionUnit(unit.ID).NetworkPlacement,
			})
			if err != nil {
				return err
			}
		}
		helmChart, err := findUnitsHelmChart(unit, dag)
		if err != nil {
			return err
		}
		helmChart.ClustersProvider = core.IaCValue{
			Resource: &resources.EksCluster{},
			Property: resources.CLUSTER_PROVIDER_IAC_VALUE,
		}
		subParams := map[string]any{
			"ClustersProvider": resources.EksClusterCreateParams{
				Refs:    core.AnnotationKeySetOf(unit.AnnotationKey),
				AppName: a.Config.AppName,
				Name:    clusterName,
			},
		}
		subParams["Values"], err = a.handleHelmChartAwsValues(helmChart, unit, dag)
		if err != nil {
			return err
		}
		err = dag.CreateDependencies(helmChart, subParams)
		if err != nil {
			return err
		}
		if fargateProfile != nil {
			dag.AddDependency(helmChart, fargateProfile)
		}
		a.MapResourceDirectlyToConstruct(helmChart, unit)
	default:
		return fmt.Errorf("unsupported execution unit type %s", a.Config.GetExecutionUnit(unit.ID).Type)
	}
	return nil
}

func (a *AWS) handleHelmChartAwsValues(chart *kubernetes.HelmChart, unit *core.ExecutionUnit, dag *core.ResourceGraph) (valueParams map[string]any, err error) {
	valueParams = make(map[string]any)
	for _, val := range chart.ProviderValues {
		if val.ExecUnitName != unit.ID {
			continue
		}
		params := config.ConvertFromInfraParams[config.KubernetesTypeParams](a.Config.GetExecutionUnit(unit.ID).InfraParams)
		clusterName := params.ClusterId
		if clusterName == "" {
			clusterName = "cluster"
		}
		switch kubernetes.ProviderValueTypes(val.Type) {
		case kubernetes.ImageTransformation:
			chart.Values[val.Key] = core.IaCValue{
				Resource: &resources.EcrImage{},
				Property: resources.ECR_IMAGE_NAME_IAC_VALUE,
			}
			valueParams[val.Key] = resources.ImageCreateParams{
				AppName: a.Config.AppName,
				Refs:    core.AnnotationKeySetOf(unit.AnnotationKey),
				Name:    unit.ID,
			}
		case kubernetes.ServiceAccountAnnotationTransformation:
			chart.Values[val.Key] = core.IaCValue{
				Resource: &resources.IamRole{},
				Property: resources.ARN_IAC_VALUE,
			}
			valueParams[val.Key] = resources.RoleCreateParams{
				Name:    fmt.Sprintf("%s-%s-ExecutionRole", a.Config.AppName, unit.ID),
				Refs:    core.AnnotationKeySetOf(unit.AnnotationKey),
				AppName: a.Config.AppName,
			}
		case kubernetes.InstanceTypeKey:
			chart.Values[val.Key] = core.IaCValue{
				Property: "eks.amazonaws.com/nodegroup",
			}
		case kubernetes.InstanceTypeValue:
			chart.Values[val.Key] = core.IaCValue{
				Resource: &resources.EksNodeGroup{},
				Property: resources.NODE_GROUP_NAME_IAC_VALUE,
			}
			valueParams[val.Key] = resources.EksNodeGroupCreateParams{
				InstanceType: params.InstanceType,
				NetworkType:  a.Config.GetExecutionUnit(unit.ID).NetworkPlacement,
				AppName:      a.Config.AppName,
				ClusterName:  clusterName,
				Refs:         core.AnnotationKeySetOf(unit.AnnotationKey),
			}
		case kubernetes.TargetGroupTransformation:
			chart.Values[val.Key] = core.IaCValue{
				Resource: &resources.TargetGroup{},
				Property: resources.ARN_IAC_VALUE,
			}
			valueParams[val.Key] = resources.TargetGroupCreateParams{
				AppName: a.Config.AppName,
				Refs:    core.AnnotationKeySetOf(unit.AnnotationKey),
				Name:    unit.ID,
			}
		}
	}
	return
}

func (a *AWS) getLambdaConfiguration(result *core.ConstructGraph, dag *core.ResourceGraph, refs core.AnnotationKeySet) (resources.LambdaFunctionConfigureParams, error) {
	ref, oneRef := refs.GetSingle()
	if !oneRef {
		return resources.LambdaFunctionConfigureParams{}, fmt.Errorf("lambda must only have one construct reference")
	}
	lambdaConfig := resources.LambdaFunctionConfigureParams{}
	construct := result.GetConstruct(ref.ToId())
	if construct == nil {
		return resources.LambdaFunctionConfigureParams{}, fmt.Errorf("construct with id %s does not exist", ref.ToId())
	}
	unit, ok := construct.(*core.ExecutionUnit)
	if !ok {
		return resources.LambdaFunctionConfigureParams{}, fmt.Errorf("lambda must only have a construct reference to an execution unit")
	}
	for _, env := range unit.EnvironmentVariables {
		if env.Construct == nil {
			lambdaConfig.EnvironmentVariables = append(lambdaConfig.EnvironmentVariables, env)
		}
	}
	cfg := config.ConvertFromInfraParams[config.ServerlessTypeParams](a.Config.GetExecutionUnit(ref.ID).InfraParams)
	lambdaConfig.MemorySize = cfg.Memory
	lambdaConfig.Timeout = cfg.Timeout
	return lambdaConfig, nil
}

func (a *AWS) getImageConfiguration(result *core.ConstructGraph, dag *core.ResourceGraph, refs core.AnnotationKeySet) (resources.EcrImageConfigureParams, error) {
	ref, oneRef := refs.GetSingle()
	if !oneRef {
		return resources.EcrImageConfigureParams{}, fmt.Errorf("image must only have one construct reference but got %d: %v", len(refs), refs)
	}
	imageConfig := resources.EcrImageConfigureParams{}
	construct := result.GetConstruct(ref.ToId())
	if construct == nil {
		return resources.EcrImageConfigureParams{}, fmt.Errorf("construct with id %s does not exist", ref.ToId())
	}
	unit, ok := construct.(*core.ExecutionUnit)
	if !ok {
		return resources.EcrImageConfigureParams{}, fmt.Errorf("image must only have a construct reference to an execution unit ExecutionUnit but got %T", construct)
	}
	imageConfig.Context = fmt.Sprintf("./%s", unit.ID)
	imageConfig.Dockerfile = fmt.Sprintf("./%s/%s", unit.ID, unit.DockerfilePath)
	return imageConfig, nil
}

func (a *AWS) getNodeGroupConfiguration(result *core.ConstructGraph, dag *core.ResourceGraph, refs core.AnnotationKeySet) (resources.EksNodeGroupConfigureParams, error) {
	nodeGroupConfig := resources.EksNodeGroupConfigureParams{}
	nodeGroupConfig.DiskSize = 20
	for ref := range refs {
		construct := result.GetConstruct(ref.ToId())
		unit, ok := construct.(*core.ExecutionUnit)
		if !ok {
			continue
		}
		cfg := config.ConvertFromInfraParams[config.KubernetesTypeParams](a.Config.GetExecutionUnit(unit.ID).InfraParams)

		if nodeGroupConfig.DiskSize < cfg.DiskSizeGiB {
			nodeGroupConfig.DiskSize = cfg.DiskSizeGiB
		}
	}
	return nodeGroupConfig, nil
}

// handleEksProxy creates the necessary dependencies and resources for pods within the same helm chart to be able to use cloudmap to communicate
func (a *AWS) handleEksProxy(source, dest *core.ExecutionUnit, chart *kubernetes.HelmChart, dag *core.ResourceGraph) error {
	refs := core.AnnotationKeySetOf(source.AnnotationKey, dest.AnnotationKey)
	privateDnsNamespace, err := core.CreateResource[*resources.PrivateDnsNamespace](dag, resources.PrivateDnsNamespaceCreateParams{
		Refs:    refs,
		AppName: a.Config.AppName,
	})
	if err != nil {
		return err
	}
	dag.AddDependency(chart, privateDnsNamespace)
	unitsRole := knowledgebase.GetRoleForUnit(chart, source.AnnotationKey)
	if unitsRole == nil {
		return fmt.Errorf("no role found for chart %s and source reference %s", chart.Id(), source.Id())
	}
	role := dag.GetResource(unitsRole.Id())
	if role == nil {
		return fmt.Errorf("no role found for chart %s and source reference %s", chart.Id(), source.Id())
	}
	policy, err := core.CreateResource[*resources.IamPolicy](dag, resources.IamPolicyCreateParams{
		AppName: a.Config.AppName,
		Name:    "servicediscovery",
		Refs:    refs,
	})
	if err != nil {
		return err
	}
	dag.AddDependency(policy, privateDnsNamespace)
	dag.AddDependency(role, policy)
	return err
}

func findUnitsHelmChart(unit *core.ExecutionUnit, dag *core.ResourceGraph) (*kubernetes.HelmChart, error) {
	for _, res := range dag.ListResources() {
		if r, ok := res.(*kubernetes.HelmChart); ok {
			for _, ref := range r.ExecutionUnits {
				if ref.Name == unit.Provenance().ID {
					return r, nil
				}
			}
		}
	}
	return nil, fmt.Errorf("helm chart not found for unit with id, %s", unit.ID)
}
