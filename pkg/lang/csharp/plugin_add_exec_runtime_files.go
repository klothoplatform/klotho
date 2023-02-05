package csharp

import (
	"fmt"
	"github.com/klothoplatform/klotho/pkg/config"
	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/filter"
	"github.com/klothoplatform/klotho/pkg/multierr"
	"github.com/smacker/go-tree-sitter"
)

type (
	AddExecRuntimeFiles struct {
		runtime Runtime
		cfg     *config.Application
	}
)

func (p *AddExecRuntimeFiles) Name() string { return "AddExecRuntimeFiles:CSharp" }

func (p *AddExecRuntimeFiles) Transform(result *core.CompilationResult, deps *core.Dependencies) error {
	var errs multierr.Error
	for _, res := range result.Resources() {
		unit, ok := res.(*core.ExecutionUnit)
		if !(ok && unit.HasSourceFilesFor(CSharp)) {
			continue
		}

		var startupClass *DotNetCoreStartupClass
		declarers := unit.GetDeclaringFiles()
		if declarers == nil {
			for _, csFile := range unit.FilesOfLang(CSharp) {
				types := FindTypeDeclarationsInFile(csFile).Types()
				for _, t := range types {
					if cls, found := getDotnetCoreStartupClass(t.Node); found {
						if startupClass != nil {
							errs.Append(fmt.Errorf("multiple ASP.net Core startup classes found in execution unit [%s] <- [%s, %s]", unit.Name, startupClass.Class.Name, cls.Class.Name))
						} else {
							startupClass = &cls
						}
					}
				}
			}
		}
		for _, declarer := range declarers {
			execUnitAnnotations := filter.NewSimpleFilter[*core.Annotation](func(a *core.Annotation) bool {
				return a.Capability.Name == "execution_unit" &&
					a.Capability.ID == unit.Name
			}).Apply(declarer.Annotations().InSourceOrder()...)

			if len(execUnitAnnotations) == 0 {
				continue
			}

			for _, annotation := range execUnitAnnotations {
				if cls, found := getDotnetCoreStartupClass(annotation.Node); found {
					if startupClass != nil {
						errs.Append(fmt.Errorf("multiple ASP.net Core startup classes found in execution unit [%s] <- [%s, %s]", unit.Name, startupClass.Class.Name, cls.Class.Name))
					} else {
						startupClass = &cls
					}
				}
			}
		}

		lambdaHandlerClasses := findLambdaHandlerClasses(unit)
		if len(lambdaHandlerClasses) > 1 {
			errs.Append(fmt.Errorf("multiple lambda handler classes detected in execution unit: %s", unit.Name))
			break
		}
		lambdaHandlerClass := ""
		if len(lambdaHandlerClasses) == 1 {
			lambdaHandlerClass = lambdaHandlerClasses[0].QualifiedName
		}

		errs.Append(p.runtime.AddExecRuntimeFiles(unit, startupClass, lambdaHandlerClass))
	}
	return errs.ErrOrNil()
}

// lambdaEntrypointClasses is a mapping between valid lambda entrypoint base classes and their fully qualified names
var lambdaEntrypointClasses = map[string]string{
	"Amazon.Lambda.AspNetCoreServer.APIGatewayProxyFunction": "",
	"APIGatewayProxyFunction":                                "Amazon.Lambda.AspNetCoreServer",
}

func findLambdaHandlerClasses(unit *core.ExecutionUnit) []TypeDeclaration {
	var handlerClasses []TypeDeclaration
	for _, csFile := range unit.FilesOfLang(CSharp) {
		importedNamespaces := FindImportsInFile(csFile)
		types := FindTypeDeclarationsInFile(csFile).Types()
		for _, t := range types {
			handlerBases := filter.NewSimpleFilter[string](func(b string) bool {
				cls, found := lambdaEntrypointClasses[b]
				return found && (len(cls) == 0 || importedNamespaces[b] != nil)
			}).Apply(t.Bases...)
			if len(handlerBases) == 0 {
				continue
			}
			if t.IsSealed || t.Visibility != VisibilityPublic {
				continue
			}
			handlerClasses = append(handlerClasses, t)
		}
	}
	return handlerClasses
}

func getDotnetCoreStartupClass(classNode *sitter.Node) (DotNetCoreStartupClass, bool) {
	classDeclaration, found := FindTypeDeclarationAtNode(classNode)
	if !found || classDeclaration.Visibility == VisibilityPrivate || classDeclaration.Kind != DeclarationKindClass {
		return DotNetCoreStartupClass{}, false
	}
	methods := FindMethodDeclarationsAtNode(classNode).Methods()
	configureMethods := filter.NewSimpleFilter[MethodDeclaration](func(md MethodDeclaration) bool {
		return md.Name == "Configure" &&
			md.Visibility == VisibilityPublic &&
			md.ReturnType == "void" &&
			md.IsStatic == false &&
			md.IsAbstract == false &&
			len(md.Parameters) == 2 &&
			(md.Parameters[0].Type == "IApplicationBuilder" || md.Parameters[0].Type == "Microsoft.AspNetCore.Builder.IApplicationBuilder") &&
			(md.Parameters[1].Type == "IWebHostEnvironment" || md.Parameters[0].Type == "Microsoft.AspNetCore.Hosting.IWebHostEnvironment")
	}).Apply(methods...)
	if len(configureMethods) != 1 {
		return DotNetCoreStartupClass{}, false
	}

	startupClass := DotNetCoreStartupClass{
		FilePath:        "",
		Class:           classDeclaration,
		ConfigureMethod: configureMethods[0],
	}

	configureServicesMethods := filter.NewSimpleFilter[MethodDeclaration](func(md MethodDeclaration) bool {
		return md.Name == "ConfigureServices" &&
			md.Visibility == VisibilityPublic &&
			md.ReturnType == "void" &&
			md.IsStatic == false &&
			md.IsAbstract == false &&
			len(md.Parameters) == 1 &&
			(md.Parameters[0].Type == "IServiceCollection" || md.Parameters[0].Type == "Microsoft.Extensions.DependencyInjection.IServiceCollection")
	}).Apply(methods...)

	if len(configureServicesMethods) == 1 {
		startupClass.ConfigureServicesMethod = configureServicesMethods[0]
	}

	return startupClass, true
}

type DotNetCoreStartupClass struct {
	FilePath                string
	Class                   TypeDeclaration
	ConfigureMethod         MethodDeclaration
	ConfigureServicesMethod MethodDeclaration
}
