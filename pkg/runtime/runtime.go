package runtime

import (
	"github.com/klothoplatform/klotho/pkg/annotation"
	"github.com/klothoplatform/klotho/pkg/compiler/types"
	"github.com/klothoplatform/klotho/pkg/lang/dockerfile"
)

func ShouldOverrideDockerfile(unit *types.ExecutionUnit) bool {

	for _, f := range unit.Files() {
		ast, ok := f.(*types.SourceFile)
		if !ok {
			continue
		}
		dFile, ok := dockerfile.DockerfileLang.CastFile(ast)
		if !ok {
			continue
		}
		caps := dFile.Annotations()
		for _, annot := range caps {
			cap := annot.Capability
			if cap.ID == unit.Name && cap.Name == annotation.ExecutionUnitCapability {
				unit.DockerfilePath = f.Path()
				return false
			}
		}
	}
	unit.DockerfilePath = "Dockerfile"
	return true
}
