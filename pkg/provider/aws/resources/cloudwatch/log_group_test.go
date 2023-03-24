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
	lg := NewLogGroup("test-app", "/aws/lambda/main", eu.Provenance(), 2)
	assert.Equal(lg.Name, "test_app_awslambdamain")
	assert.Equal(lg.ConstructsRef, []core.AnnotationKey{eu.AnnotationKey})
	assert.Equal(lg.LogGroupName, "/aws/lambda/main")
	assert.Equal(lg.RetentionInDays, 2)
}

func Test_ImageProvider(t *testing.T) {
	assert := assert.New(t)
	eu := &core.ExecutionUnit{AnnotationKey: core.AnnotationKey{ID: "test"}}
	lg := NewLogGroup("test-app", "/aws/lambda/main", eu.Provenance(), 2)
	assert.Equal(lg.Provider(), resources.AWS_PROVIDER)
}

func Test_ImageId(t *testing.T) {
	assert := assert.New(t)
	eu := &core.ExecutionUnit{AnnotationKey: core.AnnotationKey{ID: "test"}}
	lg := NewLogGroup("test-app", "/aws/lambda/main", eu.Provenance(), 2)
	assert.Equal(lg.Id(), "aws:log_group:test_app_awslambdamain")
}

func Test_ImageKlothoConstructRef(t *testing.T) {
	assert := assert.New(t)
	eu := &core.ExecutionUnit{AnnotationKey: core.AnnotationKey{ID: "test"}}
	lg := NewLogGroup("test-app", "/aws/lambda/main", eu.Provenance(), 2)
	assert.Equal(lg.KlothoConstructRef(), []core.AnnotationKey{eu.Provenance()})
}
