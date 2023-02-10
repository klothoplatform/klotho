package csharp

import "github.com/klothoplatform/klotho/pkg/core"

func FindLambdaHandlerClasses(unit *core.ExecutionUnit) []*TypeDeclaration {
	var handlerClasses []*TypeDeclaration
	for _, csFile := range unit.FilesOfLang(CSharp) {
		types := FindDeclarationsInFile[*TypeDeclaration](csFile).Declarations()
		for _, t := range types {
			if t.IsSealed || t.Visibility != VisibilityPublic {
				continue
			}
			for _, bNode := range t.Bases {
				if IsValidTypeName(bNode, "Amazon.Lambda.AspNetCoreServer", "APIGatewayProxyFunction") ||
					IsValidTypeName(bNode, "Amazon.Lambda.AspNetCoreServer", "APIGatewayHttpApiV2ProxyFunction") {
					handlerClasses = append(handlerClasses, t)
					break
				}
			}

		}
	}
	return handlerClasses
}
