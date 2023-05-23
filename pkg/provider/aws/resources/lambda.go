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
	AppName string
	Refs    []core.AnnotationKey
	Name    string
}

func (lambda *LambdaFunction) Create(dag *core.ResourceGraph, params LambdaCreateParams) error {

	name := lambdaFunctionSanitizer.Apply(fmt.Sprintf("%s-%s", params.AppName, params.Name))
	lambda.Name = name
	lambda.ConstructsRef = params.Refs

	existingLambda := dag.GetResourceByVertexId(lambda.Id().String())
	if existingLambda != nil {
		return fmt.Errorf("lambda with name %s already exists", name)
	}

	logGroup, err := core.CreateResource[*LogGroup](dag, params)
	if err != nil {
		return err
	}
	dag.AddDependency(lambda, logGroup)
	subParams := map[string]any{
		"Role": RoleCreateParams{
			AppName: params.AppName,
			Name:    fmt.Sprintf("%s-ExecutionRole", params.Name),
			Refs:    lambda.ConstructsRef,
		},
		"Image": params,
	}

	err = dag.CreateDependencies(lambda, subParams)
	if err != nil {
		return err
	}
	return nil
}

type LambdaFunctionConfigureParams struct {
	Timeout              int
	MemorySize           int
	EnvironmentVariables core.EnvironmentVariables
}

func (lambda *LambdaFunction) Configure(params LambdaFunctionConfigureParams) error {
	lambda.Timeout = 180
	lambda.MemorySize = 512
	if lambda.EnvironmentVariables == nil {
		lambda.EnvironmentVariables = make(EnvironmentVariables)
	}

	if params.Timeout != 0 {
		lambda.Timeout = params.Timeout
	}
	if params.MemorySize != 0 {
		lambda.MemorySize = params.MemorySize
	}
	for _, env := range params.EnvironmentVariables {
		lambda.EnvironmentVariables[env.GetName()] = core.IaCValue{Property: env.GetValue()}
	}

	return nil
}

type LambdaPermissionCreateParams struct {
	AppName string
	Refs    []core.AnnotationKey
	Name    string
}

func (permission *LambdaPermission) Create(dag *core.ResourceGraph, params LambdaPermissionCreateParams) error {

	permission.Name = LambdaPermissionSanitizer.Apply(fmt.Sprintf("%s-%s", params.AppName, params.Name))
	if params.AppName == "" {
		permission.Name = LambdaPermissionSanitizer.Apply(params.Name)
	}
	permission.ConstructsRef = params.Refs

	existingLambdaPermission := dag.GetResource(permission.Id())
	if existingLambdaPermission != nil {
		graphLambdaPermission := existingLambdaPermission.(*LambdaPermission)
		graphLambdaPermission.ConstructsRef = core.DedupeAnnotationKeys(append(graphLambdaPermission.ConstructsRef, params.Refs...))
		return nil
	}
	dag.AddResource(permission)
	return nil
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
