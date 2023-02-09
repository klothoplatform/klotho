package golang

import (
	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/query"
	sitter "github.com/smacker/go-tree-sitter"
)

type Package struct {
	Node *sitter.Node
	Name string
}

func FindFilesForPackageName(unit *core.ExecutionUnit, pkgName string) []*core.SourceFile {
	var packageFiles []*core.SourceFile
	for _, f := range unit.Files() {
		src, ok := goLang.CastFile(f)
		if !ok {
			continue
		}

		nextMatch := doQuery(src.Tree().RootNode(), packageQuery)
		for {
			match, found := nextMatch()
			if !found {
				break
			}
			package_name := match["package_name"]
			if query.NodeContentEquals(package_name, pkgName) {
				packageFiles = append(packageFiles, src)
			}
		}
	}

	return packageFiles
}

func FindPackageNode(f *core.SourceFile) Package {
	nextMatch := doQuery(f.Tree().RootNode(), packageQuery)
	for {
		match, found := nextMatch()
		if !found {
			break
		}
		return Package{
			Node: match["clause"],
			Name: match["package_name"].Content(),
		}
	}
	return Package{}
}
