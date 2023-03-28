package aws

import (
	"fmt"

	"github.com/klothoplatform/klotho/pkg/config"
	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/provider/aws/resources"
	"github.com/klothoplatform/klotho/pkg/provider/aws/resources/cloudwatch"
	"github.com/klothoplatform/klotho/pkg/provider/aws/resources/ecr"
	"github.com/klothoplatform/klotho/pkg/provider/aws/resources/iam"
	"github.com/klothoplatform/klotho/pkg/provider/aws/resources/lambda"
	"go.uber.org/zap"
)

// GenerateExecUnitResources generates the neccessary AWS resources for a given execution unit and adds them to the resource graph
func (a *AWS) GenerateExecUnitResources(unit *core.ExecutionUnit, dag *core.ResourceGraph) error {
	log := zap.S()

	execUnitCfg := a.Config.GetExecutionUnit(unit.ID)

	image, err := ecr.GenerateEcrRepoAndImage(a.Config.AppName, unit, dag)
	if err != nil {
		return err
	}

	role := iam.NewIamRole(a.Config.AppName, fmt.Sprintf("%s-ExecutionRole", unit.ID), unit.Provenance(), GetAssumeRolePolicyForType(execUnitCfg))
	dag.AddResource(role)

	switch execUnitCfg.Type {
	case Lambda:

		lambdaFunction := lambda.NewLambdaFunction(unit, a.Config.AppName, role, image)
		a.ConstructIdToResourceId[unit.Id()] = lambdaFunction.Id()
		logGroup := cloudwatch.NewLogGroup(a.Config.AppName, fmt.Sprintf("/aws/lambda/%s", lambdaFunction.Name), unit.Provenance(), 5)
		dag.AddResource(lambdaFunction)
		dag.AddResource(logGroup)
		dag.AddDependency(logGroup, lambdaFunction)
		dag.AddDependency(role, lambdaFunction)
		dag.AddDependency(image, lambdaFunction)
		return nil
	default:
		log.Errorf("Unsupported type, %s, for aws execution units", execUnitCfg.Type)

	}
	return nil
}

// convertExecUnitParams transforms the execution units environment variables to a map of key names and their corresponding core.IaCValue struct
//
// If an environment variable does not pertain to a construct and is just a key, value string, the resource of the IaCValue will be left null
func (a *AWS) convertExecUnitParams(result *core.ConstructGraph, dag *core.ResourceGraph) error {
	execUnits := core.GetResourcesOfType[*core.ExecutionUnit](result)
	for _, unit := range execUnits {
		resourceEnvVars := make(resources.EnvironmentVariables)

		// This set of environment variables correspond to the specific needs of the execution units and its dependencies
		for _, envVar := range unit.EnvironmentVariables {
			if envVar.Construct != nil {
				resourceId, ok := a.ConstructIdToResourceId[envVar.GetConstruct().Id()]
				if ok {
					resource := dag.GetResource(resourceId)
					if resource == nil {
						return fmt.Errorf("resource not found for id, %s", resourceId)
					}
					resourceEnvVars[envVar.Name] = core.IaCValue{
						Resource: resource,
						Property: envVar.Value,
					}
				} else {
					return fmt.Errorf("resource not found for construct with id, %s", envVar.GetConstruct().Id())
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
		resourceId, ok := a.ConstructIdToResourceId[unit.Id()]
		if ok {
			resource := dag.GetResource(resourceId)
			if resource == nil {
				return fmt.Errorf("resource not found for id, %s", resourceId)
			}
			switch r := resource.(type) {
			case *lambda.LambdaFunction:
				r.EnvironmentVariables = resourceEnvVars
			}
		} else {
			return fmt.Errorf("resource not found for construct with id, %s", unit.Id())
		}
	}
	return nil
}

// GetAssumeRolePolicyForType returns an assume role policy doc as a string, for the execution units corresponding IAM role
func GetAssumeRolePolicyForType(cfg config.ExecutionUnit) string {
	switch cfg.Type {
	case Lambda:
		return iam.LAMBDA_ASSUMER_ROLE_POLICY
	case Ecs:
		return iam.ECS_ASSUMER_ROLE_POLICY
	case Eks:
		eksConfig := cfg.GetExecutionUnitParamsAsKubernetes()
		if eksConfig.NodeType == string(resources.Fargate) {
			return iam.EKS_FARGATE_ASSUME_ROLE_POLICY
		}
		return iam.EC2_ASSUMER_ROLE_POLICY
	}
	return ""
}
