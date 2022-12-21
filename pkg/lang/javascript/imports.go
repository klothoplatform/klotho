package javascript

import (
	"sort"
	"strings"

	"github.com/klothoplatform/klotho/pkg/filter"
	"github.com/klothoplatform/klotho/pkg/filter/predicate"
	"github.com/klothoplatform/klotho/pkg/query"

	"github.com/klothoplatform/klotho/pkg/core"
	sitter "github.com/smacker/go-tree-sitter"
)

type ImportType string

var (

	// ImportTypeNamed represents an import of a specific named export from a module (other than "default")
	ImportTypeNamed = ImportType("named")

	// ImportTypeDefault represents an import of the default export from a module
	ImportTypeDefault = ImportType("default")

	// ImportTypeNamespace represents an import of all exported names from a module
	ImportTypeNamespace = ImportType("namespace")

	// ImportTypeSideEffect represents an import that is not assigned to a variable
	ImportTypeSideEffect = ImportType("side-effect")

	// ImportTypeField represents an imported field of a named export
	// (e.g. const { prop } = require("module").name or const alias = require("module").name.field ...)
	ImportTypeField = ImportType("field")
)

type ImportScope string

var (
	ImportScopeModule = ImportScope("module")
	ImportScopeLocal  = ImportScope("local")
)

type ImportSourceType string

var (
	ImportSourceTypeLocalModule = ImportSourceType("local-module")
	ImportSourceTypeNodeModule  = ImportSourceType("node-module")
)

type ImportKind string

var (
	ImportKindCommonJS = ImportKind("commonjs")
	ImportKindES       = ImportKind("es")
)

type (
	Import struct {
		// Source could be the name of a node module, a project-local module, or a filename
		Source string

		// Name is the exported name of the Import
		Name string

		// ImportNode is the *sitter.Node associated with the Import's import statement
		ImportNode *sitter.Node

		// SourceNode is the *sitter.Node associated with the import's Source.
		// For CJS imports, this will include the 'require()' expression.
		SourceNode *sitter.Node

		// Alias is the name with which this import is referred to in its enclosing Scope (i.e. module or local)
		Alias string

		Scope ImportScope
		Type  ImportType
		Kind  ImportKind
	}
)

// FileImports provides a mapping between import sources and the list of imports for each.
type FileImports map[string][]Import

// ImportedAs returns the name of the import as it will be used locally (either the exported name or local alias).
func (p *Import) ImportedAs() string {
	if p.Alias != "" {
		return p.Alias
	}
	return p.Name
}

// Filter applies the supplied Filter to all Import values and returns the filtered list of Import values.
func (imports FileImports) Filter(filter filter.Filter[Import]) []Import {
	filteredImports := filter.Apply(imports.AsSlice()...)
	return filteredImports
}

// AsSlice converts an instance of FileImports to []Import for simpler iteration over all Import values.
func (imports FileImports) AsSlice() []Import {
	var slice []Import
	for _, importsOfSource := range imports {
		slice = append(slice, importsOfSource...)
	}
	return slice
}

// FindImportForVar returns the import assigned to the supplied variable name starting from the *sitter.Node 'n' in the source
// and a boolean reflecting whether an appropriate import was found or not.
func FindImportForVar(n *sitter.Node, source []byte, varName string) Import {
	imports := FindImportsAtNode(n, source)
	if filteredImports := imports.Filter(filter.NewSimpleFilter(ImportedAs(varName))); len(filteredImports) == 1 {
		return filteredImports[0]
	}
	return Import{}
}

// FindImportsInFile returns a map containing a list of imports for each import source referenced within the file.
func FindImportsInFile(file *core.SourceFile) FileImports {
	return FindImportsAtNode(file.Tree().RootNode(), file.Program())
}

// FindImportsAtNode returns a map containing a list of imports for each import source starting from the supplied node.
func FindImportsAtNode(node *sitter.Node, program []byte) FileImports {
	fileImports := FileImports{}
	matches := queryImports(node, program)
	for _, match := range matches {
		if parsedImport, ok := parseImport(match, program); ok {
			i := fileImports[parsedImport.Source]
			fileImports[parsedImport.Source] = append(i, parsedImport)
		}
	}
	return fileImports
}

