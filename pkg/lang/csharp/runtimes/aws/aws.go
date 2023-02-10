package aws_runtime

import (
	_ "embed"
	"github.com/klothoplatform/klotho/pkg/config"
	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/lang/csharp"
	"github.com/klothoplatform/klotho/pkg/lang/csharp/csproj"
	"github.com/klothoplatform/klotho/pkg/multierr"
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
		CSProjFile   string
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

func (r *AwsRuntime) AddExecRuntimeFiles(unit *core.ExecutionUnit) error {
	var errs multierr.Error
	var err error
	var dockerFile []byte
	var startupClass *csharp.ASPDotNetCoreStartupClass
	var lambdaHandlerName string
	unitType := r.Cfg.GetResourceType(unit)
	switch unitType {
	case "lambda":
		dockerFile = dockerfileLambda

		// TODO: implement choosing the correct handler class based on the upstream gateway type
		lambdaHandlers := csharp.FindLambdaHandlerClasses(unit)
		if len(lambdaHandlers) > 0 {
			lambdaHandlerName = lambdaHandlers[0].QualifiedName
		}
		errs.Append(err)
	default:
		return errors.Errorf("unsupported execution unit type: '%s'", unitType)
	}
	startupClass, err = csharp.FindASPDotnetCoreStartupClass(unit)
	errs.Append(err)

	r.UpdateCsproj(unit)

	var projectFile *csproj.CSProjFile
	for _, file := range unit.Files() {
		if pfile, ok := file.(*csproj.CSProjFile); ok {
			projectFile = pfile
			break
		}
	}

	assembly := resolveAssemblyName(projectFile)

	startupClassName := ""
	if startupClass != nil {
		startupClassName = startupClass.Class.QualifiedName
	}

	templateData := TemplateData{
		TemplateConfig: r.TemplateConfig,
		ExecUnitName:   unit.Name,
		CSProjFile:     projectFile.Path(),
		Expose: ExposeTemplateData{
			StartupClass:            startupClassName,
			APIGatewayProxyFunction: lambdaHandlerName,
		},
		AssemblyName: assembly,
	}

	errs.Append(csharp.AddRuntimeFile(unit, templateData, "Dockerfile.tmpl", dockerFile))
	errs.Append(csharp.AddRuntimeFile(unit, templateData, "Dispatcher.cs.tmpl", dispatcherLambda))

	return errs.ErrOrNil()
}

func resolveAssemblyName(projectFile *csproj.CSProjFile) string {
	assembly, ok := projectFile.GetProperty("AssemblyName")

	if !ok {
		_, pFileName := filepath.Split(projectFile.Path())
		assembly = strings.TrimSuffix(pFileName, ".csproj")
	}
	return assembly
}
