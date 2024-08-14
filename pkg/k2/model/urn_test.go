package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMarshalUrnRoundtrip(t *testing.T) {
	noExplicitType := ""

	cases := []struct {
		name       string
		urn        string
		resultType string // "error" or UrnType
	}{
		{
			name:       "Project URN",
			urn:        "urn:123456790:my-project",
			resultType: string(ProjectUrnType),
		},
		{
			name:       "Project Environment URN",
			urn:        "urn:123456790:my-project:dev",
			resultType: string(EnvironmentUrnType),
		},
		{
			name:       "Project Application URN",
			urn:        "urn:123456790:my-project::my-app",
			resultType: noExplicitType,
		},
		{
			name:       "Project Environment Application URN",
			urn:        "urn:123456790:my-project:dev:my-app",
			resultType: string(ApplicationEnvironmentUrnType),
		},
		{
			name:       "Construct Instance URN",
			urn:        "urn:accountid:project:dev:app:construct/klotho.aws.S3:my-bucket",
			resultType: string(ResourceUrnType),
		},
		{
			name:       "Construct Output URN",
			urn:        "urn:accountid:project:dev:app:construct/klotho.aws.S3:my-bucket/s3_bucket:bucketName",
			resultType: string(OutputUrnType),
		},
		{
			name:       "Invalid URN with missing parts",
			urn:        "urn:123456790",
			resultType: "error",
		},
		{
			name:       "Invalid URN with invalid type format",
			urn:        "urn:123456790:my-project:dev:my-app:invalidtypeformat",
			resultType: "error",
		},
		{
			name:       "URN with special characters",
			urn:        "urn:account@id:proj$ect:dev::construct/klotho.aws.S3:my-bucket",
			resultType: string(ResourceUrnType),
		},
		{
			name:       "URN with ParentResourceID",
			urn:        "urn:accountid:project:dev::construct/klotho.aws.S3:parent/resource",
			resultType: string(ResourceUrnType),
		},
		{
			name:       "Invalid URN with too many parts",
			urn:        "urn:123456790:my-project:dev:my-app:construct/klotho.aws.S3:my-bucket:bucketName:extra-part",
			resultType: "error",
		},
		{
			name:       "Empty URN",
			urn:        "",
			resultType: "error",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var urn URN
			err := urn.UnmarshalText([]byte(tc.urn))
			if tc.resultType == "error" {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)

			str, err := urn.MarshalText()
			assert.Nil(t, err)

			assert.Equal(t, tc.urn, string(str))
			assert.Equal(t, tc.resultType, string(urn.UrnType()))
		})

	}
}

func createURNWithToggles(
	differentAccountID bool,
	differentProject bool,
	differentEnvironment bool,
	differentApplication bool,
	differentType bool,
	differentSubtype bool,
	differentParentResourceID bool,
	differentResourceID bool,
	differentOutput bool,
) (*URN, *URN) {
	urn1 := &URN{
		AccountID:        "accountid",
		Project:          "project",
		Environment:      "dev",
		Application:      "app",
		Type:             "construct",
		Subtype:          "klotho.aws.S3",
		ParentResourceID: "parent",
		ResourceID:       "my-bucket",
		Output:           "output",
	}

	urn2 := &URN{
		AccountID:        "accountid",
		Project:          "project",
		Environment:      "dev",
		Application:      "app",
		Type:             "construct",
		Subtype:          "klotho.aws.S3",
		ParentResourceID: "parent",
		ResourceID:       "my-bucket",
		Output:           "output",
	}

	if differentAccountID {
		urn2.AccountID = "different-accountid"
	}
	if differentProject {
		urn2.Project = "different-project"
	}
	if differentEnvironment {
		urn2.Environment = "different-environment"
	}
	if differentApplication {
		urn2.Application = "different-application"
	}
	if differentType {
		urn2.Type = "different-construct"
	}
	if differentSubtype {
		urn2.Subtype = "different-klotho.aws.S3"
	}
	if differentParentResourceID {
		urn2.ParentResourceID = "different-parent"
	}
	if differentResourceID {
		urn2.ResourceID = "different-my-bucket"
	}
	if differentOutput {
		urn2.Output = "different-output"
	}

	return urn1, urn2
}

