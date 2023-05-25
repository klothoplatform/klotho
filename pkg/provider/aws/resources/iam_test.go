package resources

import (
	"testing"

	"github.com/klothoplatform/klotho/pkg/annotation"
	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/core/coretesting"
	"github.com/stretchr/testify/assert"
)

func Test_RoleCreate(t *testing.T) {
	initialRefs := core.AnnotationKeySetOf(core.AnnotationKey{ID: "first"})
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
			name:    "existing role",
			role:    &IamRole{Name: "my-app-executionRole", ConstructsRef: initialRefs},
			wantErr: true,
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			dag := core.NewResourceGraph()
			if tt.role != nil {
				dag.AddResource(tt.role)
			}
			metadata := RoleCreateParams{
				AppName: "my-app",
				Name:    "executionRole",
				Refs:    core.AnnotationKeySetOf(core.AnnotationKey{ID: "test", Capability: annotation.ExecutionUnitCapability}),
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
			assert.Equal(role.ConstructsRef, metadata.Refs)
		})
	}
}

func Test_PolicyCreate(t *testing.T) {
	initialRefs := core.AnnotationKeySetOf(core.AnnotationKey{ID: "first"})
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
			name:    "existing policy",
			policy:  &IamPolicy{Name: "my-app-policy", ConstructsRef: initialRefs},
			wantErr: true,
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			dag := core.NewResourceGraph()
			if tt.policy != nil {
				dag.AddResource(tt.policy)
			}
			metadata := IamPolicyCreateParams{
				AppName: "my-app",
				Name:    "policy",
				Refs:    core.AnnotationKeySetOf(core.AnnotationKey{ID: "test", Capability: annotation.ExecutionUnitCapability}),
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

			assert.Equal(policy.Name, "my-app-policy")
			assert.Equal(policy.ConstructsRef, metadata.Refs)
		})
	}
}

