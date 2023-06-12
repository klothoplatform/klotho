package resources

import (
	"testing"

	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/core/coretesting"
	"github.com/stretchr/testify/assert"
)

func Test_EcrRepositoryCreate(t *testing.T) {
	eu := &core.ExecutionUnit{Name: "first"}
	eu2 := &core.ExecutionUnit{Name: "test"}
	initialRefs := core.BaseConstructSetOf(eu)
	cases := []struct {
		name string
		repo *EcrRepository
		want coretesting.ResourcesExpectation
	}{
		{
			name: "nil repo",
			want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:ecr_repo:my-app",
				},
				Deps: []coretesting.StringDep{},
			},
		},
		{
			name: "existing repo",
			repo: &EcrRepository{Name: "my-app", ConstructsRef: initialRefs, ForceDelete: true},
			want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:ecr_repo:my-app",
				},
				Deps: []coretesting.StringDep{},
			},
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			dag := core.NewResourceGraph()
			if tt.repo != nil {
				dag.AddResource(tt.repo)
			}
			metadata := RepoCreateParams{
				AppName: "my-app",
				Refs:    core.BaseConstructSetOf(eu2),
			}

			repo := &EcrRepository{}
			err := repo.Create(dag, metadata)

			if !assert.NoError(err) {
				return
			}

			tt.want.Assert(t, dag)
			graphRepo := dag.GetResource(repo.Id())
			repo = graphRepo.(*EcrRepository)
			assert.Equal(repo.Name, "my-app")
			if tt.repo == nil {
				assert.Equal(repo.ConstructsRef, metadata.Refs)
			} else {
				assert.Equal(repo, tt.repo)
				expect := initialRefs.CloneWith(core.BaseConstructSetOf(eu2))
				assert.Equal(repo.BaseConstructsRef(), expect)
			}
		})
	}
}

func Test_EcrImageCreate(t *testing.T) {
	eu := &core.ExecutionUnit{Name: "first"}
	initialRefs := core.BaseConstructSetOf(eu)
	cases := []struct {
		name    string
		image   *EcrImage
		want    coretesting.ResourcesExpectation
		wantErr bool
	}{
		{
			name: "nil repo",
			want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:ecr_image:my-app-test-unit",
					"aws:ecr_repo:my-app",
				},
				Deps: []coretesting.StringDep{
					{Source: "aws:ecr_image:my-app-test-unit", Destination: "aws:ecr_repo:my-app"},
				},
			},
		},
		{
			name:    "existing image",
			image:   &EcrImage{Name: "my-app-test-unit", ConstructsRef: initialRefs},
			wantErr: true,
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			dag := core.NewResourceGraph()
			if tt.image != nil {
				dag.AddResource(tt.image)
			}
			metadata := ImageCreateParams{
				AppName: "my-app",
				Refs:    core.BaseConstructSetOf(&core.ExecutionUnit{Name: "test"}),
				Name:    "test-unit",
			}
			image := &EcrImage{}
			err := image.Create(dag, metadata)

			if tt.wantErr {
				assert.Error(err)
				return
			}
			if !assert.NoError(err) {
				return
			}
			tt.want.Assert(t, dag)
			graphImage := dag.GetResource(image.Id())
			image = graphImage.(*EcrImage)

			assert.Equal(image.Name, "my-app-test-unit")
			assert.Equal(image.ConstructsRef, metadata.Refs)
		})
	}
}

func Test_EcrImageConfigure(t *testing.T) {
	cases := []struct {
		name    string
		params  EcrImageConfigureParams
		want    *EcrImage
		wantErr bool
	}{
		{
			name: "filled params",
			params: EcrImageConfigureParams{
				Context:    "context",
				Dockerfile: "dockerfile",
			},
			want: &EcrImage{Context: "context", Dockerfile: "dockerfile", ExtraOptions: []string{"--platform", "linux/amd64", "--quiet"}},
		},
		{
			name:    "no context",
			params:  EcrImageConfigureParams{Dockerfile: "dockerfile"},
			wantErr: true,
		},
		{
			name:    "no dockerfile",
			params:  EcrImageConfigureParams{Context: "dockerfile"},
			wantErr: true,
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)

			image := &EcrImage{}
			err := image.Configure(tt.params)

			if tt.wantErr {
				assert.Error(err)
				return
			}
			if !assert.NoError(err) {
				return
			}

			assert.Equal(tt.want, image)
		})
	}
}
