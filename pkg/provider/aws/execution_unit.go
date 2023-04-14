package aws

import (
	"fmt"

	"github.com/klothoplatform/klotho/pkg/config"
	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/infra/kubernetes"
	"github.com/klothoplatform/klotho/pkg/provider/aws/resources"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

// GenerateExecUnitResources generates the necessary AWS resources for a given execution unit and adds them to the resource graph
func (a *AWS) GenerateExecUnitResources(unit *core.ExecutionUnit, result *core.ConstructGraph, dag *core.ResourceGraph) error {
	log := zap.S()

	execUnitCfg := a.Config.GetExecutionUnit(unit.ID)

	image, err := resources.GenerateEcrRepoAndImage(a.Config.AppName, unit, dag)
	if err != nil {
		return err
	}

	role := resources.NewIamRole(a.Config.AppName, fmt.Sprintf("%s-ExecutionRole", unit.ID), []core.AnnotationKey{unit.Provenance()}, GetAssumeRolePolicyForType(execUnitCfg))
	dag.AddResource(role)
	err = a.PolicyGenerator.AddUnitRole(unit.Id(), role)
	if err != nil {
		return err
	}
	for _, construct := range result.GetDownstreamConstructs(unit) {
		resList, ok := a.GetResourcesDirectlyTiedToConstruct(construct)
		if !ok {
			return errors.Errorf("could not find resource for construct, %s, which unit, %s, depends on", construct.Id(), unit.Id())
		}
		for _, resource := range resList {
			dag.AddDependency(role, resource)
		}
	}
	unitsPolicies := a.PolicyGenerator.GetUnitPolicies(unit.Id())
	for _, pol := range unitsPolicies {
		dag.AddDependency(role, pol)
		role.ManagedPolicies = append(role.ManagedPolicies, core.IaCValue{
			Resource: pol,
			Property: resources.ARN_IAC_VALUE,
		})
	}

	switch execUnitCfg.Type {
	case Lambda:
		role.AwsManagedPolicies = append(role.AwsManagedPolicies, "arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole")
		role.AwsManagedPolicies = append(role.AwsManagedPolicies, "arn:aws:iam::aws:policy/AWSXRayDaemonWriteAccess")
		lambdaFunction := resources.NewLambdaFunction(unit, a.Config.AppName, role, image)
		dag.AddDependenciesReflect(lambdaFunction)
		a.MapResourceDirectlyToConstruct(lambdaFunction, unit)

		logGroup := resources.NewLogGroup(a.Config.AppName, fmt.Sprintf("/aws/lambda/%s", lambdaFunction.Name), unit.Provenance(), 5)
		dag.AddResource(logGroup)
		dag.AddDependency(lambdaFunction, logGroup)
		return nil
	case kubernetes.KubernetesType:
		cfg := a.Config.GetExecutionUnit(unit.Provenance().ID)
		params := cfg.GetExecutionUnitParamsAsKubernetes()
		cluster := resources.GetEksCluster(a.Config.AppName, params.ClusterId, dag)
		if cluster == nil {
			return errors.Errorf("Expected to have cluster created for unit, %s, but did not find cluster in graph", unit.ID)
		}
		role.AssumeRolePolicyDoc = &resources.PolicyDocument{
			Version: resources.VERSION,
			Statement: []resources.StatementEntry{
				{
					Effect: "Allow",
					Principal: &resources.Principal{
						Federated: core.IaCValue{
							Resource: cluster,
							Property: resources.CLUSTER_OIDC_ARN_IAC_VALUE,
						},
					},
					Action: []string{"sts:AssumeRoleWithWebIdentity"},
					Condition: &resources.Condition{
						StringEquals: map[core.IaCValue]string{
							{
								Resource: cluster,
								Property: resources.CLUSTER_OIDC_URL_IAC_VALUE,
							}: fmt.Sprintf("system:serviceaccount:default:%s", unit.ID), // TODO: Replace default with the namespace when we expose via configuration
						},
					},
				},
			},
		}
		// transform kubernetes resources for EKS
		for _, res := range dag.ListResources() {
			if khChart, ok := res.(*kubernetes.HelmChart); ok {
				for _, ref := range khChart.KlothoConstructRef() {
					if ref.ToId() == unit.ToId() {
						khChart.ClustersProvider = core.IaCValue{
							Resource: cluster,
							Property: resources.CLUSTER_PROVIDER_IAC_VALUE,
						}
						dag.AddDependenciesReflect(khChart)
						for _, val := range khChart.ProviderValues {
							if val.ExecUnitName != unit.ID {
								continue
							}
							switch kubernetes.ProviderValueTypes(val.Type) {
							case kubernetes.ImageTransformation:
								khChart.Values[val.Key] = core.IaCValue{
									Resource: image,
									Property: resources.ECR_IMAGE_NAME_IAC_VALUE,
								}
								dag.AddDependency(khChart, image)
							case kubernetes.ServiceAccountAnnotationTransformation:
								khChart.Values[val.Key] = core.IaCValue{
									Resource: role,
									Property: resources.ARN_IAC_VALUE,
								}
								dag.AddDependency(khChart, role)
							case kubernetes.InstanceTypeKey:
								khChart.Values[val.Key] = core.IaCValue{
									Property: "eks.amazonaws.com/nodegroup",
								}
							case kubernetes.InstanceTypeValue:
								khChart.Values[val.Key] = core.IaCValue{
									Property: resources.NodeGroupNameFromConfig(cfg),
								}
							case kubernetes.TargetGroupTransformation:
								targetGroup := a.createEksLoadBalancer(result, dag, unit)
								khChart.Values[val.Key] = core.IaCValue{
									Resource: targetGroup,
									Property: resources.ARN_IAC_VALUE,
								}
								dag.AddDependency(khChart, targetGroup)
							}
						}
						a.MapResourceDirectlyToConstruct(khChart, unit)
					}
				}
			}
		}
	default:
		log.Errorf("Unsupported type, %s, for aws execution units", execUnitCfg.Type)

	}
	return nil
}

func (a *AWS) handleExecUnitProxy(result *core.ConstructGraph, dag *core.ResourceGraph) error {
	for _, unit := range core.GetResourcesOfType[*core.ExecutionUnit](result) {

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
						execPolicyDoc := resources.CreateAllowPolicyDocument([]string{"lambda:InvokeFunction"}, []core.IaCValue{{Resource: targetLambda, Property: resources.ARN_IAC_VALUE}})
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
					execPolicy.ConstructsRef = append(execPolicy.ConstructsRef, unit.AnnotationKey)
					dag.AddDependency(execPolicy, a.PolicyGenerator.GetUnitRole(unit.Id()))
					dag.AddDependency(execPolicy, targetLambda)
				case kubernetes.KubernetesType:
					privateNamespace := resources.NewPrivateDnsNamespace(a.Config.AppName, []core.AnnotationKey{unit.AnnotationKey}, resources.GetVpc(a.Config, dag))
					if ns := dag.GetResource(privateNamespace.Id()); ns != nil {
						namespace, ok := ns.(*resources.PrivateDnsNamespace)
						if !ok {
							return errors.Errorf("Found a non PrivateDnsNamespace with same id as global PrivateDnsNamespace, %s", namespace.Id())
						}
						privateNamespace = namespace
						privateNamespace.ConstructsRef = append(privateNamespace.ConstructsRef, unit.Provenance())
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
					policy := resources.NewIamPolicy(a.Config.AppName, privateNamespace.Name, unit.AnnotationKey, serviceDiscoveryPolicyDoc)
					if pol := dag.GetResource(policy.Id()); pol != nil {
						pol, ok := pol.(*resources.IamPolicy)
						if !ok {
							return errors.Errorf("Found a non PrivateDnsNamespace with same id as global PrivateDnsNamespace, %s", pol.Id())
						}
						policy = pol
						policy.ConstructsRef = append(policy.ConstructsRef, unit.Provenance())
					} else {
						dag.AddDependenciesReflect(policy)
					}
					a.PolicyGenerator.AddAllowPolicyToUnit(unit.ID, policy)
				}
			}
		}

	}

	return nil
}

