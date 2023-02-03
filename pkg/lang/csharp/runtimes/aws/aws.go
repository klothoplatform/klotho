package aws_runtime

import (
	_ "embed"
	"github.com/klothoplatform/klotho/pkg/config"
	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/lang/csharp"
	"github.com/klothoplatform/klotho/pkg/lang/csharp/csproj"
	"github.com/klothoplatform/klotho/pkg/provider/aws"
	"github.com/pkg/errors"
	"path/filepath"
	"strings"
)

//go:generate ./compile_template.sh Lambda_Dispatcher

type (
	AwsRuntime struct {
		TemplateConfig aws.TemplateConfig
		Cfg            *config.Application
	}

	TemplateData struct {
		aws.TemplateConfig
		ExecUnitName string
		Expose       ExposeTemplateData
		AssemblyName string
	}

	ExposeTemplateData struct {
		StartupClass            string
		APIGatewayProxyFunction string
	}
)

//go:embed Lambda_Dockerfile.tmpl
var dockerfileLambda []byte

//go:embed Lambda_Dispatcher.cs.tmpl
var dispatcherLambda []byte

func (r *AwsRuntime) UpdateCsproj(unit *core.ExecutionUnit) {

	var projectFile *csproj.CSProjFile
	for _, file := range unit.Files() {
		if pfile, ok := file.(*csproj.CSProjFile); ok {
			projectFile = pfile
			break
		}
	}

	projectFile.AddProperty("OutDir", "klotho_bin")

}

func (r *AwsRuntime) AddExecRuntimeFiles(unit *core.ExecutionUnit, result *core.CompilationResult, deps *core.Dependencies) error {
	var DockerFile []byte
	unitType := r.Cfg.GetResourceType(unit)
	switch unitType {
	case "lambda":
		DockerFile = dockerfileLambda
	default:
		return errors.Errorf("unsupported execution unit type: '%s'", unitType)
	}

	r.UpdateCsproj(unit)

	var projectFile *csproj.CSProjFile
	for _, file := range unit.Files() {
		if pfile, ok := file.(*csproj.CSProjFile); ok {
			projectFile = pfile
			break
		}
	}

	assembly, ok := projectFile.GetProperty("AssemblyName")

	if !ok {
		_, pFileName := filepath.Split(projectFile.Path())
		assembly = strings.TrimSuffix(pFileName, ".csproj")
	}

	templateData := TemplateData{
		TemplateConfig: r.TemplateConfig,
		ExecUnitName:   unit.Name,
		Expose: ExposeTemplateData{
			StartupClass:            "SampleApp.Startup",
			APIGatewayProxyFunction: "SampleApp.LambdaEntryPoint",
		},
		AssemblyName: assembly,
	}

	err := csharp.AddRuntimeFile(unit, templateData, "Dockerfile.tmpl", DockerFile)
	if err != nil {
		return err
	}
	err = csharp.AddRuntimeFile(unit, templateData, "Dispatcher.cs.tmpl", dispatcherLambda)
	if err != nil {
		return err
	}

	return err
}
