package ecr

import (
	"testing"

	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/provider/aws/resources"
	"github.com/stretchr/testify/assert"
)

func Test_NewRepo(t *testing.T) {
	assert := assert.New(t)
	eu := &core.ExecutionUnit{AnnotationKey: core.AnnotationKey{ID: "test-eu"}, DockerfilePath: "somedir"}
	image := NewEcrRepository("test-app", eu.Provenance())
	assert.Equal(image.Name, "test-app")
	assert.Equal(image.ConstructsRef, []core.AnnotationKey{eu.AnnotationKey})
	assert.True(image.ForceDelete)
}

func Test_RepoProvider(t *testing.T) {
	assert := assert.New(t)
	eu := &core.ExecutionUnit{AnnotationKey: core.AnnotationKey{ID: "test"}}
	image := NewEcrRepository("test-app", eu.Provenance())
	assert.Equal(image.Provider(), resources.AWS_PROVIDER)
}

func Test_RepoId(t *testing.T) {
	assert := assert.New(t)
	eu := &core.ExecutionUnit{AnnotationKey: core.AnnotationKey{ID: "test"}}
	image := NewEcrRepository("test-app", eu.Provenance())
	assert.Equal(image.Id(), "ecr_repo_test-app")
}

func Test_RepoKlothoConstructRef(t *testing.T) {
	assert := assert.New(t)
	eu := &core.ExecutionUnit{AnnotationKey: core.AnnotationKey{ID: "test"}}
	image := NewEcrRepository("test-app", eu.Provenance())
	assert.Equal(image.KlothoConstructRef(), []core.AnnotationKey{eu.Provenance()})
}

func Test_GenerateRepoId(t *testing.T) {
	assert := assert.New(t)
	id := GenerateRepoId("test-app")
	assert.Equal(id, "ecr_repo_test-app")
}
