package construct

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResourceId_UnmarshalText(t *testing.T) {
	tests := []struct {
		name    string
		str     string
		want    ResourceId
		wantErr bool
	}{
		{
			name: "full id",
			str:  "aws:subnet:my_vpc:my_subnet",
			want: ResourceId{
				Provider:  "aws",
				Type:      "subnet",
				Namespace: "my_vpc",
				Name:      "my_subnet",
			},
		},
		{
			name: "no namespace",
			str:  "aws:subnet:my_subnet",
			want: ResourceId{
				Provider: "aws",
				Type:     "subnet",
				Name:     "my_subnet",
			},
		},
		{
			name: "namespace with colon in name",
			str:  "aws:subnet:my_vpc:my_subnet:with:colons",
			want: ResourceId{
				Provider:  "aws",
				Type:      "subnet",
				Namespace: "my_vpc",
				Name:      "my_subnet:with:colons",
			},
		},
		{
			name: "no namespace with colon in name",
			str:  "aws:subnet::my_subnet:with:colons",
			want: ResourceId{
				Provider: "aws",
				Type:     "subnet",
				Name:     "my_subnet:with:colons",
			},
		},
		{
			name: "no name or namespace",
			str:  "aws:subnet",
			want: ResourceId{
				Provider: "aws",
				Type:     "subnet",
			},
		},
		{
			name: "empty is zero id",
			str:  "",
			want: ResourceId{},
		},
		{
			name:    "invalid provider",
			str:     "aws$:subnet:my_subnet",
			wantErr: true,
		},
		{
			name:    "invalid type",
			str:     "aws:subnet$:my_subnet",
			wantErr: true,
		},
		{
			name:    "invalid namespace",
			str:     "aws:subnet:my_vpc$:my_subnet",
			wantErr: true,
		},
		{
			name:    "invalid name",
			str:     "aws:subnet:my_vpc:my_subnet$",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)

			var id ResourceId
			err := id.UnmarshalText([]byte(tt.str))
			if tt.wantErr {
				assert.Error(err)
				return
			}
			if !assert.NoError(err) {
				return
			}
			assert.Equal(tt.want, id)
			// Test the round trip to make sure that String() matches exactly the input string
			assert.Equal(tt.str, id.String())
		})
	}
}