func queryImports(node *sitter.Node, program []byte) []query.MatchNodes {
	nextMatch := DoQuery(node, modulesImport)
	handledNodes := map[*sitter.Node]struct{}{}

	var matches []query.MatchNodes
	for {
		if match, found := nextMatch(); found {
			matches = append(matches, match)
		} else {
			break
		}
	}

	// Prioritize processing order of matches using the following rules (primarily for CJS support):
	// 1. Declaration CJS imports before side effect imports or ES imports
	// 2. Descending 'member_expression' 'endByte' value when the import is part of a member expression
	//	  - This ensures we process 'require("module").name.field1.field2' over the likes of 'require("module").name.field1', etc.
	// 3. Deduplication ('dedup') imports before side effect imports
	// 4. Side effect imports and ES imports come last in no particular order
	//
	// Note: the sort order is unstable (especially since we don't always bother comparing i to j)
	sort.Slice(matches, func(i, j int) bool {
		fnWrapperI, iOk := matches[i]["func.wrapper"]
		fnWrapperJ, jOk := matches[j]["func.wrapper"]

		if matches[i]["cjs.requireStatement"] != nil {
			return true
		} else if iOk && jOk {
			return fnWrapperI.EndByte() > fnWrapperJ.EndByte()
		} else if iOk {
			return true
		}
		return matches[i]["dedup"] != nil
	})

	var filteredMatches []query.MatchNodes
	for _, match := range matches {

		dedup := match["dedup"]
		if dedup != nil && dedup.Parent().Type() != "variable_declarator" {
			continue // ignore deduplication matches if we're dealing with a side effect or assignment import
		}
		possibleDeclarator := ascendWhile(match["sideEffect.func.wrapper"], predicate.Not(nodeTypeIs("variable_declarator")))
		if possibleDeclarator != nil {
			continue // ignore sideEffect.func.wrapper if it's actually a declared import
		}

		fnWrapper := match["func.wrapper"]
		fn := match["func"]
		_, handledFn := handledNodes[fn]
		_, handledFnWrapper := handledNodes[fnWrapper]
		if handledFnWrapper || (handledFn && match["cjs.sideEffect.requireStatement"] != nil) {
			continue
		}
		if fn != nil {
			handledNodes[fn] = struct{}{}
		}
		if fnWrapper != nil {
			nmf := DoQuery(fnWrapper, "(_) @node")
			for fnWrapperMatches, found := nmf(); found; fnWrapperMatches, found = nmf() {
				handledNodes[fnWrapperMatches["node"]] = struct{}{}
			}

		}
		if dedup == nil {
			filteredMatches = append(filteredMatches, match)
		}
	}
	return filteredMatches

}

// FindNextImportStatement returns the imports associated with the next import statement/expression
// starting at the supplied node (typically starting from an annotation comment node).
func FindNextImportStatement(node *sitter.Node, program []byte) []Import {
	var imports []Import
	matches := queryImports(node, program)
	var previousNode *sitter.Node
	for _, match := range matches {
		if len(match) == 0 {
			continue
		}

		if imp, ok := parseImport(match, program); ok && (previousNode == nil || imp.ImportNode == previousNode) {
			imports = append(imports, imp)
			previousNode = imp.ImportNode
		} else {
			break
		}
	}
	return imports
}

func parseImport(match query.MatchNodes, program []byte) (Import, bool) {
	esImportStatement := match["es.importStatement"]
	cjsRequireStatement := match["cjs.requireStatement"]
	cjsSideEffectRequireStatement := match["cjs.sideEffect.requireStatement"]
	fn := match["func"]
	fnWrapper := match["func.wrapper"]

	looksLikeCJSImport := cjsSideEffectRequireStatement != nil || cjsRequireStatement != nil
	invokesRequire := fn != nil && fn.Content(program) == "require"
	isNestedMemberExprContainingRequire := fnWrapper != nil && strings.Contains(fnWrapper.Content(program), "require")

	var parsedImport Import
	if (looksLikeCJSImport && invokesRequire) || isNestedMemberExprContainingRequire {
		parsedImport = parseCjsImport(match, program)
	} else if esImportStatement != nil {
		parsedImport = parseESImport(match, program)
	}

	if parsedImport.Type == "" {
		return Import{}, false
	}

	return parsedImport, true
}