// convertExecUnitParams transforms the execution units environment variables to a map of key names and their corresponding core.IaCValue struct.
//
// If an environment variable does not pertain to a construct and is just a key, value string, the resource of the IaCValue will be left null.
func (a *AWS) convertExecUnitParams(result *core.ConstructGraph, dag *core.ResourceGraph) error {
	execUnits := core.GetResourcesOfType[*core.ExecutionUnit](result)
	for _, unit := range execUnits {

		resourceEnvVars := make(resources.EnvironmentVariables)

		// This set of environment variables correspond to the specific needs of the execution units and its dependencies
		for _, envVar := range unit.EnvironmentVariables {
			if envVar.Construct != nil {
				resList, ok := a.GetResourcesDirectlyTiedToConstruct(envVar.GetConstruct())
				if !ok {
					return fmt.Errorf("resource not found for construct with id, %s", envVar.GetConstruct().Id())
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

		// This set of environment variables are added to all Execution Unit's corresponding Resources
		resourceEnvVars["APP_NAME"] = core.IaCValue{
			Property: a.Config.AppName,
		}
		resourceEnvVars["EXECUNIT_NAME"] = core.IaCValue{
			Property: unit.ID,
		}

		// Retrieve the actual resource and set the environment variables on it
		resList, ok := a.GetResourcesDirectlyTiedToConstruct(unit)
		if !ok {
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
							if evVal.Resource != resource {
								dag.AddDependency(resource, evVal.Resource)
							}
						}
					}

				}
			}
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

func (a *AWS) createEksLoadBalancer(result *core.ConstructGraph, dag *core.ResourceGraph, unit *core.ExecutionUnit) *resources.TargetGroup {
	gws := result.FindUpstreamGateways(unit)
	refs := []core.AnnotationKey{unit.AnnotationKey}
	for _, gw := range gws {
		refs = append(refs, gw.AnnotationKey)
	}
	vpc := resources.GetVpc(a.Config, dag)
	subnets := vpc.GetPrivateSubnets(dag)
	securityGroups := []*resources.SecurityGroup{resources.GetSecurityGroup(a.Config, dag)}
	lb := resources.NewLoadBalancer(a.Config.AppName, unit.ID, refs, "internal", "network", subnets, securityGroups)
	unitsPort := unit.Port
	if unitsPort == 0 {
		unitsPort = 3000
	}
	targetGroup := resources.NewTargetGroup(a.Config.AppName, unit.ID, refs, unitsPort, "TCP", vpc, "ip")
	listener := resources.NewListener(unit.ID, lb, refs, 80, "TCP", []*resources.LBAction{
		{TargetGroupArn: core.IaCValue{Resource: targetGroup, Property: resources.ARN_IAC_VALUE}, Type: "forward"},
	})
	dag.AddDependenciesReflect(lb)
	dag.AddDependenciesReflect(targetGroup)
	dag.AddDependenciesReflect(listener)
	dag.AddDependency(listener, targetGroup)
	return targetGroup
}
