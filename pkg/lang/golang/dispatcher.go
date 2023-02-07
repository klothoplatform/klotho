package golang

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/pkg/errors"
	sitter "github.com/smacker/go-tree-sitter"
)

var commentRegex = regexp.MustCompile(`(?m)^(\s*)`)

// UpdateListenWithHandlerCode takes the old http.ListenAndServe string and comments it out. Then it appends the
// the lambda handler code right after. Dependencies are not added at this stage
func UpdateListenWithHandlerCode(oldFileContent string, nodeToComment string, appName string) string {
	if len(nodeToComment) == 0 {
		return oldFileContent
	}

	newFileContent := oldFileContent

	oldNodeContent := nodeToComment
	newNodeContent := commentRegex.ReplaceAllString(oldNodeContent, "// $1")

	//TODO: investigate correctly indenting code
	dispatcherCode := fmt.Sprintf(`
	// Begin - Added by Klotho
	chiLambda := chiadapter.New(%s)
	handler := func(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
		return chiLambda.ProxyWithContext(ctx, req)
	}
	lambda.StartWithContext(context.Background(), handler)
	//End - Added by Klotho`, appName)

	newNodeContent = newNodeContent + dispatcherCode
	newFileContent = strings.ReplaceAll(
		newFileContent,
		oldNodeContent,
		newNodeContent,
	)

	return newFileContent
}

func UpdateImportWithHandlerRequirements(oldFileContent string, imports *sitter.Node, f *core.SourceFile) string {
	handlerRequirements := []string{
		`"context"`,
		`"github.com/aws/aws-lambda-go/events"`,
		`"github.com/aws/aws-lambda-go/lambda"`,
		`"github.com/awslabs/aws-lambda-go-api-proxy/chi"`,
		`"github.com/go-chi/chi/v5"`,
	}

	return UpdateImportsInFile(f, handlerRequirements, []string{"github.com/go-chi/chi"})
}

func UpdateGoModWithHandlerRequirements(unit *core.ExecutionUnit) error {
	//TODO: investigate correctly indenting code
	requireCode := `
require (
	github.com/aws/aws-lambda-go v1.19.1 // indirect
	github.com/awslabs/aws-lambda-go-api-proxy v0.13.3 // indirect
	github.com/go-chi/chi/v5 v5.0.7 // indirect
)
	`
	for _, f := range unit.Files() {
		// looking for the root go.mod that we copy to each exec unit
		if f.Path() == "go.mod" {
			modFile, ok := f.(*GoMod)
			if !ok {
				return errors.Errorf("Unable to update %s with new requirements", f.Path())
			}
			// Some requires may be duplicated if the go.mod has similar existing modules but that shouldn't be an issue
			modFile.AddLine(requireCode)
		}
	}

	return nil
}
