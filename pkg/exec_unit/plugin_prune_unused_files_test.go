package execunit

import (
	"testing"

	"github.com/klothoplatform/klotho/pkg/compiler/types"
	"github.com/klothoplatform/klotho/pkg/construct"
	"github.com/klothoplatform/klotho/pkg/io"
	"github.com/stretchr/testify/assert"
)

func TestPruneUncategorizedFiles_Transform(t *testing.T) {
	tests := []struct {
		name            string
		unitFiles       []string
		unitResources   []string
		unitSourceFiles []string
		unitAssets      []string
		expectedFiles   []string
	}{
		{
			name:            "prunes all uncategorized files from an execution unit",
			unitFiles:       []string{"source1", "source2", "resource1", "asset1", "uncategorized1", "uncategorized2"},
			unitResources:   []string{"resource1"},
			unitSourceFiles: []string{"source1", "source2"},
			unitAssets:      []string{"asset1"},
			expectedFiles:   []string{"source1", "source2", "resource1", "asset1"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)

			testUnit := types.ExecutionUnit{Name: "main", Executable: types.NewExecutable()}
			for _, path := range tt.unitFiles {
				testUnit.Add(&io.FileRef{FPath: path})
			}
			for _, path := range tt.unitResources {
				testUnit.AddResource(&io.FileRef{FPath: path})
			}
			for _, path := range tt.unitSourceFiles {
				testUnit.AddSourceFile(&io.FileRef{FPath: path})
			}
			for _, path := range tt.unitAssets {
				testUnit.AddStaticAsset(&io.FileRef{FPath: path})
			}
			p := PruneUncategorizedFiles{}
			result := construct.NewConstructGraph()
			result.AddConstruct(&testUnit)
			err := p.Transform(&types.InputFiles{}, &types.FileDependencies{}, result)
			if !assert.NoError(err) {
				return
			}

			var unitFiles []string
			for path := range testUnit.Files() {
				unitFiles = append(unitFiles, path)
			}
			assert.ElementsMatch(tt.expectedFiles, unitFiles)
		})
	}
}
