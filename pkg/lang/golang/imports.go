package golang

import (
	"fmt"
	"strings"

	"github.com/klothoplatform/klotho/pkg/core"
	sitter "github.com/smacker/go-tree-sitter"
)

type Import struct {
	Alias   string
	Package string
}

func GetImportsInFile(f *core.SourceFile) []Import {
	nextMatch := doQuery(f.Tree().RootNode(), findImports)
	imports := []Import{}
	for {
		match, found := nextMatch()
		if !found {
			break
		}

		var alias string
		var packageId string
		if match["package_id"] != nil {
			alias = match["package_id"].Content()
		}
		if match["package_path"] != nil {
			packageId = strings.ReplaceAll(match["package_path"].Content(), `"`, ``)
		}

		imports = append(imports, Import{
			Alias:   alias,
			Package: packageId,
		})

	}
	return imports
}

func GetImportNode(f *core.SourceFile) *sitter.Node {
	nextMatch := doQuery(f.Tree().RootNode(), findImports)
	var imports *sitter.Node
	for {
		match, found := nextMatch()
		if !found {
			break
		}

		imports := match["expression"]

		if imports != nil {
			return imports
		}
	}
	return imports
}

func UpdateImportsInFile(f *core.SourceFile, importsToAdd []string, importsToRemove []string) string {

	imports := GetImportsInFile(f)

	// Determine which imports already exist and which we need to add
	for _, i := range importsToAdd {
		willAdd := true
		for _, singleImport := range imports {
			if singleImport.Package == i {
				willAdd = false
			}
		}
		if willAdd {
			imports = append(imports, Import{Package: i})
		}
	}
	newImports := []Import{}
	for _, singleImport := range imports {
		willAdd := true
		for _, i := range importsToRemove {
			if singleImport.Package == i {
				willAdd = false
			}
		}
		if willAdd {
			newImports = append(newImports, Import{Package: singleImport.Package, Alias: singleImport.Alias})
		}
	}

	// Create the new import block
	newImportCode := "\nimport ("
	for _, i := range newImports {
		if i.Alias != "" {
			newImportCode = fmt.Sprintf("%s\n\t%s \"%s\"", newImportCode, i.Alias, i.Package)
		} else {
			newImportCode = fmt.Sprintf("%s\n\t\"%s\"", newImportCode, i.Package)

		}
	}
	newImportCode = newImportCode + "\n)"

	// Specifically handle removing the old chi import to ensure we only use chi/v5
	oldNodeContent := GetImportNode(f).Content()

	newFileContent := string(f.Program())

	newFileContent = strings.ReplaceAll(
		newFileContent,
		oldNodeContent,
		newImportCode,
	)
	return newFileContent
}
