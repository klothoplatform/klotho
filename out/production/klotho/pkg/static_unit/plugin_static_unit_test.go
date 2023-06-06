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

func Test_assetPathMatcher_ModifyPathsForAnnotatedFile(t *testing.T) {
	type testResult struct {
		include []string
		exclude []string
	}
	tests := []struct {
		name    string
		matcher staticAssetPathMatcher
		path    string
		want    testResult
	}{
		{
			name:    "simple relative match",
			matcher: staticAssetPathMatcher{staticFiles: []string{"file.txt"}, sharedFiles: []string{"notfile.txt"}},
			path:    "file.txt",
			want:    testResult{include: []string{"file.txt"}, exclude: []string{"notfile.txt"}},
		},
		{
			name:    "nested relative match",
			matcher: staticAssetPathMatcher{staticFiles: []string{"file.txt"}, sharedFiles: []string{"notfile.txt"}},
			path:    "dir/file.txt",
			want:    testResult{include: []string{"dir/file.txt"}, exclude: []string{"dir/notfile.txt"}},
		},
		{
			name:    "simple absolute match",
			matcher: staticAssetPathMatcher{staticFiles: []string{"/file.txt"}, sharedFiles: []string{"/notfile.txt"}},
			path:    "file.txt",
			want:    testResult{include: []string{"file.txt"}, exclude: []string{"notfile.txt"}},
		},
		{
			name:    "nested absolute match",
			matcher: staticAssetPathMatcher{staticFiles: []string{"/dir/file.txt"}, sharedFiles: []string{"/dir/notfile.txt"}},
			path:    "dir/file.txt",
			want:    testResult{include: []string{"dir/file.txt"}, exclude: []string{"dir/notfile.txt"}},
		},
		{
			name:    "mix relative and absolute match",
			matcher: staticAssetPathMatcher{staticFiles: []string{"/dir/file.txt", "other.txt"}, sharedFiles: []string{"/dir/notfile.txt", "notother.txt"}},
			path:    "dir/file.txt",
			want:    testResult{include: []string{"dir/file.txt", "dir/other.txt"}, exclude: []string{"dir/notfile.txt", "dir/notother.txt"}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)

			err := tt.matcher.ModifyPathsForAnnotatedFile(tt.path)
			if !assert.NoError(err) {
				return
			}

			for _, wantPath := range tt.want.include {
				found := false
				for _, path := range tt.matcher.staticFiles {
					if wantPath == path {
						found = true
					}
				}
				assert.True(found)
			}

			for _, wantPath := range tt.want.exclude {
				found := false
				for _, path := range tt.matcher.sharedFiles {
					if wantPath == path {
						found = true
					}
				}
				assert.True(found)
			}
		})
	}
}
