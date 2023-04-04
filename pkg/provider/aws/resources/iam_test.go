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
		existingPolicies map[string]*PolicyDocument
		actions          []string
		resource         []core.IaCValue
		want             StatementEntry
	}{
		{
			name:             "Add policy, none exists",
			existingPolicies: map[string]*PolicyDocument{},
			actions:          []string{"s3:*"},
			resource:         []core.IaCValue{{Resource: bucket, Property: core.ARN_IAC_VALUE}, {Resource: bucket, Property: core.ALL_BUCKET_DIRECTORY_IAC_VALUE}},
			want: StatementEntry{
				Effect:   "Allow",
				Action:   []string{"s3:*"},
				Resource: []core.IaCValue{{Resource: bucket, Property: core.ARN_IAC_VALUE}, {Resource: bucket, Property: core.ALL_BUCKET_DIRECTORY_IAC_VALUE}},
			},
		},
		{
			name: "Add policy, one already exists",
			existingPolicies: map[string]*PolicyDocument{
				unitId: {
					Version: VERSION,
					Statement: []StatementEntry{
						{
							Effect:   "Allow",
							Action:   []string{"dynamodb:*"},
							Resource: []core.IaCValue{{Resource: bucket, Property: core.ARN_IAC_VALUE}, {Resource: bucket, Property: core.ALL_BUCKET_DIRECTORY_IAC_VALUE}},
						},
					},
				},
			},
			actions:  []string{"s3:*"},
			resource: []core.IaCValue{{Resource: bucket, Property: core.ARN_IAC_VALUE}, {Resource: bucket, Property: core.ALL_BUCKET_DIRECTORY_IAC_VALUE}},
			want: StatementEntry{
				Effect:   "Allow",
				Action:   []string{"s3:*"},
				Resource: []core.IaCValue{{Resource: bucket, Property: core.ARN_IAC_VALUE}, {Resource: bucket, Property: core.ALL_BUCKET_DIRECTORY_IAC_VALUE}},
			},
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)

			p := PolicyGenerator{
				unitsPolicies: tt.existingPolicies,
			}

			p.AddAllowPolicyToUnit(unitId, tt.actions, tt.resource)
			policies := p.unitsPolicies[unitId]
			assert.Contains(policies.Statement, tt.want)
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
			role:          NewIamRole("test-app", "test-role", []core.AnnotationKey{}, ""),
		},
		{
			name: "Add role, one already exists",
			existingRoles: map[string]*IamRole{
				unitId: NewIamRole("test-app", "diff-role", []core.AnnotationKey{}, ""),
			},
			role:    NewIamRole("test-app", "test-role", []core.AnnotationKey{}, ""),
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
