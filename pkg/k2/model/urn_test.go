package model

import (
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
	"testing"
)

func TestParseURN(t *testing.T) {
	testCases := []struct {
		name      string
		urnString string
		expected  *URN
	}{
		{
			name:      "Project URN",
			urnString: "urn:123456790:my-project",
			expected: &URN{
				AccountID: "123456790",
				Project:   "my-project",
			},
		},
		{
			name:      "Project Environment URN",
			urnString: "urn:123456790:my-project:dev",
			expected: &URN{
				AccountID:   "123456790",
				Project:     "my-project",
				Environment: "dev",
			},
		},
		{
			name:      "Project Application URN",
			urnString: "urn:123456790:my-project::my-app",
			expected: &URN{
				AccountID:   "123456790",
				Project:     "my-project",
				Application: "my-app",
			},
		},
		{
			name:      "Project Environment Application URN",
			urnString: "urn:123456790:my-project:dev:my-app",
			expected: &URN{
				AccountID:   "123456790",
				Project:     "my-project",
				Environment: "dev",
				Application: "my-app",
			},
		},
		{
			name:      "Construct Instance URN",
			urnString: "urn:accountid:project:dev::construct/klotho.aws.S3:my-bucket",
			expected: &URN{
				AccountID:   "accountid",
				Project:     "project",
				Environment: "dev",
				Type:        "construct",
				Subtype:     "klotho.aws.S3",
				ResourceID:  "my-bucket",
			},
		},
		{
			name:      "Construct Output URN",
			urnString: "urn:accountid:project:dev::construct/klotho.aws.S3:my-bucket:bucketName",
			expected: &URN{
				AccountID:   "accountid",
				Project:     "project",
				Environment: "dev",
				Type:        "construct",
				Subtype:     "klotho.aws.S3",
				ResourceID:  "my-bucket",
				Output:      "bucketName",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := ParseURN(tc.urnString)
			assert.NoError(t, err)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestString(t *testing.T) {
	testCases := []struct {
		name     string
		urn      *URN
		expected string
	}{
		{
			name: "Project URN",
			urn: &URN{
				AccountID: "123456790",
				Project:   "my-project",
			},
			expected: "urn:123456790:my-project",
		},
		{
			name: "Project Environment URN",
			urn: &URN{
				AccountID:   "123456790",
				Project:     "my-project",
				Environment: "dev",
			},
			expected: "urn:123456790:my-project:dev",
		},
		{
			name: "Project Application URN",
			urn: &URN{
				AccountID:   "123456790",
				Project:     "my-project",
				Application: "my-app",
			},
			expected: "urn:123456790:my-project::my-app",
		},
		{
			name: "Project Environment Application URN",
			urn: &URN{
				AccountID:   "123456790",
				Project:     "my-project",
				Environment: "dev",
				Application: "my-app",
			},
			expected: "urn:123456790:my-project:dev:my-app",
		},
		{
			name: "Construct Instance URN",
			urn: &URN{
				AccountID:   "accountid",
				Project:     "project",
				Environment: "dev",
				Type:        "construct",
				Subtype:     "klotho.aws.S3",
				ResourceID:  "my-bucket",
			},
			expected: "urn:accountid:project:dev::construct/klotho.aws.S3:my-bucket",
		},
		{
			name: "Construct Output URN",
			urn: &URN{
				AccountID:   "accountid",
				Project:     "project",
				Environment: "dev",
				Type:        "construct",
				Subtype:     "klotho.aws.S3",
				ResourceID:  "my-bucket",
				Output:      "bucketName",
			},
			expected: "urn:accountid:project:dev::construct/klotho.aws.S3:my-bucket:bucketName",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := tc.urn.String()
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestMarshalYAML(t *testing.T) {
	testCases := []struct {
		name     string
		urn      *URN
		expected string
	}{
		{
			name: "Project URN",
			urn: &URN{
				AccountID: "123456790",
				Project:   "my-project",
			},
			expected: "urn:123456790:my-project",
		},
		{
			name: "Project Environment URN",
			urn: &URN{
				AccountID:   "123456790",
				Project:     "my-project",
				Environment: "dev",
			},
			expected: "urn:123456790:my-project:dev",
		},
		{
			name: "Project Application URN",
			urn: &URN{
				AccountID:   "123456790",
				Project:     "my-project",
				Application: "my-app",
			},
			expected: "urn:123456790:my-project::my-app",
		},
		{
			name: "Project Environment Application URN",
			urn: &URN{
				AccountID:   "123456790",
				Project:     "my-project",
				Environment: "dev",
				Application: "my-app",
			},
			expected: "urn:123456790:my-project:dev:my-app",
		},
		{
			name: "Construct Instance URN",
			urn: &URN{
				AccountID:   "accountid",
				Project:     "project",
				Environment: "dev",
				Type:        "construct",
				Subtype:     "klotho.aws.S3",
				ResourceID:  "my-bucket",
			},
			expected: "urn:accountid:project:dev::construct/klotho.aws.S3:my-bucket",
		},
		{
			name: "Construct Output URN",
			urn: &URN{
				AccountID:   "accountid",
				Project:     "project",
				Environment: "dev",
				Type:        "construct",
				Subtype:     "klotho.aws.S3",
				ResourceID:  "my-bucket",
				Output:      "bucketName",
			},
			expected: "urn:accountid:project:dev::construct/klotho.aws.S3:my-bucket:bucketName",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := tc.urn.MarshalYAML()
			assert.NoError(t, err)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestUnmarshalYAML(t *testing.T) {
	testCases := []struct {
		name       string
		yamlString string
		expected   *URN
	}{
		{
			name:       "Project URN",
			yamlString: "urn:123456790:my-project",
			expected: &URN{
				AccountID: "123456790",
				Project:   "my-project",
			},
		},
		{
			name:       "Project Environment URN",
			yamlString: "urn:123456790:my-project:dev",
			expected: &URN{
				AccountID:   "123456790",
				Project:     "my-project",
				Environment: "dev",
			},
		},
		{
			name:       "Project Application URN",
			yamlString: "urn:123456790:my-project::my-app",
			expected: &URN{
				AccountID:   "123456790",
				Project:     "my-project",
				Application: "my-app",
			},
		},
		{
			name:       "Project Environment Application URN",
			yamlString: "urn:123456790:my-project:dev:my-app",
			expected: &URN{
				AccountID:   "123456790",
				Project:     "my-project",
				Environment: "dev",
				Application: "my-app",
			},
		},
		{
			name:       "Construct Instance URN",
			yamlString: "urn:accountid:project:dev::construct/klotho.aws.S3:my-bucket",
			expected: &URN{
				AccountID:   "accountid",
				Project:     "project",
				Environment: "dev",
				Type:        "construct",
				Subtype:     "klotho.aws.S3",
				ResourceID:  "my-bucket",
			},
		},
		{
			name:       "Construct Output URN",
			yamlString: "urn:accountid:project:dev::construct/klotho.aws.S3:my-bucket:bucketName",
			expected: &URN{
				AccountID:   "accountid",
				Project:     "project",
				Environment: "dev",
				Type:        "construct",
				Subtype:     "klotho.aws.S3",
				ResourceID:  "my-bucket",
				Output:      "bucketName",
			},
		},
		{
			name:       "Container Instance URN",
			yamlString: "urn:accountid:my-project:dev::construct/klotho.aws.Container:my-container",
			expected: &URN{
				AccountID:   "accountid",
				Project:     "my-project",
				Environment: "dev",
				Type:        "construct",
				Subtype:     "klotho.aws.Container",
				ResourceID:  "my-container",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var urn URN
			err := yaml.Unmarshal([]byte(tc.yamlString), &urn)
			assert.NoError(t, err)
			assert.Equal(t, tc.expected, &urn)
		})
	}
}

func TestMarshalYAML2(t *testing.T) {
	testCases := []struct {
		name     string
		urn      string
		expected string
	}{
		{
			name:     "Container Instance URN",
			urn:      "urn:accountid:project:dev::construct/klotho.aws.Container:my-container",
			expected: "urn:accountid:project:dev::construct/klotho.aws.Container:my-container\n",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			urn, err := ParseURN(tc.urn)
			var urn2 URN

			urn2 = *urn

			if assert.NoError(t, err); err != nil {
				return
			}
			result, err := yaml.Marshal(urn2)
			assert.NoError(t, err)
			assert.Equal(t, tc.expected, string(result))
		})
	}
}