func parseESImport(match query.MatchNodes, content []byte) Import {

	esImportStatement := match["es.importStatement"]
	source := match["source"]
	alias := match["alias"]
	export := match["export"]

	esImport := Import{Kind: ImportKindES, ImportNode: esImportStatement, SourceNode: source}
	esImport.Source = StringLiteralContent(source, content)

	if esImportStatement.Parent().Type() == "program" {
		esImport.Scope = ImportScopeModule
	} else {
		esImport.Scope = ImportScopeLocal
	}

	if alias != nil {
		aliasContent := alias.Content(content)
		esImport.Alias = aliasContent

		aliasParentType := alias.Parent().Type()
		if aliasParentType == "import_clause" {
			esImport.Name = "default"
			esImport.Type = ImportTypeDefault
		}
		if aliasParentType == "namespace_import" {
			esImport.Name = "*"
			esImport.Type = ImportTypeNamespace
		}
	}

	exportContent := ""
	if export != nil {
		if export.Type() == "string" {
			exportContent = StringLiteralContent(export, content)
		} else {
			exportContent = export.Content(content)
		}
		esImport.Name = exportContent

		if exportContent == "default" {
			esImport.Type = ImportTypeDefault
		} else {
			esImport.Type = ImportTypeNamed
		}
	}

	if esImport.Name == "" && esImport.Alias == "" {
		esImport.Type = ImportTypeSideEffect
	}

	return esImport
}

