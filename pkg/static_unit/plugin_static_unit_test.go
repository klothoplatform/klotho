package staticunit

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_assetPathMatcher_Matches(t *testing.T) {
	tests := []struct {
		name    string
		matcher staticAssetPathMatcher
		path    string
		want    []bool
		wantErr bool
	}{
		{
			name:    "simple staticFiles match",
			matcher: staticAssetPathMatcher{staticFiles: []string{"file.txt"}},
			path:    "file.txt",
			want:    []bool{true, false},
		},
		{
			name:    "simple sharedFiles match",
			matcher: staticAssetPathMatcher{sharedFiles: []string{"file.txt"}},
			path:    "file.txt",
			want:    []bool{false, true},
		},
		{
			name:    "staticFiles match with sharedFiles non-match",
			matcher: staticAssetPathMatcher{staticFiles: []string{"*"}, sharedFiles: []string{"other.txt"}},
			path:    "file.txt",
			want:    []bool{true, false},
		},
		{
			name:    "bad staticFiles pattern",
			matcher: staticAssetPathMatcher{staticFiles: []string{`\`}},
			path:    "file.txt",
			want:    []bool{false, false},
			wantErr: true,
		},
		{
			name:    "bad sharedFiles pattern",
			matcher: staticAssetPathMatcher{staticFiles: []string{"*"}, sharedFiles: []string{`\`}},
			path:    "file.txt",
			want:    []bool{false, false},
			wantErr: true,
		},
		{
			name:    "staticFiles subfolder",
			matcher: staticAssetPathMatcher{staticFiles: []string{"static/**"}},
			path:    "static/assets/index.ab213.css",
			want:    []bool{true, false},
		},
		{
			name:    "sharedFiles subfolder",
			matcher: staticAssetPathMatcher{sharedFiles: []string{"static/**"}},
			path:    "static/assets/index.ab213.css",
			want:    []bool{false, true},
		},
		{
			name:    "multiple static files one matches",
			matcher: staticAssetPathMatcher{staticFiles: []string{"js/bundle.js", "ts/*", "static/**", "random/random.js"}},
			path:    "static/assets/index.ab213.css",
			want:    []bool{true, false},
		}, {
			name:    "multiple shared files one matches",
			matcher: staticAssetPathMatcher{sharedFiles: []string{"js/bundle.js", "ts/*", "static/**", "random/random.js"}},
			path:    "static/assets/index.ab213.css",
			want:    []bool{false, true},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)

			static, shared := tt.matcher.Matches(tt.path)
			result := []bool{static, shared}
			if tt.wantErr {
				assert.Error(tt.matcher.err)
			}
			assert.Equal(tt.want, result)
		})
	}
}
