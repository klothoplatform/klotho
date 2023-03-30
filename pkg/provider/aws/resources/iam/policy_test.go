package iam

import (
	"testing"

	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/provider/aws/resources"
	"github.com/stretchr/testify/assert"
)

func NewPolicy(t *testing.T) {
	assert := assert.New(t)
	eu := &core.ExecutionUnit{AnnotationKey: core.AnnotationKey{ID: "test-eu"}}
	doc := &PolicyDocument{}
	role := NewIamPolicy("test-app", "test-policy", eu.Provenance(), doc)
	assert.Equal(role.Name, "test-app-test-role@___@__________aslk_lajsfjafkljasgfjalsfhaksja")
	assert.Equal(role.ConstructsRef, []core.AnnotationKey{eu.Provenance()})
}

func Test_PolicyProvider(t *testing.T) {
	assert := assert.New(t)
	eu := &core.ExecutionUnit{AnnotationKey: core.AnnotationKey{ID: "test"}}
	doc := &PolicyDocument{}
	role := NewIamPolicy("test-app", "test-policy", eu.Provenance(), doc)
	assert.Equal(role.Provider(), resources.AWS_PROVIDER)
}

func Test_PolicyId(t *testing.T) {
	assert := assert.New(t)
	eu := &core.ExecutionUnit{AnnotationKey: core.AnnotationKey{ID: "test"}}
	doc := &PolicyDocument{}
	role := NewIamPolicy("test-app", "test-policy", eu.Provenance(), doc)
	assert.Equal(role.Id(), "aws:iam_policy:test-app-test-policy")
}

func Test_PolicyKlothoConstructRef(t *testing.T) {
	assert := assert.New(t)
	eu := &core.ExecutionUnit{AnnotationKey: core.AnnotationKey{ID: "test"}}
	doc := &PolicyDocument{}
	role := NewIamPolicy("test-app", "test-policy", eu.Provenance(), doc)
	assert.Equal(role.KlothoConstructRef(), []core.AnnotationKey{eu.Provenance()})
}