func parseCjsImport(match query.MatchNodes, content []byte) Import {
	cjsDeclarativeRequireStatement := match["cjs.requireStatement"]
	cjsSideEffectRequireStatement := match["cjs.sideEffect.requireStatement"]
	fnWrapper := match["func.wrapper"]

	requireStatement := cjsDeclarativeRequireStatement
	if requireStatement == nil {
		requireStatement = cjsSideEffectRequireStatement
	}
	if requireStatement == nil {
		requireStatement = fnWrapper
	}

	if requireStatement == nil {
		return Import{}
	}

	alias := match["local.name"]
	exportedName := match["func.source.name"]
	tsWrapper := match["ts.wrapper"]
	source := match["source"]
	fnExpr := match["func.expr"]
	destructuredExportedName := match["destructured.source.name"]

	cjsImport := Import{Kind: ImportKindCommonJS}
	if fnExpr != nil {
		cjsImport.SourceNode = fnExpr
		cjsImport.Source = StringLiteralContent(source, content)
	}

	importNode := ascendWhile(requireStatement,
		predicate.AnyOf(
			nodeTypeIs("member_expression"),
			nodeTypeIs("call_expression"),
		))

	if importNode.Type() == "assignment_expression" {
		// handle assignment expressions by checking the parent of a detected side effect import
		// rather than duplicating the entire query again.
		// e.g. x = require('module');
		cjsImport.Alias = StringLiteralContent(importNode.ChildByFieldName("left"), content)
	}
	if importNode.Parent().Type() == "sequence_expression" { // e.g. x = require("y"), z = require("a")
		importNode = ascendWhile(importNode.Parent(), nodeTypeIs("sequence_expression"))
	}
	if importNode.Parent().Type() == "expression_statement" {
		// capture the entire statement if it's a stand-alone side effect or assignment import
		cjsImport.ImportNode = importNode.Parent()
	} else {
		cjsImport.ImportNode = importNode
	}

	if alias != nil {
		aliasContent := alias.Content(content)
		cjsImport.Alias = aliasContent

		if alias.Type() == "property_identifier" {
			if aliasContent == "default" {
				cjsImport.Type = ImportTypeDefault
			} else {
				cjsImport.Type = ImportTypeNamed
			}
		}
	}

	nameNode := exportedName
	if destructuredExportedName != nil {
		nameNode = destructuredExportedName
	}

	// unwrap an import wrapped by one more member expressions:
	// e.g. require("module").name.field1.field2.field3 ... or {x, x:y } = require("module").name
	if fnWrapper != nil || (fnExpr != nil && destructuredExportedName != nil && exportedName != nil) {
		memberNode := fnWrapper
		if memberNode == nil {
			memberNode = fnExpr
		}

		// most import forms can end up with at least one layer of member expressions
		// above the capture point that we're interested in
		if memberNode.Parent().Type() == "member_expression" {
			memberNode = memberNode.Parent()
		}

		// query the details of the wrapped import
		nextMatch := DoQuery(memberNode, functionInvocation)
		nestedMatch, found := nextMatch()

		if !found {
			return Import{}
		}

		fnWrapperName := nestedMatch["function.name"]
		fnArgs := nestedMatch["function.args"]
		fnArg := nestedMatch["function.arg"]

		if !(fnWrapperName.Content(content) == "require" &&
			fnArgs != nil &&
			fnArgs.NamedChildCount() == 1 &&
			fnArg.Type() == "string") {
			return Import{}
		}
		cjsImport.Source = StringLiteralContent(fnArg, content)
		cjsImport.SourceNode = fnArg

		// resolve the path to the field being accessed on the import
		var accessChain []string
		if memberNode == fnExpr && exportedName != nil {
			accessChain = []string{exportedName.Content(content)}
		}
		for node := memberNode; node.Type() == "member_expression"; node = node.ChildByFieldName("object") {
			if node == nil {
				break
			}
			accessChain = append([]string{node.ChildByFieldName("property").Content(content)}, accessChain...)
		}
		if destructuredExportedName != nil {
			accessChain = append(accessChain, destructuredExportedName.Content(content))
		}
		name := strings.Join(accessChain, ".")
		cjsImport.Name = name

		if memberNode.Parent().Type() != "assignment_expression" && memberNode.Parent().Type() != "variable_declarator" {
			cjsImport.Type = ImportTypeSideEffect
		} else if strings.Contains(name, ".") {
			cjsImport.Type = ImportTypeField
		} else if name == "default" {
			cjsImport.Type = ImportTypeDefault
		} else if name != "" {
			cjsImport.Type = ImportTypeNamed
		}
	}

	// make sure to resolve the scope after first setting cjsImport.ImportNode to the outermost member_expression node
	if cjsImport.ImportNode.Parent().Type() == "program" {
		cjsImport.Scope = ImportScopeModule
	} else {
		cjsImport.Scope = ImportScopeLocal
	}

	// handle import types for TypeScript's esModuleInterop config: https://www.typescriptlang.org/tsconfig#esModuleInterop
	if tsWrapper != nil {
		tsWrapperContent := tsWrapper.Content(content)
		if tsWrapperContent == "__importStar" {
			cjsImport.Type = ImportTypeNamespace
			cjsImport.Name = "*"
		} else if tsWrapperContent == "__importDefault" {
			cjsImport.Type = ImportTypeDefault
			cjsImport.Name = "default"
		}
	}

	if cjsImport.Type != "" {
		return cjsImport
	}

	// make a best-effort guess at the correct import type
	if nameNode != nil {
		exportContent := nameNode.Content(content)
		cjsImport.Name = exportContent

		if exportContent == "default" {
			cjsImport.Type = ImportTypeDefault
		} else {
			cjsImport.Type = ImportTypeNamed
		}
	} else if tsWrapper == nil && cjsImport.Alias != "" {
		// default to a star/namespace import
		cjsImport.Name = "*"
		cjsImport.Type = ImportTypeNamespace
	}

	// fall back to a side effect import if we can't otherwise make a determination
	if cjsImport.Source != "" && cjsImport.Type == "" || (cjsImport.Name == "" && cjsImport.Alias == "") {
		cjsImport.Type = ImportTypeSideEffect
	}
	return cjsImport
}

