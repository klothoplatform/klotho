package ecr

import (
	"testing"

	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/provider/aws/resources"
	"github.com/stretchr/testify/assert"
)

func Test_NewImage(t *testing.T) {
	assert := assert.New(t)
	eu := &core.ExecutionUnit{AnnotationKey: core.AnnotationKey{ID: "test-eu"}, DockerfilePath: "somedir"}
	image := NewEcrImage(eu, "test-app")
	assert.Equal(image.Name, "test-app-test-eu")
	assert.Equal(image.ConstructsRef, []core.AnnotationKey{eu.AnnotationKey})
	assert.Equal(image.Context, "./test-eu")
	assert.Equal(image.Dockerfile, "./test-eu/somedir")
	assert.Equal(image.ExtraOptions, []string{"--platform", "linux/amd64", "--quiet"})
}

func Test_ImageProvider(t *testing.T) {
	assert := assert.New(t)
	eu := &core.ExecutionUnit{AnnotationKey: core.AnnotationKey{ID: "test"}}
	image := NewEcrImage(eu, "test-app")
	assert.Equal(image.Provider(), resources.AWS_PROVIDER)
}

func Test_ImageId(t *testing.T) {
	assert := assert.New(t)
	eu := &core.ExecutionUnit{AnnotationKey: core.AnnotationKey{ID: "test"}}
	image := NewEcrImage(eu, "test-app")
	assert.Equal(image.Id(), "aws:ecr_image:test-app-test")
}

func Test_ImageKlothoConstructRef(t *testing.T) {
	assert := assert.New(t)
	eu := &core.ExecutionUnit{AnnotationKey: core.AnnotationKey{ID: "test"}}
	image := NewEcrImage(eu, "test-app")
	assert.Equal(image.KlothoConstructRef(), []core.AnnotationKey{eu.Provenance()})
}
