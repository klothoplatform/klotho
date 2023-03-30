package aws

import (
	"fmt"

	"github.com/klothoplatform/klotho/pkg/config"
	"github.com/klothoplatform/klotho/pkg/core"
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
			return errors.Errorf("could not find resource for construct, %s, which unit, %s, depends on", unit.Id(), construct.Id())
		}
		for _, resource := range resList {
			dag.AddDependency2(role, resource)
		}
	}
	role.InlinePolicy = a.PolicyGenerator.GetUnitPolicies(unit.Id())

	switch execUnitCfg.Type {
	case Lambda:

		lambdaFunction := resources.NewLambdaFunction(unit, a.Config.AppName, role, image)
		a.MapResourceDirectlyToConstruct(lambdaFunction, unit)
		logGroup := resources.NewLogGroup(a.Config.AppName, fmt.Sprintf("/aws/lambda/%s", lambdaFunction.Name), unit.Provenance(), 5)
		dag.AddResource(lambdaFunction)
		dag.AddResource(logGroup)
		dag.AddDependency2(lambdaFunction, logGroup)
		dag.AddDependency2(lambdaFunction, role)
		dag.AddDependency2(lambdaFunction, image)
		return nil
	case Kubernetes:
		return nil
	default:
		log.Errorf("Unsupported type, %s, for aws execution units", execUnitCfg.Type)

	}
	return nil
}

// convertExecUnitParams transforms the execution units environment variables to a map of key names and their corresponding core.IaCValue struct.
//
// If an environment variable does not pertain to a construct and is just a key, value string, the resource of the IaCValue will be left null.
func (a *AWS) convertExecUnitParams(result *core.ConstructGraph, dag *core.ResourceGraph) error {
	execUnits := core.GetResourcesOfType[*core.ExecutionUnit](result)
	for _, unit := range execUnits {

		if a.Config.GetExecutionUnit(unit.ID).Type == Kubernetes {
			continue
		}
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
			}
		}
	}
	return nil
}

// GetAssumeRolePolicyForType returns an assume role policy doc as a string, for the execution units corresponding IAM role
func GetAssumeRolePolicyForType(cfg config.ExecutionUnit) string {
	switch cfg.Type {
	case Lambda:
		return resources.LAMBDA_ASSUMER_ROLE_POLICY
	case Ecs:
		return resources.ECS_ASSUMER_ROLE_POLICY
	case Kubernetes:
		eksConfig := cfg.GetExecutionUnitParamsAsKubernetes()
		if eksConfig.NodeType == string(resources.Fargate) {
			return resources.EKS_FARGATE_ASSUME_ROLE_POLICY
		}
		return resources.EC2_ASSUMER_ROLE_POLICY
	}
	return ""
}