// ResolveImportSourceType resolves ImportSourceType for the supplied source using the supplied project file paths as context.
//
// This function returns ImportSourceTypeLocalModule if the source either starts with a period ('.')
// or a variant exists in the supplied projectFilePaths slice.
// If the source type cannot be resolved locally, it is assumed to be of type ImportSourceTypeNodeModule.
func ResolveImportSourceType(source string, projectFilePaths []string, enableAbsolute bool) ImportSourceType {
	if strings.HasPrefix(source, ".") {
		return ImportSourceTypeLocalModule
	} else if enableAbsolute {
		filePaths := make(map[string]struct{})

		filePaths[source] = struct{}{}             // X
		filePaths[source+".js"] = struct{}{}       // X.js
		filePaths[source+"/index.js"] = struct{}{} // X/index.js

		for _, filePath := range projectFilePaths {
			if _, ok := filePaths[filePath]; ok {
				return ImportSourceTypeLocalModule
			}
		}
	}
	return ImportSourceTypeNodeModule
}

// IsRelativeImportOfModule matches any local imports with sources that match the following forms for the supplied path:
//
// | `require(X)`   | description |
// | ------------   | ----------- |
// | `./X`          | If X is a file, load X as its file extension format. |
// | `./X.js`       | If X.js is a file, load X.js as JavaScript text. |
// | `./X/index.js` | If X/index.js is a file, load X/index.js as JavaScript text. |
// | `./X/`         | (load as `./X/index.js`) |
//
// This predicate does not match absolute imports supported by Webpack or Babel
func IsRelativeImportOfModule(sourcePath string) predicate.Predicate[Import] {
	return func(p Import) bool {
		return IsImportOfModule(FileToLocalModule(sourcePath))(p)
	}
}

// IsImportOfModule matches any imports with sources that match the following forms for the supplied path:
//
// | `require(X)`   | description |
// | ------------   | ----------- |
// | `X`          | If X is a file, load X as its file extension format. |
// | `X.js`       | If X.js is a file, load X.js as JavaScript text. |
// | `X/index.js` | If X/index.js is a file, load X/index.js as JavaScript text. |
// | `X/`         | (load as `./X/index.js`) |
//
// Unlike IsRelativeImportOfModule, no './' prefix is applied to sourcePath.
func IsImportOfModule(sourcePath string) predicate.Predicate[Import] {
	return func(p Import) bool {
		module := FileToModule(sourcePath)
		modules := make(map[string]struct{})

		modules[module] = struct{}{}             // X
		modules[module+".js"] = struct{}{}       // X.js
		modules[module+"/index.js"] = struct{}{} // X/index.js
		modules[module+"/"] = struct{}{}         // X/

		_, ok := modules[p.Source]
		return ok
	}
}

func IsImportOfKind(importKind ImportKind) predicate.Predicate[Import] {
	return func(p Import) bool {
		return p.Kind == importKind
	}
}
func IsImportOfType(importType ImportType) predicate.Predicate[Import] {
	return func(p Import) bool {
		return p.Type == importType
	}
}

func IsImportInScope(scope ImportScope) predicate.Predicate[Import] {
	return func(p Import) bool {
		return p.Scope == scope
	}
}

func IsRelativeImport(p Import) bool {
	return strings.HasPrefix(p.Source, ".")
}

func ImportedAs(localName string) predicate.Predicate[Import] {
	return func(p Import) bool {
		return p.ImportedAs() == localName
	}
}

func ImportHasName(name string) predicate.Predicate[Import] {
	return func(p Import) bool {
		return p.Name == name
	}
}

func ascendWhile(node *sitter.Node, predicate predicate.Predicate[*sitter.Node]) *sitter.Node {
	for ; node != nil && predicate(node); node = node.Parent() {
	}
	return node
}

func nodeTypeIs(nodeType string) predicate.Predicate[*sitter.Node] {
	return func(n *sitter.Node) bool {
		return n.Type() == nodeType
	}
}
