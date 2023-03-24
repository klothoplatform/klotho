package lambda

import (
	"fmt"

	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/provider/aws/resources"
	"github.com/klothoplatform/klotho/pkg/provider/aws/resources/iam"
	"github.com/klothoplatform/klotho/pkg/sanitization/aws"
)

const LAMBDA_FUNCTION_TYPE = "lambda_function"

var sanitizer = aws.LambdaFunctionSanitizer

type (
	LambdaFunction struct {
		Name          string
		ConstructsRef []core.AnnotationKey
		// Role points to the id of the cloud resource
		Role      *iam.IamRole
		VpcConfig LambdaVpcConfig
	}

	LambdaVpcConfig struct {
		SecurityGroupIds []string
		SubnetIds        []string
	}
)

func NewLambdaFunction(unit *core.ExecutionUnit, appName string, role *iam.IamRole) *LambdaFunction {
	return &LambdaFunction{
		Name:          sanitizer.Apply(fmt.Sprintf("%s-%s", appName, unit.ID)),
		ConstructsRef: []core.AnnotationKey{unit.Provenance()},
		Role:          role,
	}
}

// Provider returns name of the provider the resource is correlated to
func (lambda *LambdaFunction) Provider() string {
	return resources.AWS_PROVIDER
}

// KlothoResource returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (lambda *LambdaFunction) KlothoConstructRef() []core.AnnotationKey {
	return lambda.ConstructsRef
}

// ID returns the id of the cloud resource
func (lambda *LambdaFunction) Id() string {
	return fmt.Sprintf("%s:%s:%s", lambda.Provider(), LAMBDA_FUNCTION_TYPE, lambda.Name)
}
