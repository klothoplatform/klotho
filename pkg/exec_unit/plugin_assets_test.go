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

func Test_assetPathMatcher_ModifyPathsForAnnotatedFile(t *testing.T) {
	type testResult struct {
		include []string
		exclude []string
	}
	tests := []struct {
		name    string
		matcher assetPathMatcher
		path    string
		want    testResult
	}{
		{
			name:    "simple relative match",
			matcher: assetPathMatcher{include: []string{"file.txt"}, exclude: []string{"notfile.txt"}},
			path:    "file.txt",
			want:    testResult{include: []string{"file.txt"}, exclude: []string{"notfile.txt"}},
		},
		{
			name:    "nested relative match",
			matcher: assetPathMatcher{include: []string{"file.txt"}, exclude: []string{"notfile.txt"}},
			path:    "dir/file.txt",
			want:    testResult{include: []string{"dir/file.txt"}, exclude: []string{"dir/notfile.txt"}},
		},
		{
			name:    "simple absolute match",
			matcher: assetPathMatcher{include: []string{"/file.txt"}, exclude: []string{"/notfile.txt"}},
			path:    "file.txt",
			want:    testResult{include: []string{"file.txt"}, exclude: []string{"notfile.txt"}},
		},
		{
			name:    "nested absolute match",
			matcher: assetPathMatcher{include: []string{"/dir/file.txt"}, exclude: []string{"/dir/notfile.txt"}},
			path:    "dir/file.txt",
			want:    testResult{include: []string{"dir/file.txt"}, exclude: []string{"dir/notfile.txt"}},
		},
		{
			name:    "mix relative and absolute match",
			matcher: assetPathMatcher{include: []string{"/dir/file.txt", "other.txt"}, exclude: []string{"/dir/notfile.txt", "notother.txt"}},
			path:    "dir/file.txt",
			want:    testResult{include: []string{"dir/file.txt", "dir/other.txt"}, exclude: []string{"dir/notfile.txt", "dir/notother.txt"}},
		},
		{
			name:    "mix relative and absolute match",
			matcher: assetPathMatcher{include: []string{"/dir/file.txt", "other.txt"}, exclude: []string{"/dir/notfile.txt", "notother.txt"}},
			path:    "dir/file.txt",
			want:    testResult{include: []string{"dir/file.txt", "dir/other.txt"}, exclude: []string{"dir/notfile.txt", "dir/notother.txt"}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)

			matcher, err := NewAssetPathMatcher(tt.matcher.include, tt.matcher.exclude, tt.path)
			if !assert.NoError(err) {
				return
			}

			for _, wantPath := range tt.want.include {
				found := false
				for _, path := range matcher.include {
					if wantPath == path {
						found = true
					}
				}
				assert.True(found)
			}

			for _, wantPath := range tt.want.exclude {
				found := false
				for _, path := range matcher.exclude {
					if wantPath == path {
						found = true
					}
				}
				assert.True(found)
			}
		})
	}
}
