package lambda

import (
	"testing"

	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/provider/aws/resources"
	"github.com/klothoplatform/klotho/pkg/provider/aws/resources/iam"
	"github.com/stretchr/testify/assert"
)

func Test_NewLambdaFunction(t *testing.T) {
	assert := assert.New(t)
	eu := &core.ExecutionUnit{AnnotationKey: core.AnnotationKey{ID: "somelongWackyId%^&**thatsgoingtobewayoverthecharcountificontinuetotypethingsouthere"}}
	role := &iam.IamRole{Name: "testRole"}
	lambda := NewLambdaFunction(eu, "test-app", role)
	assert.Equal(lambda.Name, "test_app_somelongWackyIdthatsgoingtobewayoverthecharcountificont")
	assert.Equal(lambda.ConstructsRef, []core.AnnotationKey{eu.AnnotationKey})
	assert.Equal(lambda.Role, role)
	assert.Equal(lambda.VpcConfig, LambdaVpcConfig{})
}

func Test_Provider(t *testing.T) {
	assert := assert.New(t)
	eu := &core.ExecutionUnit{AnnotationKey: core.AnnotationKey{ID: "test"}}
	role := &iam.IamRole{Name: "testRole"}
	lambda := NewLambdaFunction(eu, "test-app", role)
	assert.Equal(lambda.Provider(), resources.AWS_PROVIDER)
}

func Test_Id(t *testing.T) {
	assert := assert.New(t)
	eu := &core.ExecutionUnit{AnnotationKey: core.AnnotationKey{ID: "test"}}
	role := &iam.IamRole{Name: "testRole"}
	lambda := NewLambdaFunction(eu, "test-app", role)
	assert.Equal(lambda.Id(), "lambda_function_test_app_test")
}

func Test_KlothoConstructRef(t *testing.T) {
	assert := assert.New(t)
	eu := &core.ExecutionUnit{AnnotationKey: core.AnnotationKey{ID: "test"}}
	role := &iam.IamRole{Name: "testRole"}
	lambda := NewLambdaFunction(eu, "test-app", role)
	assert.Equal(lambda.KlothoConstructRef(), []core.AnnotationKey{eu.Provenance()})
}
