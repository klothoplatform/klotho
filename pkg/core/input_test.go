package core

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestInputFiles_Add(t *testing.T) {

	tests := []struct {
		name              string
		files             []File
		expectedFilePaths []string
	}{
		{
			name: "added files are present in InputFiles",
			files: []File{
				&FileRef{FPath: "file1.js"},
				&FileRef{FPath: "dir/file2.js"},
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
			assert.Equal(len(tt.expectedFilePaths), (*ConcurrentMap[string, File])(fg).Len())
			for _, filePath := range tt.expectedFilePaths {
				_, found := (*ConcurrentMap[string, File])(fg).Get(filePath)
				assert.True(found, "Contains file: %s", filePath)
			}

		})
	}
}

func TestInputFiles_Files(t *testing.T) {

	tests := []struct {
		name  string
		want  []string
		files []File
	}{
		{
			name: "returns all files in InputFiles",
			want: []string{"file1.js", "dir/file2.js"},
			files: []File{
				&FileRef{FPath: "file1.js"},
				&FileRef{FPath: "dir/file2.js"},
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
