package csharp

import (
	"fmt"
	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/filter"
	sitter "github.com/smacker/go-tree-sitter"
)

type ASPDotNetCoreStartupClass struct {
	Class                   TypeDeclaration
	ConfigureMethod         MethodDeclaration
	ConfigureServicesMethod MethodDeclaration
}

func FindASPDotnetCoreStartupClass(unit *core.ExecutionUnit) (*ASPDotNetCoreStartupClass, error) {
	var startupClass *ASPDotNetCoreStartupClass
	declarers := unit.GetDeclaringFiles()
	if declarers == nil {
		for _, csFile := range unit.FilesOfLang(CSharp) {
			types := FindDeclarationsInFile[*TypeDeclaration](csFile).Declarations()
			for _, t := range types {
				if cls, found := getDotnetCoreStartupClass(t.Node); found {
					if startupClass != nil {
						return nil, fmt.Errorf("multiple ASP.NET Core startup classes found in execution unit [%s] <- [%s, %s]", unit.Name, startupClass.Class.Name, cls.Class.Name)
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
					return nil, fmt.Errorf("multiple ASP.NET Core startup classes found in execution unit [%s] <- [%s, %s]", unit.Name, startupClass.Class.Name, cls.Class.Name)
				} else {
					startupClass = &cls
				}
			}
		}
	}
	return startupClass, nil
}

func getDotnetCoreStartupClass(classNode *sitter.Node) (ASPDotNetCoreStartupClass, bool) {
	classDeclaration, found := FindDeclarationAtNode[*TypeDeclaration](classNode)
	if !found || classDeclaration.Visibility == VisibilityPrivate || classDeclaration.Kind != DeclarationKindClass {
		return ASPDotNetCoreStartupClass{}, false
	}
	methods := FindDeclarationsAtNode[*MethodDeclaration](classNode).Declarations()
	configureMethods := filter.NewSimpleFilter[*MethodDeclaration](func(md *MethodDeclaration) bool {
		return md.Name == "Configure" &&
			md.Visibility == VisibilityPublic &&
			md.ReturnType == "void" &&
			!md.IsStatic &&
			!md.IsAbstract &&
			len(md.Parameters) == 2 &&
			IsValidTypeName(md.Parameters[0].TypeNode, "Microsoft.AspNetCore.Builder", "IApplicationBuilder") &&
			IsValidTypeName(md.Parameters[1].TypeNode, "Microsoft.AspNetCore.Hosting", "IWebHostEnvironment")
	}).Apply(methods...)
	if len(configureMethods) != 1 {
		return ASPDotNetCoreStartupClass{}, false
	}

	startupClass := ASPDotNetCoreStartupClass{
		Class:           *classDeclaration,
		ConfigureMethod: *configureMethods[0],
	}

	configureServicesMethods := filter.NewSimpleFilter[*MethodDeclaration](func(md *MethodDeclaration) bool {
		return md.Name == "ConfigureServices" &&
			md.Visibility == VisibilityPublic &&
			md.ReturnType == "void" &&
			!md.IsStatic &&
			!md.IsAbstract &&
			len(md.Parameters) == 1 &&
			IsValidTypeName(md.Parameters[0].TypeNode, "Microsoft.Extensions.DependencyInjection", "IServiceCollection")
	}).Apply(methods...)

	if len(configureServicesMethods) == 1 {
		startupClass.ConfigureServicesMethod = *configureServicesMethods[0]
	}

	return startupClass, true
}
