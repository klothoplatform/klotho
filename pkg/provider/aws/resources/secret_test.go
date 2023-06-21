package resources

import (
	"testing"

	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/core/coretesting"
	"github.com/stretchr/testify/assert"
)

func Test_SecretCreate(t *testing.T) {
	eu := &core.ExecutionUnit{Name: "test"}
	eu2 := &core.ExecutionUnit{Name: "first"}
	initialRefs := core.BaseConstructSetOf(eu2)
	cases := []coretesting.CreateCase[SecretCreateParams, *Secret]{
		{
			Name: "nil igw",
			Want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:secret:my-app-secret",
				},
				Deps: []coretesting.StringDep{},
			},
			Check: func(assert *assert.Assertions, s *Secret) {
				assert.Equal(s.Name, "my-app-secret")
				assert.Equal(s.ConstructsRef, core.BaseConstructSetOf(eu))
			},
		},
		{
			Name:     "existing igw",
			Existing: &Secret{Name: "my-app-secret", ConstructsRef: initialRefs},
			WantErr:  true,
		},
	}
	for _, tt := range cases {
		t.Run(tt.Name, func(t *testing.T) {
			tt.Params = SecretCreateParams{
				AppName: "my-app",
				Refs:    core.BaseConstructSetOf(eu),
				Name:    "secret",
			}
			tt.Run(t)
		})
	}
}

func Test_SecretVersionCreate(t *testing.T) {
	eu := &core.ExecutionUnit{Name: "test"}
	eu2 := &core.ExecutionUnit{Name: "first"}
	initialRefs := core.BaseConstructSetOf(eu2)
	cases := []coretesting.CreateCase[SecretVersionCreateParams, *SecretVersion]{
		{
			Name: "nil igw",
			Want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:secret_version:my-app-secret",
				},
				Deps: []coretesting.StringDep{},
			},
			Check: func(assert *assert.Assertions, sv *SecretVersion) {
				assert.Equal(sv.Name, "my-app-secret")
				assert.Equal(sv.ConstructsRef, core.BaseConstructSetOf(eu))
				assert.Equal(sv.DetectedPath, "path")
			},
		},
		{
			Name:     "existing igw",
			Existing: &SecretVersion{Name: "my-app-secret", ConstructsRef: initialRefs},
			WantErr:  true,
		},
	}
	for _, tt := range cases {
		t.Run(tt.Name, func(t *testing.T) {
			tt.Params = SecretVersionCreateParams{
				AppName:      "my-app",
				Refs:         core.BaseConstructSetOf(eu),
				Name:         "secret",
				DetectedPath: "path",
			}
			tt.Run(t)
		})
	}
}

func Test_SecretVersionMakeOperational(t *testing.T) {
	cases := []coretesting.MakeOperationalCase[*SecretVersion]{
		{
			Name:     "only sv",
			Resource: &SecretVersion{Name: "secretv"},
			AppName:  "my-app",
			Want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:secret_version:secretv",
					"aws:secret:my-app-secretv",
				},
				Deps: []coretesting.StringDep{
					{Source: "aws:secret_version:secretv", Destination: "aws:secret:my-app-secretv"},
				},
			},
			Check: func(assert *assert.Assertions, sv *SecretVersion) {
				assert.NotNil(sv.Secret)
			},
		},
		{
			Name:     "sv with downstream secret",
			Resource: &SecretVersion{Name: "secretv"},
			AppName:  "my-app",
			Existing: []core.Resource{&Secret{Name: "test-down"}},
			ExistingDependencies: []coretesting.StringDep{
				{Source: "aws:secret_version:secretv", Destination: "aws:secret:test-down"},
			},
			Want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:secret_version:secretv",
					"aws:secret:test-down",
				},
				Deps: []coretesting.StringDep{
					{Source: "aws:secret_version:secretv", Destination: "aws:secret:test-down"},
				},
			},
			Check: func(assert *assert.Assertions, sv *SecretVersion) {
				assert.Equal(sv.Secret.Name, "test-down")
			},
		},
		{
			Name:     "sv with secret set",
			Resource: &SecretVersion{Name: "secretv", Secret: &Secret{Name: "s"}},
			AppName:  "my-app",
			Want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:secret_version:secretv",
					"aws:secret:s",
				},
				Deps: []coretesting.StringDep{
					{Source: "aws:secret_version:secretv", Destination: "aws:secret:s"},
				},
			},
			Check: func(assert *assert.Assertions, sv *SecretVersion) {
				assert.Equal(sv.Secret.Name, "s")
			},
		},
		{
			Name:     "multiple secrets error",
			Resource: &SecretVersion{Name: "my_app"},
			AppName:  "my-app",
			Existing: []core.Resource{&Secret{Name: "test-down"}, &Secret{Name: "test"}},
			ExistingDependencies: []coretesting.StringDep{
				{Source: "aws:secret_version:my_app", Destination: "aws:secret:test-down"},
				{Source: "aws:secret_version:my_app", Destination: "aws:secret:test"},
			},
			WantErr: true,
		},
	}
	for _, tt := range cases {
		t.Run(tt.Name, func(t *testing.T) {
			tt.Run(t)
		})
	}
}
