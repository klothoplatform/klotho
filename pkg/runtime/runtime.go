package runtime

import (
	"path/filepath"

	"github.com/klothoplatform/klotho/pkg/annotation"
	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/lang/dockerfile"
)

func ShouldOverrideDockerfile(unit *core.ExecutionUnit) bool {

	for _, f := range unit.Files() {
		ast, ok := f.(*core.SourceFile)
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
				unit.DockerfilePath = filepath.Dir(f.Path())
				return false
			}
		}
	}
	return true
}