func TestEquals(t *testing.T) {
	testCases := []struct {
		name                      string
		differentAccountID        bool
		differentProject          bool
		differentEnvironment      bool
		differentApplication      bool
		differentType             bool
		differentSubtype          bool
		differentParentResourceID bool
		differentResourceID       bool
		differentOutput           bool
		urn2IsNil                 bool
		urn2IsNonUrn              bool
		urn1IsSelf                bool
		expected                  bool
	}{
		{
			name:     "Equal URNs",
			expected: true,
		},
		{
			name:               "Different AccountIDs",
			differentAccountID: true,
			expected:           false,
		},
		{
			name:             "Different Projects",
			differentProject: true,
			expected:         false,
		},
		{
			name:                 "Different Environments",
			differentEnvironment: true,
			expected:             false,
		},
		{
			name:                 "Different Applications",
			differentApplication: true,
			expected:             false,
		},
		{
			name:          "Different Types",
			differentType: true,
			expected:      false,
		},
		{
			name:             "Different Subtypes",
			differentSubtype: true,
			expected:         false,
		},
		{
			name:                      "Different ParentResourceIDs",
			differentParentResourceID: true,
			expected:                  false,
		},
		{
			name:                "Different ResourceIDs",
			differentResourceID: true,
			expected:            false,
		},
		{
			name:            "Different Outputs",
			differentOutput: true,
			expected:        false,
		},
		{
			name:       "Self Comparison",
			urn1IsSelf: true,
			expected:   true,
		},
		{
			name:      "Nil URN Comparison",
			urn2IsNil: true,
			expected:  false,
		},
		{
			name:         "Non-URN Type Comparison",
			urn2IsNonUrn: true,
			expected:     false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			urn1, urn2 := createURNWithToggles(
				tc.differentAccountID,
				tc.differentProject,
				tc.differentEnvironment,
				tc.differentApplication,
				tc.differentType,
				tc.differentSubtype,
				tc.differentParentResourceID,
				tc.differentResourceID,
				tc.differentOutput,
			)

			if tc.urn2IsNil {
				urn2 = nil
			}

			if tc.urn2IsNonUrn {
				assert.Equal(t, tc.expected, urn1.Equals("non-urn-type"))
			} else if tc.urn1IsSelf {
				assert.Equal(t, tc.expected, urn1.Equals(urn1))
			} else {
				assert.Equal(t, tc.expected, urn1.Equals(urn2))
			}
		})
	}
}

func TestUrnTypes(t *testing.T) {
	testCases := []struct {
		name     string
		urn      *URN
		expected UrnType
	}{
		{
			name: "Account URN",
			urn: &URN{
				AccountID: "accountid",
			},
			expected: AccountUrnType,
		},
		{
			name: "Project URN",
			urn: &URN{
				AccountID: "accountid",
				Project:   "project",
			},
			expected: ProjectUrnType,
		},
		{
			name: "Environment URN",
			urn: &URN{
				AccountID:   "accountid",
				Project:     "project",
				Environment: "dev",
			},
			expected: EnvironmentUrnType,
		},
		{
			name: "Application Environment URN",
			urn: &URN{
				AccountID:   "accountid",
				Project:     "project",
				Environment: "dev",
				Application: "app",
			},
			expected: ApplicationEnvironmentUrnType,
		},
		{
			name: "Resource URN",
			urn: &URN{
				AccountID:   "accountid",
				Project:     "project",
				Environment: "dev",
				Type:        "construct",
				Subtype:     "klotho.aws.S3",
				ResourceID:  "resource",
			},
			expected: ResourceUrnType,
		},
		{
			name: "Output URN",
			urn: &URN{
				AccountID:        "accountid",
				Project:          "project",
				Environment:      "dev",
				Type:             "construct",
				Subtype:          "klotho.aws.S3",
				ParentResourceID: "parent",
				ResourceID:       "resource",
				Output:           "output",
			},
			expected: OutputUrnType,
		},
		{
			name: "Type URN",
			urn: &URN{
				AccountID:   "accountid",
				Project:     "project",
				Environment: "dev",
				Type:        "construct",
			},
			expected: TypeUrnType,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			urnType := tc.urn.UrnType()
			assert.Equal(t, tc.expected, urnType)
		})
	}
}

