package resources

import (
	"testing"

	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/stretchr/testify/assert"
)

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
			resource:         []core.IaCValue{{Resource: bucket, Property: core.ARN_IAC_VALUE}, {Resource: bucket, Property: ALL_BUCKET_DIRECTORY_IAC_VALUE}},
			want: StatementEntry{
				Effect:   "Allow",
				Action:   []string{"s3:*"},
				Resource: []core.IaCValue{{Resource: bucket, Property: core.ARN_IAC_VALUE}, {Resource: bucket, Property: ALL_BUCKET_DIRECTORY_IAC_VALUE}},
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
									Resource: []core.IaCValue{{Resource: bucket, Property: core.ARN_IAC_VALUE}, {Resource: bucket, Property: ALL_BUCKET_DIRECTORY_IAC_VALUE}},
								},
							},
						},
					},
				},
			},
			actions:  []string{"s3:*"},
			resource: []core.IaCValue{{Resource: bucket, Property: core.ARN_IAC_VALUE}, {Resource: bucket, Property: ALL_BUCKET_DIRECTORY_IAC_VALUE}},
			want: StatementEntry{
				Effect:   "Allow",
				Action:   []string{"dynamodb:*"},
				Resource: []core.IaCValue{{Resource: bucket, Property: core.ARN_IAC_VALUE}, {Resource: bucket, Property: ALL_BUCKET_DIRECTORY_IAC_VALUE}},
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
