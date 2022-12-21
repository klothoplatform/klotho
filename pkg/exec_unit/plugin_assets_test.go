package execunit

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_assetPathMatcher_Matches(t *testing.T) {
	tests := []struct {
		name    string
		matcher assetPathMatcher
		path    string
		want    bool
		wantErr bool
	}{
		{
			name:    "simple include match",
			matcher: assetPathMatcher{include: []string{"file.txt"}},
			path:    "file.txt",
			want:    true,
		},
		{
			name:    "simple exclude match",
			matcher: assetPathMatcher{include: []string{"*"}, exclude: []string{"file.txt"}},
			path:    "file.txt",
			want:    false,
		},
		{
			name:    "include match with exclude non-match",
			matcher: assetPathMatcher{include: []string{"*"}, exclude: []string{"other.txt"}},
			path:    "file.txt",
			want:    true,
		},
		{
			name:    "bad include pattern",
			matcher: assetPathMatcher{include: []string{`\`}},
			path:    "file.txt",
			wantErr: true,
		},
		{
			name:    "bad exclude pattern",
			matcher: assetPathMatcher{include: []string{"*"}, exclude: []string{`\`}},
			path:    "file.txt",
			wantErr: true,
		},
		{
			name:    "include subfolder",
			matcher: assetPathMatcher{include: []string{"static/**"}},
			path:    "static/assets/index.ab213.css",
			want:    true,
		},
		// Could add more tests, but at this point it'd be testing filepath.Match which doesn't add much value.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)

			got := tt.matcher.Matches(tt.path)
			if tt.wantErr {
				assert.Error(tt.matcher.err)
				return
			}
			if !assert.NoError(tt.matcher.err) {
				return
			}
			assert.Equal(tt.want, got)
		})
	}
}
