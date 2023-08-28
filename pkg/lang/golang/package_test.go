package golang

import (
	"sort"
	"strings"
	"testing"

	"github.com/klothoplatform/klotho/pkg/compiler/types"
	"github.com/stretchr/testify/assert"
)

func Test_findFilesForPackage(t *testing.T) {
	tests := []struct {
		name    string
		sources map[string]string
		pkgName string
		want    []string
	}{
		{
			name: "Single file correct package",
			sources: map[string]string{
				"file1.go": `package test`,
			},
			pkgName: "test",
			want:    []string{"file1.go"},
		},
		{
			name: "Multiple files correct package",
			sources: map[string]string{
				"file1.go": `package test`,
				"file2.go": `package test`,
				"file3.go": `package test`,
			},
			pkgName: "test",
			want:    []string{"file1.go", "file2.go", "file3.go"},
		},
		{
			name: "Multiple files with different packages",
			sources: map[string]string{
				"file1.go": `package test`,
				"file2.go": `package wrong`,
				"file3.go": `package wrong2`,
			},
			pkgName: "test",
			want:    []string{"file1.go"},
		},
		{
			name: "No files with correct packages",
			sources: map[string]string{
				"file1.go": `package wrong`,
				"file2.go": `package wrong`,
				"file3.go": `package wrong`,
			},
			pkgName: "test",
			want:    []string{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)

			unit := &types.ExecutionUnit{Name: "testUnit", Executable: types.NewExecutable()}

			for path, src := range tt.sources {
				f, err := types.NewSourceFile(path, strings.NewReader(src), Language)
				if !assert.NoError(err) {
					return
				}
				unit.AddSourceFile(f)
			}

			foundFiles := FindFilesForPackageName(unit, tt.pkgName)
			var filePaths = make([]string, 0)
			for _, f := range foundFiles {
				filePaths = append(filePaths, f.Path())
			}
			sort.Strings(filePaths)
			assert.Equal(tt.want, filePaths)
		})
	}
}
