package iam

import (
	"testing"

	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/provider/aws/resources"
	"github.com/stretchr/testify/assert"
)

func Test_NewLambdaFunction(t *testing.T) {
	assert := assert.New(t)
	eu := &core.ExecutionUnit{AnnotationKey: core.AnnotationKey{ID: "test-eu"}}
	role := NewIamRole("test-app", "test-role@#$%@#^$%&_?/;/aslk;lajsfjafkljasgfjalsfhaksjalsfakjhlkkljh;lkhlkjhl;kna;lfbkjkhaksjb;lkj", eu.Provenance(), EC2_ASSUMER_ROLE_POLICY)
	assert.Equal(role.Name, "test-app-test-role@___@__________aslk_lajsfjafkljasgfjalsfhaksja")

}

func Test_Provider(t *testing.T) {
	assert := assert.New(t)
	eu := &core.ExecutionUnit{AnnotationKey: core.AnnotationKey{ID: "test"}}
	role := NewIamRole("test-app", "test-role", eu.Provenance(), EC2_ASSUMER_ROLE_POLICY)
	assert.Equal(role.Provider(), resources.AWS_PROVIDER)
}

func Test_Id(t *testing.T) {
	assert := assert.New(t)
	eu := &core.ExecutionUnit{AnnotationKey: core.AnnotationKey{ID: "test"}}
	role := NewIamRole("test-app", "test-role", eu.Provenance(), EC2_ASSUMER_ROLE_POLICY)
	assert.Equal(role.Id(), "iam_role_test-app-test-role")
}

func Test_KlothoConstructRef(t *testing.T) {
	assert := assert.New(t)
	eu := &core.ExecutionUnit{AnnotationKey: core.AnnotationKey{ID: "test"}}
	role := NewIamRole("test-app", "test-role", eu.Provenance(), EC2_ASSUMER_ROLE_POLICY)
	assert.Equal(role.KlothoConstructRef(), []core.AnnotationKey{eu.Provenance()})
}
