package aws_runtime

import (
	_ "embed"

	"github.com/klothoplatform/klotho/pkg/config"
	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/lang/golang"
	"github.com/klothoplatform/klotho/pkg/provider/aws"
	"github.com/pkg/errors"
)

type (
	AwsRuntime struct {
		TemplateConfig aws.TemplateConfig
		Cfg            *config.Application
	}

	TemplateData struct {
		aws.TemplateConfig
		ExecUnitName string
		Expose       ExposeTemplateData
		MainModule   string
	}

	ExposeTemplateData struct {
		ExportedAppVar string
		AppModule      string
	}
)

//go:embed Lambda_Dockerfile
var dockerfileLambda []byte

func (r *AwsRuntime) AddExecRuntimeFiles(unit *core.ExecutionUnit, result *core.CompilationResult, deps *core.Dependencies) error {
	var DockerFile []byte
	unitType := r.Cfg.GetResourceType(unit)
	switch unitType {
	case "lambda":
		DockerFile = dockerfileLambda
	default:
		return errors.Errorf("unsupported execution unit type: '%s'", unitType)
	}

	templateData := TemplateData{
		TemplateConfig: r.TemplateConfig,
		ExecUnitName:   unit.Name,
	}

	err := golang.AddRuntimeFile(unit, templateData, "Dockerfile", DockerFile)
	if err != nil {
		return err
	}

	return err
}
