package resources

import (
	"fmt"

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
		ConstructsRef        core.BaseConstructSet `yaml:"-"`
		Role                 *IamRole
		Image                *EcrImage
		EnvironmentVariables map[string]*AwsResourceValue `yaml:"-"`
		SecurityGroups       []*SecurityGroup
		Subnets              []*Subnet
		Timeout              int
		MemorySize           int
	}

	LambdaPermission struct {
		Name          string
		ConstructsRef core.BaseConstructSet `yaml:"-"`
		Function      *LambdaFunction
		Principal     string
		Source        *AwsResourceValue
		Action        string
	}
)

type LambdaCreateParams struct {
	AppName string
	Refs    core.BaseConstructSet
	Name    string
}

func (lambda *LambdaFunction) Create(dag *core.ResourceGraph, params LambdaCreateParams) error {

	name := lambdaFunctionSanitizer.Apply(fmt.Sprintf("%s-%s", params.AppName, params.Name))
	lambda.Name = name
	lambda.ConstructsRef = params.Refs.Clone()

	existingLambda := dag.GetResource(lambda.Id())
	if existingLambda != nil {
		return fmt.Errorf("lambda with name %s already exists", name)
	}

	logGroup, err := core.CreateResource[*LogGroup](dag, params)
	if err != nil {
		return err
	}
	dag.AddDependency(lambda, logGroup)
	return nil
}

func (lambda *LambdaFunction) MakeOperational(dag *core.ResourceGraph, appName string) error {
	if lambda.Role == nil {
		roles := core.GetDownstreamResourcesOfType[*IamRole](dag, lambda)
		if len(roles) == 0 {
			err := dag.CreateDependencies(lambda, map[string]any{
				"Role": RoleCreateParams{
					AppName: appName,
					Name:    fmt.Sprintf("%s-ExecutionRole", lambda.Name),
					Refs:    core.BaseConstructSetOf(lambda),
				},
			})
			if err != nil {
				return err
			}
		} else if len(roles) == 1 {
			lambda.Role = roles[0]
		} else {
			return fmt.Errorf("lambda %s has more than one role downstream", lambda.Id())
		}
	}

	if lambda.Image == nil {
		images := core.GetDownstreamResourcesOfType[*EcrImage](dag, lambda)
		if len(images) == 0 {
			err := dag.CreateDependencies(lambda, map[string]any{
				"Image": ImageCreateParams{
					AppName: appName,
					Name:    lambda.Name,
					Refs:    core.BaseConstructSetOf(lambda),
				},
			})
			if err != nil {
				return err
			}
		} else if len(images) == 1 {
			lambda.Image = images[0]
		} else {
			return fmt.Errorf("lambda %s has more than one role downstream", lambda.Id())
		}
	}
	dag.AddDependenciesReflect(lambda)
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
		lambda.EnvironmentVariables = make(map[string]*AwsResourceValue)
	}

	if params.Timeout != 0 {
		lambda.Timeout = params.Timeout
	}
	if params.MemorySize != 0 {
		lambda.MemorySize = params.MemorySize
	}
	for _, env := range params.EnvironmentVariables {
		lambda.EnvironmentVariables[env.GetName()] = &AwsResourceValue{PropertyVal: env.GetValue()}
	}

	return nil
}

type LambdaPermissionCreateParams struct {
	AppName string
	Refs    core.BaseConstructSet
	Name    string
}

func (permission *LambdaPermission) Create(dag *core.ResourceGraph, params LambdaPermissionCreateParams) error {

	permission.Name = LambdaPermissionSanitizer.Apply(fmt.Sprintf("%s-%s", params.AppName, params.Name))
	if params.AppName == "" {
		permission.Name = LambdaPermissionSanitizer.Apply(params.Name)
	}
	permission.ConstructsRef = params.Refs.Clone()

	existingLambdaPermission := dag.GetResource(permission.Id())
	if existingLambdaPermission != nil {
		graphLambdaPermission := existingLambdaPermission.(*LambdaPermission)
		graphLambdaPermission.ConstructsRef.AddAll(params.Refs)
		return nil
	}
	dag.AddResource(permission)
	return nil
}
func (permission *LambdaPermission) MakeOperational(dag *core.ResourceGraph, appName string) error {
	if permission.Function == nil {
		functions := core.GetDownstreamResourcesOfType[*LambdaFunction](dag, permission)
		if len(functions) == 0 {
			return fmt.Errorf("lambda permission %s has no lambda function downstream", permission.Id())
		} else if len(functions) > 1 {
			return fmt.Errorf("lambda permission %s has more than one lambda function downstream", permission.Id())
		}
		permission.Function = functions[0]
		dag.AddDependenciesReflect(permission)
	}
	return nil
}

// BaseConstructsRef returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (lambda *LambdaFunction) BaseConstructsRef() core.BaseConstructSet {
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

func (lambda *LambdaFunction) DeleteContext() core.DeleteContext {
	return core.DeleteContext{
		RequiresNoUpstream:     true,
		RequiresNoDownstream:   true,
		RequiresExplicitDelete: true,
	}
}

func (lambda *LambdaFunction) GetFunctionality() core.Functionality {
	return core.Compute
}

// BaseConstructsRef returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (permission *LambdaPermission) BaseConstructsRef() core.BaseConstructSet {
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

func (permission *LambdaPermission) DeleteContext() core.DeleteContext {
	return core.DeleteContext{
		RequiresNoUpstream: true,
	}
}
