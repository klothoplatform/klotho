package golang

import (
	"fmt"
	"strings"

	"github.com/klothoplatform/klotho/pkg/compiler/types"
	"github.com/pkg/errors"
	sitter "github.com/smacker/go-tree-sitter"
)

type Import struct {
	Alias    string
	Package  string
	SpecNode *sitter.Node
}

func (i *Import) ToString() string {
	return i.SpecNode.Content()
}

func GetImportsInFile(f *types.SourceFile) []Import {
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
			Alias:    alias,
			Package:  packageId,
			SpecNode: match["spec"],
		})

	}
	return imports
}

func GetNamedImportInFile(f *types.SourceFile, namedImport string) Import {
	imports := GetImportsInFile(f)
	for _, i := range imports {
		if i.Package == namedImport {
			return i
		}
	}
	return Import{}
}

func UpdateImportsInFile(f *types.SourceFile, importsToAdd []Import, importsToRemove []Import) error {

	imports := GetImportsInFile(f)
	newImports := []Import{}
	// Determine which imports already exist and which we need to add
	for _, i := range importsToAdd {
		willAdd := true
		for _, singleImport := range imports {
			if singleImport.Package == i.Package {
				willAdd = false
			}
		}
		if willAdd {
			newImports = append(newImports, i)
		}
	}

	for _, singleImport := range imports {
		willAdd := true
		for _, i := range importsToRemove {
			if singleImport.Package == i.Package {
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

	// Delete all old import nodes
	err := DeleteImportNodes(f)
	if err != nil {
		return errors.Wrap(err, "could not delete imports")
	}

	packageNode := FindPackageNode(f)
	insertionPoint := packageNode.Node.EndByte()
	content := f.Program()
	contentStr := string(content[0:insertionPoint]) + "\n" + newImportCode
	if len(f.Program()) > int(insertionPoint) {
		contentStr += string(content[insertionPoint:])
	}

	err = f.Reparse([]byte(contentStr))
	if err != nil {
		return errors.Wrap(err, "could not reparse inserted import")
	}

	return nil
}

func DeleteImportNodes(f *types.SourceFile) error {
	for {
		nextMatch := doQuery(f.Tree().RootNode(), findImports)
		match, found := nextMatch()
		if !found {
			break
		}
		err := f.ReplaceNodeContent(match["expression"], "")
		if err != nil {
			return err
		}
	}
	return nil
}