func TestCompare(t *testing.T) {
	testCases := []struct {
		name                      string
		differentAccountID        bool
		differentProject          bool
		differentEnvironment      bool
		differentApplication      bool
		differentType             bool
		differentSubtype          bool
		differentParentResourceID bool
		differentResourceID       bool
		differentOutput           bool
	}{
		{
			name: "Equal URNs",
		},
		{
			name:               "Different AccountIDs",
			differentAccountID: true,
		},
		{
			name:             "Different Projects",
			differentProject: true,
		},
		{
			name:                 "Different Environments",
			differentEnvironment: true,
		},
		{
			name:                 "Different Applications",
			differentApplication: true,
		},
		{
			name:          "Different Types",
			differentType: true,
		},
		{
			name:             "Different Subtypes",
			differentSubtype: true,
		},
		{
			name:                      "Different ParentResourceIDs",
			differentParentResourceID: true,
		},
		{
			name:                "Different ResourceIDs",
			differentResourceID: true,
		},
		{
			name:            "Different Outputs",
			differentOutput: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			urn1, urn2 := createURNWithToggles(
				tc.differentAccountID,
				tc.differentProject,
				tc.differentEnvironment,
				tc.differentApplication,
				tc.differentType,
				tc.differentSubtype,
				tc.differentParentResourceID,
				tc.differentResourceID,
				tc.differentOutput,
			)

			result1 := urn1.Compare(*urn2)
			result2 := urn2.Compare(*urn1)

			if result1 == 0 {
				assert.Equal(t, 0, result2)
			} else {
				assert.Equal(t, -1*result1, result2)
			}
		})
	}
}

func TestUrnPath(t *testing.T) {
	testCases := []struct {
		name     string
		urn      URN
		expected string
	}{
		{
			name: "Full URN",
			urn: URN{
				Project:     "project",
				Application: "app",
				Environment: "dev",
				ResourceID:  "resource",
			},
			expected: "project/app/dev/resource",
		},
		{
			name: "Partial URN",
			urn: URN{
				Project:     "project",
				Application: "app",
			},
			expected: "project/app",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := UrnPath(tc.urn)
			assert.NoError(t, err)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestIsOutput(t *testing.T) {
	urn := &URN{
		AccountID:        "accountid",
		Project:          "project",
		Environment:      "dev",
		Application:      "app",
		Type:             "construct",
		Subtype:          "klotho.aws.S3",
		ParentResourceID: "parent",
		ResourceID:       "resource",
		Output:           "output",
	}
	assert.True(t, urn.IsOutput())
	urn.Output = ""
	assert.False(t, urn.IsOutput())
}

func TestIsResource(t *testing.T) {
	urn := &URN{
		AccountID:        "accountid",
		Project:          "project",
		Environment:      "dev",
		Application:      "app",
		Type:             "construct",
		Subtype:          "klotho.aws.S3",
		ParentResourceID: "parent",
		ResourceID:       "resource",
	}
	assert.True(t, urn.IsResource())
	urn.ResourceID = ""
	assert.False(t, urn.IsResource())
}

func TestIsApplicationEnvironment(t *testing.T) {
	urn := &URN{
		AccountID:   "accountid",
		Project:     "project",
		Environment: "dev",
		Application: "app",
	}
	assert.True(t, urn.IsApplicationEnvironment())
	urn.Application = ""
	assert.False(t, urn.IsApplicationEnvironment())
}

func TestIsType(t *testing.T) {
	urn := &URN{
		AccountID:   "accountid",
		Project:     "project",
		Environment: "dev",
		Type:        "construct",
	}
	assert.True(t, urn.IsType())
	urn.Type = ""
	assert.False(t, urn.IsType())
}

func TestIsEnvironment(t *testing.T) {
	urn := &URN{
		AccountID:   "accountid",
		Project:     "project",
		Environment: "dev",
	}
	assert.True(t, urn.IsEnvironment())
	urn.Environment = ""
	assert.False(t, urn.IsEnvironment())
}

func TestIsProject(t *testing.T) {
	urn := &URN{
		AccountID: "accountid",
		Project:   "project",
	}
	assert.True(t, urn.IsProject())
	urn.Project = ""
	assert.False(t, urn.IsProject())
}

func TestIsAccount(t *testing.T) {
	urn := &URN{
		AccountID: "accountid",
	}
	assert.True(t, urn.IsAccount())
	urn.AccountID = ""
	assert.False(t, urn.IsAccount())
}
