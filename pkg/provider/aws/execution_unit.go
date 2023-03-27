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

		lambdaFunction := lambda.NewLambdaFunction(unit, a.Config.AppName, role)
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

func (a *AWS) convertExecUnitParams(unit *core.ExecutionUnit, dag *core.ResourceGraph) resources.EnvironmentVariables {
	resourceEnvVars := make(resources.EnvironmentVariables)
	for _, envVar := range unit.EnvironmentVariables {
		resource := a.ConstructIdToResourceId[envVar.GetConstruct().Id()]
		resourceEnvVars[envVar.Name] = resources.EnvironmentVariable{
			Resource: dag.GetResource(resource),
			Value:    envVar.Value,
		}
	}
	return resourceEnvVars
}

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
