package types

import (
	"testing"

	"github.com/klothoplatform/klotho/pkg/async"
	"github.com/klothoplatform/klotho/pkg/io"
	"github.com/stretchr/testify/assert"
)

func TestInputFiles_Add(t *testing.T) {

	tests := []struct {
		name              string
		files             []io.File
		expectedFilePaths []string
	}{
		{
			name: "added files are present in InputFiles",
			files: []io.File{
				&io.FileRef{FPath: "file1.js"},
				&io.FileRef{FPath: "dir/file2.js"},
			},
			expectedFilePaths: []string{
				"file1.js",
				"dir/file2.js",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			fg := &InputFiles{}
			for _, file := range tt.files {
				fg.Add(file)
			}
			assert.Equal(len(tt.expectedFilePaths), (*async.ConcurrentMap[string, io.File])(fg).Len())
			for _, filePath := range tt.expectedFilePaths {
				_, found := (*async.ConcurrentMap[string, io.File])(fg).Get(filePath)
				assert.True(found, "Contains file: %s", filePath)
			}

		})
	}
}

func TestInputFiles_Files(t *testing.T) {

	tests := []struct {
		name  string
		want  []string
		files []io.File
	}{
		{
			name: "returns all files in InputFiles",
			want: []string{"file1.js", "dir/file2.js"},
			files: []io.File{
				&io.FileRef{FPath: "file1.js"},
				&io.FileRef{FPath: "dir/file2.js"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)
			fg := &InputFiles{}
			for _, file := range tt.files {
				fg.Add(file)
			}
			fgFiles := fg.Files()
			assert.Equal(len(tt.want), len(fgFiles))
			for path, file := range fgFiles {
				assert.Contains(tt.want, path)
				assert.NotNil(file)
			}
		})
	}
}
