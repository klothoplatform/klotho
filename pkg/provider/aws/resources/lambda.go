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

// Provider returns name of the provider the resource is correlated to
func (lambda *LambdaFunction) Provider() string {
	return AWS_PROVIDER
}

// KlothoConstructRef returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (lambda *LambdaFunction) KlothoConstructRef() []core.AnnotationKey {
	return lambda.ConstructsRef
}

// Id returns the id of the cloud resource
func (lambda *LambdaFunction) Id() string {
	return fmt.Sprintf("%s:%s:%s", lambda.Provider(), LAMBDA_FUNCTION_TYPE, lambda.Name)
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

// Provider returns name of the provider the resource is correlated to
func (permission *LambdaPermission) Provider() string {
	return AWS_PROVIDER
}

// KlothoConstructRef returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (permission *LambdaPermission) KlothoConstructRef() []core.AnnotationKey {
	return permission.ConstructsRef
}

// Id returns the id of the cloud resource
func (permission *LambdaPermission) Id() string {
	return fmt.Sprintf("%s:%s:%s", permission.Provider(), LAMBDA_PERMISSION_TYPE, permission.Name)
}
