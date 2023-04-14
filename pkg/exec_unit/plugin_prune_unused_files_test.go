package execunit

import (
	"testing"

	"github.com/klothoplatform/klotho/pkg/annotation"
	"github.com/klothoplatform/klotho/pkg/core"
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

			testUnit := core.ExecutionUnit{AnnotationKey: core.AnnotationKey{ID: "main", Capability: annotation.ExecutionUnitCapability}, Executable: core.NewExecutable()}
			for _, path := range tt.unitFiles {
				testUnit.Add(&core.FileRef{FPath: path})
			}
			for _, path := range tt.unitResources {
				testUnit.AddResource(&core.FileRef{FPath: path})
			}
			for _, path := range tt.unitSourceFiles {
				testUnit.AddSourceFile(&core.FileRef{FPath: path})
			}
			for _, path := range tt.unitAssets {
				testUnit.AddStaticAsset(&core.FileRef{FPath: path})
			}
			p := PruneUncategorizedFiles{}
			result := core.NewConstructGraph()
			result.AddConstruct(&testUnit)
			err := p.Transform(&core.InputFiles{}, &core.FileDependencies{}, result)
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
