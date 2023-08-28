package resources

import (
	"testing"

	"github.com/klothoplatform/klotho/pkg/compiler/types"
	"github.com/klothoplatform/klotho/pkg/construct"
	"github.com/klothoplatform/klotho/pkg/construct/coretesting"
	"github.com/stretchr/testify/assert"
)

func Test_RoleCreate(t *testing.T) {
	eu := &types.ExecutionUnit{Name: "first"}
	initialRefs := construct.BaseConstructSetOf(eu)
	cases := []struct {
		name    string
		role    *IamRole
		want    coretesting.ResourcesExpectation
		wantErr bool
	}{
		{
			name: "nil role",
			want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:iam_role:my-app-executionRole",
				},
				Deps: []coretesting.StringDep{},
			},
		},
		{
			name: "existing role",
			role: &IamRole{Name: "my-app-executionRole", ConstructRefs: initialRefs},
			want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:iam_role:my-app-executionRole",
				},
				Deps: []coretesting.StringDep{},
			},
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			dag := construct.NewResourceGraph()
			if tt.role != nil {
				dag.AddResource(tt.role)
			}
			metadata := RoleCreateParams{
				AppName: "my-app",
				Name:    "executionRole",
				Refs:    construct.BaseConstructSetOf(&types.ExecutionUnit{Name: "test"}),
			}
			role := &IamRole{}
			err := role.Create(dag, metadata)

			if tt.wantErr {
				assert.Error(err)
				return
			}
			if !assert.NoError(err) {
				return
			}
			tt.want.Assert(t, dag)

			assert.Equal(role.Name, "my-app-executionRole")
			assert.Equal(role.ConstructRefs, metadata.Refs)
		})
	}
}

func Test_PolicyCreate(t *testing.T) {
	eu := &types.ExecutionUnit{Name: "first"}
	initialRefs := construct.BaseConstructSetOf(eu)
	cases := []struct {
		name    string
		policy  *IamPolicy
		want    coretesting.ResourcesExpectation
		wantErr bool
	}{
		{
			name: "nil policy",
			want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:iam_policy:my-app-policy",
				},
				Deps: []coretesting.StringDep{},
			},
		},
		{
			name:   "existing policy",
			policy: &IamPolicy{Name: "my-app-policy", ConstructRefs: initialRefs},
			want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:iam_policy:my-app-policy",
				},
				Deps: []coretesting.StringDep{},
			},
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			dag := construct.NewResourceGraph()
			if tt.policy != nil {
				dag.AddResource(tt.policy)
			}
			metadata := IamPolicyCreateParams{
				AppName: "my-app",
				Name:    "policy",
				Refs:    construct.BaseConstructSetOf(&types.ExecutionUnit{Name: "test"}),
			}
			policy := &IamPolicy{}
			err := policy.Create(dag, metadata)

			if tt.wantErr {
				assert.Error(err)
				return
			}
			if !assert.NoError(err) {
				return
			}
			tt.want.Assert(t, dag)

			policy, _ = construct.GetResource[*IamPolicy](dag, policy.Id())
			assert.Equal(policy.Name, "my-app-policy")
			if tt.policy == nil {
				assert.Equal(policy.ConstructRefs, metadata.Refs)
			} else {
				initialRefs.Add(&types.ExecutionUnit{Name: "test"})
				assert.Equal(policy.ConstructRefs, initialRefs)

			}
		})
	}
}

func Test_OidcCreate(t *testing.T) {
	eu := &types.ExecutionUnit{Name: "test"}
	eu2 := &types.ExecutionUnit{Name: "first"}
	initialRefs := construct.BaseConstructSetOf(eu2)
	cases := []coretesting.CreateCase[OidcCreateParams, *OpenIdConnectProvider]{
		{
			Name: "nil oidc",
			Want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:iam_oidc_provider:my-app-cluster",
				},
				Deps: []coretesting.StringDep{},
			},
			Check: func(assert *assert.Assertions, oidc *OpenIdConnectProvider) {
				assert.Equal(oidc.Name, "my-app-cluster")
				assert.Equal(oidc.ConstructRefs, construct.BaseConstructSetOf(eu))
			},
		},
		{
			Name:     "existing oidc",
			Existing: &OpenIdConnectProvider{Name: "my-app-cluster", ConstructRefs: initialRefs},
			Want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:iam_oidc_provider:my-app-cluster",
				},
				Deps: []coretesting.StringDep{},
			},
			Check: func(assert *assert.Assertions, oidc *OpenIdConnectProvider) {
				assert.Equal(oidc.Name, "my-app-cluster")
				expect := initialRefs.CloneWith(construct.BaseConstructSetOf(eu))
				assert.Equal(oidc.ConstructRefs, expect)
			},
		},
	}
	for _, tt := range cases {
		t.Run(tt.Name, func(t *testing.T) {
			tt.Params = OidcCreateParams{
				AppName:     "my-app",
				Refs:        construct.BaseConstructSetOf(eu),
				ClusterName: "cluster",
			}
			tt.Run(t)
		})
	}
}

func Test_InstanceProfileCreate(t *testing.T) {
	eu := &types.ExecutionUnit{Name: "test"}
	eu2 := &types.ExecutionUnit{Name: "first"}
	initialRefs := construct.BaseConstructSetOf(eu2)
	cases := []coretesting.CreateCase[InstanceProfileCreateParams, *InstanceProfile]{
		{
			Name: "nil profile",
			Want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:iam_instance_profile:my-app-profile",
				},
				Deps: []coretesting.StringDep{},
			},
			Check: func(assert *assert.Assertions, profile *InstanceProfile) {
				assert.Equal(profile.Name, "my-app-profile")
				assert.Equal(profile.ConstructRefs, construct.BaseConstructSetOf(eu))
			},
		},
		{
			Name:     "existing profile",
			Existing: &InstanceProfile{Name: "my-app-profile", ConstructRefs: initialRefs},
			Want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:iam_instance_profile:my-app-profile",
				},
				Deps: []coretesting.StringDep{},
			},
			Check: func(assert *assert.Assertions, profile *InstanceProfile) {
				assert.Equal(profile.Name, "my-app-profile")
				expect := initialRefs.CloneWith(construct.BaseConstructSetOf(eu))
				assert.Equal(profile.ConstructRefs, expect)
			},
		},
	}
	for _, tt := range cases {
		t.Run(tt.Name, func(t *testing.T) {
			tt.Params = InstanceProfileCreateParams{
				AppName: "my-app",
				Refs:    construct.BaseConstructSetOf(eu),
				Name:    "profile",
			}
			tt.Run(t)
		})
	}
}