func Test_OidcCreate(t *testing.T) {
	eu := &core.ExecutionUnit{AnnotationKey: core.AnnotationKey{ID: "test", Capability: annotation.ExecutionUnitCapability}}
	initialRefs := core.AnnotationKeySetOf(core.AnnotationKey{ID: "first"})
	cases := []coretesting.CreateCase[OidcCreateParams, *OpenIdConnectProvider]{
		{
			Name: "nil oidc",
			Want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:availability_zones:AvailabilityZones",
					"aws:eks_addon:my-app-cluster-addon-vpc-cni",
					"aws:eks_cluster:my-app-cluster",
					"aws:elastic_ip:my_app_0",
					"aws:elastic_ip:my_app_1",
					"aws:iam_role:my-app-cluster-ClusterAdmin",
					"aws:internet_gateway:my_app_igw",
					"aws:nat_gateway:my_app_0",
					"aws:nat_gateway:my_app_1",
					"aws:route_table:my_app_private0",
					"aws:route_table:my_app_private1",
					"aws:route_table:my_app_public",
					"aws:security_group:my_app:my-app",
					"aws:subnet_private:my_app:my_app_private0",
					"aws:subnet_private:my_app:my_app_private1",
					"aws:subnet_public:my_app:my_app_public0",
					"aws:subnet_public:my_app:my_app_public1",
					"aws:vpc:my_app",
					"aws:iam_oidc_provider:my-app-cluster",
					"aws:region:region",
				},
				Deps: []coretesting.StringDep{
					{Source: "aws:eks_addon:my-app-cluster-addon-vpc-cni", Destination: "aws:eks_cluster:my-app-cluster"},
					{Source: "aws:eks_cluster:my-app-cluster", Destination: "aws:iam_role:my-app-cluster-ClusterAdmin"},
					{Source: "aws:eks_cluster:my-app-cluster", Destination: "aws:security_group:my_app:my-app"},
					{Source: "aws:eks_cluster:my-app-cluster", Destination: "aws:subnet_private:my_app:my_app_private0"},
					{Source: "aws:eks_cluster:my-app-cluster", Destination: "aws:subnet_private:my_app:my_app_private1"},
					{Source: "aws:eks_cluster:my-app-cluster", Destination: "aws:subnet_public:my_app:my_app_public0"},
					{Source: "aws:eks_cluster:my-app-cluster", Destination: "aws:subnet_public:my_app:my_app_public1"},
					{Source: "aws:eks_cluster:my-app-cluster", Destination: "aws:vpc:my_app"},
					{Source: "aws:internet_gateway:my_app_igw", Destination: "aws:vpc:my_app"},
					{Source: "aws:nat_gateway:my_app_0", Destination: "aws:elastic_ip:my_app_0"},
					{Source: "aws:nat_gateway:my_app_0", Destination: "aws:subnet_public:my_app:my_app_public0"},
					{Source: "aws:nat_gateway:my_app_1", Destination: "aws:elastic_ip:my_app_1"},
					{Source: "aws:nat_gateway:my_app_1", Destination: "aws:subnet_public:my_app:my_app_public1"},
					{Source: "aws:route_table:my_app_private0", Destination: "aws:nat_gateway:my_app_0"},
					{Source: "aws:route_table:my_app_private0", Destination: "aws:subnet_private:my_app:my_app_private0"},
					{Source: "aws:route_table:my_app_private0", Destination: "aws:vpc:my_app"},
					{Source: "aws:route_table:my_app_private1", Destination: "aws:nat_gateway:my_app_1"},
					{Source: "aws:route_table:my_app_private1", Destination: "aws:subnet_private:my_app:my_app_private1"},
					{Source: "aws:route_table:my_app_private1", Destination: "aws:vpc:my_app"},
					{Source: "aws:route_table:my_app_public", Destination: "aws:internet_gateway:my_app_igw"},
					{Source: "aws:route_table:my_app_public", Destination: "aws:subnet_public:my_app:my_app_public0"},
					{Source: "aws:route_table:my_app_public", Destination: "aws:subnet_public:my_app:my_app_public1"},
					{Source: "aws:route_table:my_app_public", Destination: "aws:vpc:my_app"},
					{Source: "aws:security_group:my_app:my-app", Destination: "aws:vpc:my_app"},
					{Source: "aws:subnet_private:my_app:my_app_private0", Destination: "aws:availability_zones:AvailabilityZones"},
					{Source: "aws:subnet_private:my_app:my_app_private0", Destination: "aws:vpc:my_app"},
					{Source: "aws:subnet_private:my_app:my_app_private1", Destination: "aws:availability_zones:AvailabilityZones"},
					{Source: "aws:subnet_private:my_app:my_app_private1", Destination: "aws:vpc:my_app"},
					{Source: "aws:subnet_public:my_app:my_app_public0", Destination: "aws:availability_zones:AvailabilityZones"},
					{Source: "aws:subnet_public:my_app:my_app_public0", Destination: "aws:vpc:my_app"},
					{Source: "aws:subnet_public:my_app:my_app_public1", Destination: "aws:availability_zones:AvailabilityZones"},
					{Source: "aws:subnet_public:my_app:my_app_public1", Destination: "aws:vpc:my_app"},
					{Source: "aws:iam_oidc_provider:my-app-cluster", Destination: "aws:eks_cluster:my-app-cluster"},
					{Source: "aws:iam_oidc_provider:my-app-cluster", Destination: "aws:region:region"},
				},
			},
			Check: func(assert *assert.Assertions, oidc *OpenIdConnectProvider) {
				assert.Equal(oidc.Name, "my-app-cluster")
				assert.NotNil(oidc.Cluster)
				assert.Equal(oidc.ConstructsRef, core.AnnotationKeySetOf(eu.AnnotationKey))
			},
		},
		{
			Name:     "existing oidc",
			Existing: []core.Resource{&OpenIdConnectProvider{Name: "my-app-cluster", ConstructsRef: initialRefs}},
			Want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:iam_oidc_provider:my-app-cluster",
				},
				Deps: []coretesting.StringDep{},
			},
			Check: func(assert *assert.Assertions, oidc *OpenIdConnectProvider) {
				assert.Equal(oidc.Name, "my-app-cluster")
				expect := initialRefs.CloneWith(core.AnnotationKeySetOf(eu.AnnotationKey))
				assert.Equal(oidc.ConstructsRef, expect)
			},
		},
	}
	for _, tt := range cases {
		t.Run(tt.Name, func(t *testing.T) {
			tt.Params = OidcCreateParams{
				AppName:     "my-app",
				Refs:        core.AnnotationKeySetOf(eu.AnnotationKey),
				ClusterName: "cluster",
			}
			tt.Run(t)
		})
	}
}

