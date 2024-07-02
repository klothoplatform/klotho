package model

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestUUIDUnmarshalYAML(t *testing.T) {
	testCases := []struct {
		name          string
		input         string
		expectedUUID  uuid.UUID
		expectedError bool
	}{
		{
			name:          "Valid UUID",
			input:         "123e4567-e89b-12d3-a456-426614174000",
			expectedUUID:  uuid.MustParse("123e4567-e89b-12d3-a456-426614174000"),
			expectedError: false,
		},
		{
			name:          "Invalid UUID Format",
			input:         "invalid-uuid-format",
			expectedUUID:  uuid.UUID{},
			expectedError: true,
		},
		{
			name:          "Empty UUID String",
			input:         "",
			expectedUUID:  uuid.UUID{},
			expectedError: true,
		},
		{
			name:          "Nil Unmarshal Function",
			input:         "123e4567-e89b-12d3-a456-426614174000",
			expectedUUID:  uuid.UUID{},
			expectedError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			u := &UUID{}
			err := u.UnmarshalYAML(func(v interface{}) error {
				if tc.name == "Nil Unmarshal Function" {
					return nil // Simulate a nil unmarshal function
				}
				*v.(*string) = tc.input
				return nil
			})
			if tc.expectedError {
				assert.Error(t, err, "Expected an error but got nil")
			} else {
				assert.NoError(t, err, "Expected no error but got an error")
				assert.Equal(t, tc.expectedUUID, u.UUID, "Expected UUID to be %s, got %s", tc.expectedUUID, u.UUID)
			}
		})
	}
}
