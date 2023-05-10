package resources

import (
	"fmt"

	"github.com/klothoplatform/klotho/pkg/config"
	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/sanitization/aws"
)

const (
	LAMBDA_FUNCTION_TYPE   = "lambda_function"
	LAMBDA_PERMISSION_TYPE = "lambda_permission"
)

var lambdaFunctionSanitizer = aws.LambdaFunctionSanitizer
var LambdaPermissionSanitizer = aws.LambdaPermissionSanitizer

type (
	LambdaFunction struct {
		Name                 string
		ConstructsRef        []core.AnnotationKey
		Role                 *IamRole
		Image                *EcrImage
		EnvironmentVariables EnvironmentVariables
		SecurityGroups       []*SecurityGroup
		Subnets              []*Subnet
		Timeout              int
		MemorySize           int
	}

	LambdaPermission struct {
		Name          string
		ConstructsRef []core.AnnotationKey
		Function      *LambdaFunction
		Principal     string
		Source        core.IaCValue
		Action        string
	}
)

type LambdaCreateParams struct {
	AppName          string
	Unit             *core.ExecutionUnit
	Vpc              bool
	NetworkPlacement string
	Params           config.ServerlessTypeParams
}

func (lambda *LambdaFunction) Create(dag *core.ResourceGraph, params LambdaCreateParams) error {

	name := lambdaFunctionSanitizer.Apply(fmt.Sprintf("%s-%s", params.AppName, params.Unit.ID))
	lambda.Name = name
	lambda.ConstructsRef = []core.AnnotationKey{params.Unit.AnnotationKey}
	lambda.MemorySize = params.Params.Memory
	lambda.Timeout = params.Params.Timeout

	existingLambda := dag.GetResourceByVertexId(lambda.Id().String())
	if existingLambda != nil {
		return fmt.Errorf("lambda with name %s already exists", name)
	}

	subParams := map[string]any{
		"Role": RoleCreateParams{
			RoleName:            fmt.Sprintf("%s-ExecutionRole", lambda.Name),
			Refs:                lambda.ConstructsRef,
			AssumeRolePolicyDoc: LAMBDA_ASSUMER_ROLE_POLICY,
		},
		"Image": ImageCreateParams{
			AppName:        params.AppName,
			Refs:           lambda.ConstructsRef,
			Unit:           params.Unit.ID,
			DockerfilePath: params.Unit.DockerfilePath,
		},
	}

	if params.Vpc {
		subnetType := PrivateSubnet
		if params.NetworkPlacement == "public" {
			subnetType = PublicSubnet
		}
		lambda.Subnets = make([]*Subnet, 2)
		lambda.SecurityGroups = make([]*SecurityGroup, 1)

		subParams["Subnets"] = []SubnetCreateParams{
			{
				AppName: params.AppName,
				Refs:    lambda.ConstructsRef,
				AZ:      "0",
				Type:    subnetType,
			},
			{
				AppName: params.AppName,
				Refs:    lambda.ConstructsRef,
				AZ:      "1",
				Type:    subnetType,
			},
		}
		subParams["SecurityGroups"] = []SecurityGroupCreateParams{
			{
				AppName: params.AppName,
				Refs:    lambda.ConstructsRef,
			},
		}
	}

	err := dag.CreateDependencies(lambda, subParams)
	return err
}

func (lambda *LambdaPermission) Create(dag *core.ResourceGraph, metadata map[string]any) (core.Resource, error) {
	panic("Not Implemented")
}

func NewLambdaFunction(unit *core.ExecutionUnit, cfg *config.Application, role *IamRole, image *EcrImage) *LambdaFunction {
	params := config.ConvertFromInfraParams[config.ServerlessTypeParams](cfg.GetExecutionUnit(unit.ID).InfraParams)
	return &LambdaFunction{
		Name:          lambdaFunctionSanitizer.Apply(fmt.Sprintf("%s-%s", cfg.AppName, unit.ID)),
		ConstructsRef: []core.AnnotationKey{unit.Provenance()},
		Role:          role,
		Image:         image,
		MemorySize:    params.Memory,
		Timeout:       params.Timeout,
	}
}

// KlothoConstructRef returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (lambda *LambdaFunction) KlothoConstructRef() []core.AnnotationKey {
	return lambda.ConstructsRef
}

// Id returns the id of the cloud resource
func (lambda *LambdaFunction) Id() core.ResourceId {
	return core.ResourceId{
		Provider: AWS_PROVIDER,
		Type:     LAMBDA_FUNCTION_TYPE,
		Name:     lambda.Name,
	}
}

func NewLambdaPermission(function *LambdaFunction, source core.IaCValue, principal string, action string, ref []core.AnnotationKey) *LambdaPermission {
	return &LambdaPermission{
		Name:          LambdaPermissionSanitizer.Apply(fmt.Sprintf("%s-%s", function.Name, source.Resource.Id())),
		ConstructsRef: ref,
		Function:      function,
		Source:        source,
		Action:        action,
		Principal:     principal,
	}
}

// KlothoConstructRef returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (permission *LambdaPermission) KlothoConstructRef() []core.AnnotationKey {
	return permission.ConstructsRef
}

// Id returns the id of the cloud resource
func (permission *LambdaPermission) Id() core.ResourceId {
	return core.ResourceId{
		Provider: AWS_PROVIDER,
		Type:     LAMBDA_PERMISSION_TYPE,
		Name:     permission.Name,
	}
}
