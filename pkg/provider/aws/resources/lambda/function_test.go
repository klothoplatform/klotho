package lambda

import (
	"testing"

	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/provider/aws/resources"
	"github.com/klothoplatform/klotho/pkg/provider/aws/resources/ecr"
	"github.com/klothoplatform/klotho/pkg/provider/aws/resources/iam"
	"github.com/stretchr/testify/assert"
)

func Test_NewLambdaFunction(t *testing.T) {
	assert := assert.New(t)
	eu := &core.ExecutionUnit{AnnotationKey: core.AnnotationKey{ID: "somelongWackyId%^&**thatsgoingtobewayoverthecharcountificontinuetotypethingsouthere"}}
	role := &iam.IamRole{Name: "testRole"}
	image := &ecr.EcrImage{}
	lambda := NewLambdaFunction(eu, "test-app", role, image)
	assert.Equal(lambda.Name, "test_app_somelongWackyIdthatsgoingtobewayoverthecharcountificont")
	assert.Equal(lambda.ConstructsRef, []core.AnnotationKey{eu.AnnotationKey})
	assert.Equal(lambda.Role, role)
	assert.Equal(lambda.VpcConfig, LambdaVpcConfig{})
	assert.Equal(lambda.Image, image)
}

func Test_Provider(t *testing.T) {
	assert := assert.New(t)
	eu := &core.ExecutionUnit{AnnotationKey: core.AnnotationKey{ID: "test"}}
	role := &iam.IamRole{Name: "testRole"}
	image := &ecr.EcrImage{}
	lambda := NewLambdaFunction(eu, "test-app", role, image)
	assert.Equal(lambda.Provider(), resources.AWS_PROVIDER)
}

func Test_Id(t *testing.T) {
	assert := assert.New(t)
	eu := &core.ExecutionUnit{AnnotationKey: core.AnnotationKey{ID: "test"}}
	role := &iam.IamRole{Name: "testRole"}
	image := &ecr.EcrImage{}
	lambda := NewLambdaFunction(eu, "test-app", role, image)
	assert.Equal(lambda.Id(), "aws:lambda_function:test_app_test")
}

func Test_KlothoConstructRef(t *testing.T) {
	assert := assert.New(t)
	eu := &core.ExecutionUnit{AnnotationKey: core.AnnotationKey{ID: "test"}}
	role := &iam.IamRole{Name: "testRole"}
	image := &ecr.EcrImage{}
	lambda := NewLambdaFunction(eu, "test-app", role, image)
	assert.Equal(lambda.KlothoConstructRef(), []core.AnnotationKey{eu.Provenance()})
}
