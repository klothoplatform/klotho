package ecr

import (
	"testing"

	"github.com/klothoplatform/klotho/pkg/annotation"
	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/stretchr/testify/assert"
)

func Test_GenerateExecUnitResources(t *testing.T) {
	appName := "test-app"
	unit := &core.ExecutionUnit{AnnotationKey: core.AnnotationKey{ID: "test", Capability: annotation.ExecutionUnitCapability}}
	repo := NewEcrRepository(appName, core.AnnotationKey{ID: "test", Capability: annotation.ExecutionUnitCapability})
	image := NewEcrImage(&core.ExecutionUnit{AnnotationKey: core.AnnotationKey{ID: "test", Capability: annotation.ExecutionUnitCapability}}, appName, repo)

	cases := []struct {
		name         string
		existingRepo *EcrRepository
		want         *EcrImage
		wantErr      bool
	}{
		{
			name: "generate nothing existing",
			want: image,
		},
		{
			name:         "ecr repo already exists",
			existingRepo: NewEcrRepository(appName, core.AnnotationKey{ID: "test2", Capability: annotation.ExecutionUnitCapability}),
			want:         image,
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			dag := core.NewResourceGraph()

			if tt.existingRepo != nil {
				dag.AddResource(tt.existingRepo)

			}

			actualImage, err := GenerateEcrRepoAndImage(appName, unit, dag)
			if tt.wantErr {
				assert.Error(err)
				return
			}
			if !assert.NoError(err) {
				return
			}
			assert.Equal(tt.want.Name, actualImage.Name)
			assert.Equal(tt.want.ConstructsRef, actualImage.ConstructsRef)
			assert.Equal(tt.want.Context, actualImage.Context)
			assert.Equal(tt.want.Dockerfile, actualImage.Dockerfile)
			assert.Equal(tt.want.Repo.Id(), actualImage.Repo.Id())
			assert.Equal(tt.want.ExtraOptions, actualImage.ExtraOptions)

			for _, res := range dag.ListResources() {
				if repo, ok := res.(*EcrRepository); ok {
					if tt.existingRepo != nil {
						assert.Len(repo.ConstructsRef, 2)
					} else {
						assert.Len(repo.ConstructsRef, 1)
					}
				}
			}
		})

	}
}
