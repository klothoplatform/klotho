package cloudwatch

import (
	"testing"

	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/provider/aws/resources"
	"github.com/stretchr/testify/assert"
)

func Test_NewImage(t *testing.T) {
	assert := assert.New(t)
	eu := &core.ExecutionUnit{AnnotationKey: core.AnnotationKey{ID: "test-eu"}, DockerfilePath: "somedir"}
	image := NewLogGroup("test-app", "/aws/lambda/main", eu.Provenance(), 2)
	assert.Equal(image.Name, "test_app_awslambdamain")
	assert.Equal(image.ConstructsRef, []core.AnnotationKey{eu.AnnotationKey})
	assert.Equal(image.LogGroupName, "/aws/lambda/main")
	assert.Equal(image.RetentionInDays, 2)
}

func Test_ImageProvider(t *testing.T) {
	assert := assert.New(t)
	eu := &core.ExecutionUnit{AnnotationKey: core.AnnotationKey{ID: "test"}}
	image := NewLogGroup("test-app", "/aws/lambda/main", eu.Provenance(), 2)
	assert.Equal(image.Provider(), resources.AWS_PROVIDER)
}

func Test_ImageId(t *testing.T) {
	assert := assert.New(t)
	eu := &core.ExecutionUnit{AnnotationKey: core.AnnotationKey{ID: "test"}}
	image := NewLogGroup("test-app", "/aws/lambda/main", eu.Provenance(), 2)
	assert.Equal(image.Id(), "log_group_test_app_awslambdamain")
}

func Test_ImageKlothoConstructRef(t *testing.T) {
	assert := assert.New(t)
	eu := &core.ExecutionUnit{AnnotationKey: core.AnnotationKey{ID: "test"}}
	image := NewLogGroup("test-app", "/aws/lambda/main", eu.Provenance(), 2)
	assert.Equal(image.KlothoConstructRef(), []core.AnnotationKey{eu.Provenance()})
}
