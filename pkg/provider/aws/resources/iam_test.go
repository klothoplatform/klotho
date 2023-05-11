package resources

import (
	"testing"

	"github.com/klothoplatform/klotho/pkg/annotation"
	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/core/coretesting"
	"github.com/stretchr/testify/assert"
)

func Test_RoleCreate(t *testing.T) {
	initialRefs := []core.AnnotationKey{{ID: "first"}}
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
					"aws:iam_role:my-app",
				},
				Deps: []coretesting.StringDep{},
			},
		},
		{
			name:    "existing role",
			role:    &IamRole{Name: "my-app", ConstructsRef: initialRefs},
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
				RoleName:            "my-app",
				Refs:                []core.AnnotationKey{{ID: "test", Capability: annotation.ExecutionUnitCapability}},
				AssumeRolePolicyDoc: LAMBDA_ASSUMER_ROLE_POLICY,
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

			assert.Equal(role.Name, "my-app")
			assert.Equal(role.ConstructsRef, metadata.Refs)
			assert.Equal(role.AssumeRolePolicyDoc, LAMBDA_ASSUMER_ROLE_POLICY)
		})
	}
}

func Test_OidcCreate(t *testing.T) {
	initialRefs := []core.AnnotationKey{{ID: "first"}}
	cases := []struct {
		name string
		oidc *OpenIdConnectProvider
		want coretesting.ResourcesExpectation
	}{
		{
			name: "nil role",
			want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:availability_zones:AvailabilityZones",
					"aws:eks_addon:app-my-cluster-addon-vpc-cni",
					"aws:eks_cluster:app-my-cluster",
					"aws:elastic_ip:app_0",
					"aws:elastic_ip:app_1",
					"aws:iam_oidc_provider:app-my-cluster",
					"aws:iam_role:app-my-cluster-ClusterAdmin",
					"aws:internet_gateway:app_igw",
					"aws:nat_gateway:app_0",
					"aws:nat_gateway:app_1",
					"aws:route_table:app_0",
					"aws:route_table:app_1",
					"aws:route_table:app_igw",
					"aws:security_group:app",
					"aws:vpc:app",
					"aws:vpc_subnet:app_private0",
					"aws:vpc_subnet:app_private1",
					"aws:vpc_subnet:app_public0",
					"aws:vpc_subnet:app_public1",
				},
				Deps: []coretesting.StringDep{
					{Source: "aws:eks_addon:app-my-cluster-addon-vpc-cni", Destination: "aws:eks_cluster:app-my-cluster"},
					{Source: "aws:eks_cluster:app-my-cluster", Destination: "aws:iam_role:app-my-cluster-ClusterAdmin"},
					{Source: "aws:eks_cluster:app-my-cluster", Destination: "aws:security_group:app"},
					{Source: "aws:eks_cluster:app-my-cluster", Destination: "aws:vpc_subnet:app_private0"},
					{Source: "aws:eks_cluster:app-my-cluster", Destination: "aws:vpc_subnet:app_private1"},
					{Source: "aws:eks_cluster:app-my-cluster", Destination: "aws:vpc_subnet:app_public0"},
					{Source: "aws:eks_cluster:app-my-cluster", Destination: "aws:vpc_subnet:app_public1"},
					{Source: "aws:iam_oidc_provider:app-my-cluster", Destination: "aws:eks_cluster:app-my-cluster"},
					{Source: "aws:internet_gateway:app_igw", Destination: "aws:vpc:app"},
					{Source: "aws:nat_gateway:app_0", Destination: "aws:elastic_ip:app_0"},
					{Source: "aws:nat_gateway:app_0", Destination: "aws:vpc_subnet:app_public0"},
					{Source: "aws:nat_gateway:app_1", Destination: "aws:elastic_ip:app_1"},
					{Source: "aws:nat_gateway:app_1", Destination: "aws:vpc_subnet:app_public1"},
					{Source: "aws:route_table:app_0", Destination: "aws:nat_gateway:app_0"},
					{Source: "aws:route_table:app_0", Destination: "aws:vpc:app"},
					{Source: "aws:route_table:app_1", Destination: "aws:nat_gateway:app_1"},
					{Source: "aws:route_table:app_1", Destination: "aws:vpc:app"},
					{Source: "aws:route_table:app_igw", Destination: "aws:internet_gateway:app_igw"},
					{Source: "aws:route_table:app_igw", Destination: "aws:vpc:app"},
					{Source: "aws:security_group:app", Destination: "aws:vpc:app"},
					{Source: "aws:vpc_subnet:app_private0", Destination: "aws:availability_zones:AvailabilityZones"},
					{Source: "aws:vpc_subnet:app_private0", Destination: "aws:vpc:app"},
					{Source: "aws:vpc_subnet:app_private1", Destination: "aws:availability_zones:AvailabilityZones"},
					{Source: "aws:vpc_subnet:app_private1", Destination: "aws:vpc:app"},
					{Source: "aws:vpc_subnet:app_public0", Destination: "aws:availability_zones:AvailabilityZones"},
					{Source: "aws:vpc_subnet:app_public0", Destination: "aws:vpc:app"},
					{Source: "aws:vpc_subnet:app_public1", Destination: "aws:availability_zones:AvailabilityZones"},
					{Source: "aws:vpc_subnet:app_public1", Destination: "aws:vpc:app"},
				},
			},
		},
		{
			name: "existing role",
			oidc: &OpenIdConnectProvider{Name: "app-my-cluster", ConstructsRef: initialRefs},
			want: coretesting.ResourcesExpectation{
				Nodes: []string{
					"aws:iam_oidc_provider:app-my-cluster",
				},
				Deps: []coretesting.StringDep{},
			},
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			dag := core.NewResourceGraph()
			if tt.oidc != nil {
				dag.AddResource(tt.oidc)
			}
			metadata := OidcCreateParams{
				ClusterName: "my-cluster",
				Refs:        []core.AnnotationKey{{ID: "test", Capability: annotation.ExecutionUnitCapability}},
				AppName:     "app",
			}
			oidc := &OpenIdConnectProvider{}
			err := oidc.Create(dag, metadata)

			if !assert.NoError(err) {
				return
			}
			tt.want.Assert(t, dag)

			graphOidc := dag.GetResourceByVertexId(oidc.Id().String()).(*OpenIdConnectProvider)

			assert.Equal(graphOidc.Name, "app-my-cluster")
			if tt.oidc == nil {
				assert.Equal(graphOidc.ConstructsRef, metadata.Refs)
			} else {
				assert.Equal(graphOidc.KlothoConstructRef(), append(initialRefs, core.AnnotationKey{ID: "test", Capability: annotation.ExecutionUnitCapability}))

			}
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
			role:          NewIamRole("test-app", "test-role", []core.AnnotationKey{}, nil),
		},
		{
			name: "Add role, one already exists",
			existingRoles: map[string]*IamRole{
				unitId: NewIamRole("test-app", "diff-role", []core.AnnotationKey{}, nil),
			},
			role:    NewIamRole("test-app", "test-role", []core.AnnotationKey{}, nil),
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