func Test_AddAllowPolicyToUnit(t *testing.T) {
	bucket := NewS3Bucket(&core.Fs{}, "test-app")
	unitId := "testUnit"

	cases := []struct {
		name             string
		existingPolicies map[string][]*IamPolicy
		actions          []string
		resource         []core.IaCValue
		want             StatementEntry
	}{
		{
			name:             "Add policy, none exists",
			existingPolicies: map[string][]*IamPolicy{},
			actions:          []string{"s3:*"},
			resource:         []core.IaCValue{{Resource: bucket, Property: ARN_IAC_VALUE}, {Resource: bucket, Property: ALL_BUCKET_DIRECTORY_IAC_VALUE}},
			want: StatementEntry{
				Effect:   "Allow",
				Action:   []string{"s3:*"},
				Resource: []core.IaCValue{{Resource: bucket, Property: ARN_IAC_VALUE}, {Resource: bucket, Property: ALL_BUCKET_DIRECTORY_IAC_VALUE}},
			},
		},
		{
			name: "Add policy, one already exists",
			existingPolicies: map[string][]*IamPolicy{
				unitId: {
					{
						Name: "test_policy",
						Policy: &PolicyDocument{
							Version: VERSION,
							Statement: []StatementEntry{
								{
									Effect:   "Allow",
									Action:   []string{"dynamodb:*"},
									Resource: []core.IaCValue{{Resource: bucket, Property: ARN_IAC_VALUE}, {Resource: bucket, Property: ALL_BUCKET_DIRECTORY_IAC_VALUE}},
								},
							},
						},
					},
				},
			},
			actions:  []string{"s3:*"},
			resource: []core.IaCValue{{Resource: bucket, Property: ARN_IAC_VALUE}, {Resource: bucket, Property: ALL_BUCKET_DIRECTORY_IAC_VALUE}},
			want: StatementEntry{
				Effect:   "Allow",
				Action:   []string{"dynamodb:*"},
				Resource: []core.IaCValue{{Resource: bucket, Property: ARN_IAC_VALUE}, {Resource: bucket, Property: ALL_BUCKET_DIRECTORY_IAC_VALUE}},
			},
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)

			p := PolicyGenerator{
				unitsPolicies: tt.existingPolicies,
			}

			policy := &IamPolicy{
				Name: "test_policy",
				Policy: &PolicyDocument{
					Version: VERSION,
					Statement: []StatementEntry{
						{
							Action:   tt.actions,
							Effect:   "Allow",
							Resource: tt.resource,
						},
					},
				},
			}

			p.AddAllowPolicyToUnit(unitId, policy)
			policies := p.unitsPolicies[unitId]
			assert.Len(policies, 1)
			assert.Contains(policies[0].Policy.Statement, tt.want)
		})

	}
}

func Test_AddUnitRole(t *testing.T) {
	unitId := "testUnit"
	cases := []struct {
		name          string
		existingRoles map[string]*IamRole
		role          *IamRole
		wantErr       bool
	}{
		{
			name:          "Add role, none exists",
			existingRoles: map[string]*IamRole{},
			role:          NewIamRole("test-app", "test-role", core.AnnotationKeySetOf(), nil),
		},
		{
			name: "Add role, one already exists",
			existingRoles: map[string]*IamRole{
				unitId: NewIamRole("test-app", "diff-role", core.AnnotationKeySetOf(), nil),
			},
			role:    NewIamRole("test-app", "test-role", core.AnnotationKeySetOf(), nil),
			wantErr: true,
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)

			p := PolicyGenerator{
				unitToRole: tt.existingRoles,
			}

			err := p.AddUnitRole(unitId, tt.role)
			if tt.wantErr {
				assert.Error(err)
				return
			}
			if !assert.NoError(err) {
				return
			}
			role := p.unitToRole[unitId]
			assert.Equal(role, tt.role)
		})

	}
}
