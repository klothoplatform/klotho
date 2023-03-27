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
		lambda := lambda.NewLambdaFunction(unit, a.Config.AppName, role, image)
		logGroup := cloudwatch.NewLogGroup(a.Config.AppName, fmt.Sprintf("/aws/lambda/%s", lambda.Name), unit.Provenance(), 5)
		dag.AddResource(lambda)
		dag.AddResource(logGroup)
		dag.AddDependency(logGroup, lambda)
		dag.AddDependency(role, lambda)
		dag.AddDependency(image, lambda)
		return nil
	default:
		log.Errorf("Unsupported type, %s, for aws execution units", execUnitCfg.Type)

	}
	return nil
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
