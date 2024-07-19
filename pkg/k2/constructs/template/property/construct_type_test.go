package property

import (
	"testing"

	"github.com/klothoplatform/klotho/pkg/k2/model"
	"github.com/stretchr/testify/assert"
)

func TestConstructType_FromURN(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected ConstructType
		wantErr  bool
	}{
		{
			name:     "Valid URN",
			input:    "urn:accountid:project:dev::construct/package.name",
			expected: ConstructType{Package: "package", Name: "name"},
			wantErr:  false,
		},
		{
			name:    "Invalid URN type",
			input:   "urn:accountid:project:dev::other/package.name",
			wantErr: true,
		},
		{
			name:    "Invalid URN format",
			input:   "urn:accountid:project:dev::construct/invalid",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var ctId ConstructType
			urn, err := model.ParseURN(tt.input)
			if assert.NoError(t, err) {
				err = ctId.FromURN(*urn)
				if tt.wantErr {
					assert.Error(t, err)
				} else {
					assert.NoError(t, err)
					assert.Equal(t, tt.expected, ctId)
				}
			}
		})
	}
}
