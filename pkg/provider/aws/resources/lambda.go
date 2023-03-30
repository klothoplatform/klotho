package resources

import (
	"fmt"

	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/sanitization/aws"
)

const LAMBDA_FUNCTION_TYPE = "lambda_function"

var lambdaFunctionSanitizer = aws.LambdaFunctionSanitizer

type (
	LambdaFunction struct {
		Name          string
		ConstructsRef []core.AnnotationKey
		// Role points to the id of the cloud resource
		Role                 *IamRole
		VpcConfig            LambdaVpcConfig
		Image                *EcrImage
		EnvironmentVariables EnvironmentVariables
	}

	LambdaVpcConfig struct {
		SecurityGroupIds []string
		SubnetIds        []string
	}
)

func NewLambdaFunction(unit *core.ExecutionUnit, appName string, role *IamRole, image *EcrImage) *LambdaFunction {
	return &LambdaFunction{
		Name:          lambdaFunctionSanitizer.Apply(fmt.Sprintf("%s-%s", appName, unit.ID)),
		ConstructsRef: []core.AnnotationKey{unit.Provenance()},
		Role:          role,
		Image:         image,
	}
}

// Provider returns name of the provider the resource is correlated to
func (lambda *LambdaFunction) Provider() string {
	return AWS_PROVIDER
}

// KlothoResource returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (lambda *LambdaFunction) KlothoConstructRef() []core.AnnotationKey {
	return lambda.ConstructsRef
}

// ID returns the id of the cloud resource
func (lambda *LambdaFunction) Id() string {
	return fmt.Sprintf("%s:%s:%s", lambda.Provider(), LAMBDA_FUNCTION_TYPE, lambda.Name)
}
