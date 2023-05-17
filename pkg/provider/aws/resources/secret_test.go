package resources

import (
	"fmt"
	"testing"

	"github.com/klothoplatform/klotho/pkg/annotation"
	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/core/coretesting"
	"github.com/stretchr/testify/assert"
)

func Test_SecretCreate(t *testing.T) {
	cases := []struct {
		name    string
		secret  *Secret
		want    coretesting.ResourcesExpectation
		wantErr bool
	}{
		{
			name: "nil secret",
			want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:secret:my-app-test",
				},
				Deps: []coretesting.StringDep{},
			},
		},
		{
			name:    "existing secret",
			secret:  &Secret{Name: "my-app-test"},
			wantErr: true,
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {

			assert := assert.New(t)
			dag := core.NewResourceGraph()

			if tt.secret != nil {
				dag.AddResource(tt.secret)
			}

			metadata := SecretCreateParams{
				AppName: "my-app",
				Refs:    []core.AnnotationKey{{ID: "test", Capability: annotation.PersistCapability}},
				Name:    "test",
			}
			secret := &Secret{}
			err := secret.Create(dag, metadata)
			if tt.wantErr {
				assert.Error(err)
				return
			}
			if !assert.NoError(err) {
				return
			}
			tt.want.Assert(t, dag)

			fmt.Println(coretesting.ResoucesFromDAG(dag).GoString())
			graphInstance := dag.GetResource(secret.Id())
			secret = graphInstance.(*Secret)

			assert.Equal(secret.Name, "my-app-test")
			assert.ElementsMatch(secret.ConstructsRef, metadata.Refs)
		})
	}
}

func Test_SecretVersionCreate(t *testing.T) {
	cases := []struct {
		name    string
		secret  *SecretVersion
		want    coretesting.ResourcesExpectation
		wantErr bool
	}{
		{
			name: "nil secret",
			want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:secret:my-app-test",
					"aws:secret_version:my-app-test",
				},
				Deps: []coretesting.StringDep{
					{Source: "aws:secret_version:my-app-test", Destination: "aws:secret:my-app-test"},
				},
			},
		},
		{
			name:    "existing secret",
			secret:  &SecretVersion{Name: "my-app-test"},
			wantErr: true,
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {

			assert := assert.New(t)
			dag := core.NewResourceGraph()

			if tt.secret != nil {
				dag.AddResource(tt.secret)
			}

			metadata := SecretVersionCreateParams{
				AppName: "my-app",
				Refs:    []core.AnnotationKey{{ID: "test", Capability: annotation.PersistCapability}},
				Name:    "test",
			}
			secret := &SecretVersion{}
			err := secret.Create(dag, metadata)
			if tt.wantErr {
				assert.Error(err)
				return
			}
			if !assert.NoError(err) {
				return
			}
			tt.want.Assert(t, dag)

			fmt.Println(coretesting.ResoucesFromDAG(dag).GoString())
			graphInstance := dag.GetResource(secret.Id())
			secret = graphInstance.(*SecretVersion)

			assert.Equal(secret.Name, "my-app-test")
			assert.ElementsMatch(secret.ConstructsRef, metadata.Refs)
		})
	}
}
