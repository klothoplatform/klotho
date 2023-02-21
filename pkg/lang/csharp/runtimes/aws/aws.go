package aws_runtime

import (
	_ "embed"
	"fmt"
	"github.com/klothoplatform/klotho/pkg/config"
	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/lang/csharp"
	"github.com/klothoplatform/klotho/pkg/lang/csharp/csproj"
	"github.com/klothoplatform/klotho/pkg/multierr"
	"github.com/klothoplatform/klotho/pkg/provider/aws"
	"github.com/klothoplatform/klotho/pkg/runtime"
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
		APIGatewayProxyFunction string
		FunctionType            string
		StartupClass            string
	}

	qualifiedName struct {
		namespace string
		name      string
	}
)

var lambdaApiTypeClasses = map[string]qualifiedName{
	"REST": {
		namespace: "Amazon.Lambda.AspNetCoreServer",
		name:      "APIGatewayProxyFunction",
	},
	"HTTP": {
		namespace: "Amazon.Lambda.AspNetCoreServer",
		name:      "APIGatewayHttpApiV2ProxyFunction",
	},
}

//go:embed Lambda_Dockerfile.tmpl
var dockerfileLambda []byte

//go:embed Lambda_Dispatcher.cs.tmpl
var dispatcherLambda []byte

func updateCsproj(unit *core.ExecutionUnit) {
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
	var errs multierr.Error
	var err error
	var dockerFile []byte
	unitType := r.Cfg.GetResourceType(unit)
	switch unitType {
	case "lambda":
		dockerFile = dockerfileLambda
	default:
		return errors.Errorf("unsupported execution unit type: '%s'", unitType)
	}

	updateCsproj(unit)

	var projectFile *csproj.CSProjFile
	for _, file := range unit.Files() {
		if pfile, ok := file.(*csproj.CSProjFile); ok {
			projectFile = pfile
			break
		}
	}

	assembly := resolveAssemblyName(projectFile)

	exposeData, err := r.getExposeTemplateData(unit, result, deps)
	errs.Append(err)

	templateData := TemplateData{
		TemplateConfig: r.TemplateConfig,
		ExecUnitName:   unit.Name,
		CSProjFile:     projectFile.Path(),
		Expose:         exposeData,
		AssemblyName:   assembly,
	}

	if runtime.ShouldOverrideDockerfile(unit) {
		errs.Append(csharp.AddRuntimeFile(unit, templateData, "Dockerfile.tmpl", dockerFile))
	}
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

func (r *AwsRuntime) getExposeTemplateData(unit *core.ExecutionUnit, result *core.CompilationResult, deps *core.Dependencies) (ExposeTemplateData, error) {
	upstreamGateways := core.FindUpstreamGateways(unit, result, deps)

	var sgw *core.Gateway
	var sgwApiType string
	for _, gw := range upstreamGateways {
		gwApiType := r.Cfg.GetExposed(gw.Name).ApiType
		if sgw != nil {
			if sgw.DefinedIn != gw.DefinedIn || sgw.ExportVarName != gw.ExportVarName {
				return ExposeTemplateData{},
					errors.Errorf("multiple gateways cannot target different web applications in the same execution unit: [%s -> %s],[%s -> %s]",
						gw.Name, unit.Name,
						sgw.Name, unit.Name)
			}
			if sgwApiType != gwApiType {
				return ExposeTemplateData{},
					errors.Errorf("an execution unit cannot be targeted by different gateways with different API types : [%s:%s -> %s],[%s:%s -> %s]",
						gwApiType, gw.Name, unit.Name,
						sgwApiType, sgw.Name, unit.Name)
			}
		}
		sgw = gw
		sgwApiType = gwApiType
	}

	if sgw == nil {
		return ExposeTemplateData{}, nil
	}

	startupClass, err := csharp.FindASPDotnetCoreStartupClass(unit)
	if err != nil {
		return ExposeTemplateData{}, err
	}

	unitType := r.Cfg.GetExecutionUnit(unit.Name).Type

	if unitType != "lambda" {
		return ExposeTemplateData{}, fmt.Errorf("unit type \"%s\" is not supported in C# execution units", unitType)
	}

	className := lambdaApiTypeClasses[sgwApiType]

	entrypointClasses := csharp.FindSubtypes(unit, className.namespace, className.name)
	var validEntrypoints []*csharp.TypeDeclaration
	for _, h := range entrypointClasses {
		if h.IsSealed() || h.Visibility != csharp.VisibilityPublic {
			continue
		}
		validEntrypoints = append(validEntrypoints, h)
	}
	if len(validEntrypoints) > 1 {
		var names []string
		for _, h := range validEntrypoints {
			names = append(names, h.QualifiedName)
		}
		return ExposeTemplateData{}, fmt.Errorf("ambiguous user defined AWS Lamba entrypoint: more than 1 implementation provided :%s", strings.Join(names, ", "))
	}
	entrypointName := ""
	if len(validEntrypoints) == 1 {
		entrypointName = validEntrypoints[0].QualifiedName
	}

	exposeData := ExposeTemplateData{
		StartupClass:            startupClass.Class.QualifiedName,
		APIGatewayProxyFunction: entrypointName,
		FunctionType:            fmt.Sprintf("%s.%s", className.namespace, className.name),
	}

	return exposeData, nil
}
