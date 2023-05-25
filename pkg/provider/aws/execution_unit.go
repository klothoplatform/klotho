package aws

import (
	"fmt"

	"github.com/klothoplatform/klotho/pkg/config"
	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/infra/kubernetes"
	"github.com/klothoplatform/klotho/pkg/provider/aws/resources"
	"github.com/pkg/errors"
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
		err = a.MapResourceToConstruct(lambda, unit)
		if err != nil {
			return err
		}
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
		err = a.MapResourceToConstruct(helmChart, unit)
		if err != nil {
			return err
		}
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
			return nodeGroupConfig, fmt.Errorf("node group must only have construct references to ExecutionUnits but got %T", construct)
		}
		cfg := config.ConvertFromInfraParams[config.KubernetesTypeParams](a.Config.GetExecutionUnit(unit.ID).InfraParams)

		if nodeGroupConfig.DiskSize < cfg.DiskSizeGiB {
			nodeGroupConfig.DiskSize = cfg.DiskSizeGiB
		}
	}
	return nodeGroupConfig, nil
}

func (a *AWS) handleExecUnitProxy(result *core.ConstructGraph, dag *core.ResourceGraph) error {
	for _, unit := range core.GetConstructsOfType[*core.ExecutionUnit](result) {

		downstreamConstructs := result.GetDownstreamConstructs(unit)
		for _, construct := range downstreamConstructs {
			if targetUnit, ok := construct.(*core.ExecutionUnit); ok {
				switch a.Config.GetExecutionUnit(targetUnit.ID).Type {
				case Lambda:
					targetResources, _ := a.GetResourcesDirectlyTiedToConstruct(targetUnit)
					var targetLambda *resources.LambdaFunction
					var execPolicy *resources.IamPolicy
					var execPolicyDoc *resources.PolicyDocument
					for _, resource := range targetResources {
						if lambdafunc, ok := resource.(*resources.LambdaFunction); ok {
							targetLambda = lambdafunc
						}
						execPolicyDoc = resources.CreateAllowPolicyDocument([]string{"lambda:InvokeFunction"}, []core.IaCValue{{Resource: targetLambda, Property: resources.ARN_IAC_VALUE}})
						if execPol, ok := resource.(*resources.IamPolicy); ok {
							if len(execPol.Policy.Statement) == 1 {
								statement := execPol.Policy.Statement[0]
								if statement.Action[0] == execPolicyDoc.Statement[0].Action[0] && statement.Resource[0] == execPolicyDoc.Statement[0].Resource[0] {
									execPolicy = execPol
								}
							}
						}
					}
					if targetLambda == nil {
						return errors.Errorf("Could not find a lambda function tied to execution unit %s", targetUnit.ID)
					}
					if execPolicy == nil {
						execPolicy = resources.NewIamPolicy(a.Config.AppName, fmt.Sprintf("%s-invoke", targetUnit.ID), targetUnit.Provenance(), execPolicyDoc)
						dag.AddResource(execPolicy)
					}
					// We do not add the policy to the units list in policy generator otherwise we will cause a circular dependency
					execPolicy.ConstructsRef.Add(unit.AnnotationKey)
					dag.AddDependency(a.PolicyGenerator.GetUnitRole(unit.Id()), execPolicy)
					dag.AddDependency(execPolicy, targetLambda)
				case kubernetes.KubernetesType:
					privateNamespace := resources.NewPrivateDnsNamespace(a.Config.AppName, core.AnnotationKeySetOf(unit.AnnotationKey), resources.GetVpc(a.Config, dag))
					if ns := dag.GetResource(privateNamespace.Id()); ns != nil {
						namespace, ok := ns.(*resources.PrivateDnsNamespace)
						if !ok {
							return errors.Errorf("Found a non PrivateDnsNamespace with same id as global PrivateDnsNamespace, %s", namespace.Id())
						}
						privateNamespace = namespace
						privateNamespace.ConstructsRef.Add(unit.Provenance())
					} else {
						dag.AddDependenciesReflect(privateNamespace)
					}

					// Add a dependency from clusters to the namespace so its available before any pods could come up
					for _, resource := range dag.ListResources() {
						if cluster, ok := resource.(*resources.EksCluster); ok {
							dag.AddDependency(cluster, privateNamespace)
						}
					}

					serviceDiscoveryPolicyDoc := resources.CreateAllowPolicyDocument([]string{"servicediscovery:DiscoverInstances"}, []core.IaCValue{{Property: core.ALL_RESOURCES_IAC_VALUE}})
					execPolicy := resources.NewIamPolicy(a.Config.AppName, fmt.Sprintf("%s-servicediscovery", privateNamespace.Name), unit.AnnotationKey, serviceDiscoveryPolicyDoc)
					dag.AddResource(execPolicy)
					execPolicy.ConstructsRef.Add(unit.AnnotationKey)
					dag.AddDependency(a.PolicyGenerator.GetUnitRole(unit.Id()), execPolicy)

					cluster, err := findUnitsCluster(targetUnit, dag)
					if err != nil {
						return err
					}
					cloudmap, err := cluster.InstallCloudMapController(targetUnit.AnnotationKey, dag)
					if err != nil {
						return err
					}
					klothoChart, err := findUnitsHelmChart(unit, dag)
					if err != nil {
						return err
					}
					dag.AddDependency(klothoChart, cloudmap)
				}
			}
		}
	}
	return nil
}

func findUnitsCluster(unit *core.ExecutionUnit, dag *core.ResourceGraph) (*resources.EksCluster, error) {
	for _, res := range dag.ListResources() {
		if r, ok := res.(*resources.EksCluster); ok {
			if r.ConstructsRef.Has(unit.Provenance()) {
				return r, nil
			}
		}
	}
	return nil, fmt.Errorf("eks cluster not found for unit with id, %s", unit.ID)
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

// convertExecUnitParams transforms the execution units environment variables to a map of key names and their corresponding core.IaCValue struct.
//
// If an environment variable does not pertain to a construct and is just a key, value string, the resource of the IaCValue will be left null.
func (a *AWS) convertExecUnitParams(result *core.ConstructGraph, dag *core.ResourceGraph) error {
	execUnits := core.GetConstructsOfType[*core.ExecutionUnit](result)
	for _, unit := range execUnits {

		resourceEnvVars := make(resources.EnvironmentVariables)

		// This set of environment variables correspond to the specific needs of the execution units and its dependencies
		for _, envVar := range unit.EnvironmentVariables {
			if envVar.Construct != nil {
				resList, ok := a.GetResourcesDirectlyTiedToConstruct(envVar.GetConstruct())
				if !ok {
					return fmt.Errorf("resource not found for env var construct with id, %s", envVar.GetConstruct().Id())
				}
				for _, resource := range resList {
					resourceEnvVars[envVar.Name] = core.IaCValue{
						Resource: resource,
						Property: envVar.Value,
					}

				}
			} else {
				resourceEnvVars[envVar.Name] = core.IaCValue{
					Property: envVar.Value,
				}
			}
		}

		// Retrieve the actual resource and set the environment variables on it
		resList, _ := a.GetResourcesDirectlyTiedToConstruct(unit)

		// If the unit is a kubernetes unit, the helm chart wont be directly tied to the unit so we need to ensure we grab it
		if a.Config.GetExecutionUnit(unit.ID).Type == kubernetes.KubernetesType {
			chart, err := findUnitsHelmChart(unit, dag)
			if err != nil {
				return err
			}
			resList = append(resList, chart)
		}
		if len(resList) == 0 {
			return fmt.Errorf("resource not found for construct with id, %s", unit.Id())
		}
		for _, resource := range resList {
			switch r := resource.(type) {
			case *resources.LambdaFunction:
				r.EnvironmentVariables = resourceEnvVars
			case *kubernetes.HelmChart:
				for evName, evVal := range resourceEnvVars {
					for _, val := range r.ProviderValues {
						if val.EnvironmentVariable != nil && evName == val.EnvironmentVariable.GetName() {
							r.Values[val.Key] = evVal
							if evVal.Resource != nil && evVal.Resource != resource {
								dag.AddDependency(resource, evVal.Resource)
							}
						}
					}
				}
			}
			dag.AddDependenciesReflect(resource)
		}
	}
	return nil
}

// GetAssumeRolePolicyForType returns an assume role policy doc as a string, for the execution units corresponding IAM role
func GetAssumeRolePolicyForType(cfg config.ExecutionUnit) *resources.PolicyDocument {
	switch cfg.Type {
	case Lambda:
		return resources.LAMBDA_ASSUMER_ROLE_POLICY
	case Ecs:
		return resources.ECS_ASSUMER_ROLE_POLICY
	case kubernetes.KubernetesType:
		eksConfig := cfg.GetExecutionUnitParamsAsKubernetes()
		if eksConfig.NodeType == string(resources.Fargate) {
			return resources.EKS_FARGATE_ASSUME_ROLE_POLICY
		}
		return resources.EC2_ASSUMER_ROLE_POLICY
	}
	return nil
}
