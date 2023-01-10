package javascript

import (
	"fmt"
	"strings"

	"github.com/klothoplatform/klotho/pkg/filter"
	"github.com/klothoplatform/klotho/pkg/filter/predicate"

	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/logging"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

type (
	varFinder struct {
		// queryMatchType is the (javascript) type to look for.
		//
		// e.g. "EventEmitter"
		queryMatchType string
		// queryMatchTypeModule (javascript) package that the `queryMatchType` is defined in.
		//
		// e.g. "events", if `queryMatchType` is "EventEmitter" and we expect "events.EventEmitter".
		//
		// If this is empty, we would not expect a qualified type.
		queryMatchTypeModule string
		// annotationFilter filters which variable declarations this will look for
		annotationFilter AnnotationFilter
		// requireExport controls the varFinder's behavior when the target javascript does not export a klotho-annotated field.
		//
		// If this is set to true, the varfinder will ignore (with a warning to the user) any field that isn't exported.
		requireExport bool
	}

	AnnotationFilter func(declaringFile *core.SourceFile, annot *core.Annotation) bool

	VarDeclarations map[VarSpec]*VarParseStructs

	// VarSpec defines a variable defined in a file
	//
	// VarSpecs often come paired with `VarParseStructs`, which contain the tree-sitter file and nodes, along with some metadata.
	VarSpec struct {
		// DefinedIn is the path of the file that this variable is in
		DefinedIn string
		// InternalName is the variable name as appears on the left-hand side of the assignment (if assigned directly to exports) or declaration (if not).
		InternalName string
		// VarName is the name that is exported for other modules to use
		VarName string
	}

	// VarParseStructs hold data about a variable in a javascript file.
	//
	// VarSpecStructs often come paired with a `VarSpec`, which tells you the variable name within the file.
	VarParseStructs struct {
		File       *core.SourceFile
		Annotation *core.Annotation
	}
)

// SplitByFile splits this VarDeclarations into several, each corresponding to the declarations for a single file.
//
// The keys of the map are the file paths.
func (v VarDeclarations) SplitByFile() map[string]VarDeclarations {
	byPath := make(map[string]VarDeclarations)
	for spec, varStructs := range v {
		path := spec.DefinedIn
		if byPath[path] == nil {
			byPath[path] = make(VarDeclarations)
		}
		byPath[path][spec] = varStructs
	}
	return byPath
}

func FilterByCapability(capability string) AnnotationFilter {
	return func(_ *core.SourceFile, annot *core.Annotation) bool {
		return annot.Capability.Name == capability
	}
}

// DiscoverDeclarations finds the var declarations, using the `varFinder`'s `searchSpec`.
func DiscoverDeclarations(files map[string]core.File, queryMatchType string, queryMatchTypeModule string, requireExport bool, annotationFilter AnnotationFilter) VarDeclarations {
	vf := varFinder{
		queryMatchType:       queryMatchType,
		queryMatchTypeModule: queryMatchTypeModule,
		annotationFilter:     annotationFilter,
		requireExport:        requireExport,
	}
	vars := make(VarDeclarations)
	for _, f := range files {
		js, ok := Language.ID.CastFile(f)
		if !ok {
			continue
		}

		for spec, varRef := range vf.discoverFileDeclarations(js) {
			vars[spec] = varRef
		}
	}
	return vars
}

func (vf *varFinder) discoverFileDeclarations(f *core.SourceFile) VarDeclarations {
	vars := make(VarDeclarations)

	for _, annot := range f.Annotations() {
		if !vf.annotationFilter(f, annot) {
			continue
		}

		log := zap.L().With(
			logging.AnnotationField(annot),
			logging.FileField(f),
		)

		internalName, exportName, validErr := vf.parseNode(f, annot)
		if validErr != nil {
			log.Sugar().Warnf("Invalid annotation node: %s", validErr)
			// even though it is invalid, continue, the warn is enough to fail in strict mode
			continue
		}

		spec := VarSpec{
			DefinedIn:    f.Path(),
			InternalName: internalName,
			VarName:      exportName,
		}
		vars[spec] = &VarParseStructs{
			File:       f,
			Annotation: annot,
		}
	}
	return vars
}

func (vf *varFinder) parseNode(f *core.SourceFile, annot *core.Annotation) (internalName string, exportName string, err error) {
	next := DoQuery(annot.Node, declareAndInstantiate)

	for {
		match, found := next()
		if !found {
			break
		}
		if match["type"].Content(f.Program()) != vf.queryMatchType {
			continue
		}
		internalName = match["name"].Content(f.Program())

		if module := match["ctor.obj"]; module != nil {
			moduleStr := module.Content(f.Program())
			if moduleStr != vf.queryMatchTypeModule {
				return "", "", errors.Errorf("ignoring export because its qualified module name is not \"%s\"", vf.queryMatchTypeModule)
			}
		}
		if obj := match["var.obj"]; obj != nil {
			objStr := obj.Content(f.Program())
			if objStr != "exports" {
				return "", "", errors.Errorf(`unsupported: non-"exports" object property assignment (%s.%s)`, objStr, internalName)
			}
			exportName = internalName
			internalName = match["var"].Content(f.Program()) // use fully-qualified object & property
			return
		}

		nextExport := DoQuery(f.Tree().RootNode(), exportedVar)
		for {
			exportMatch, exportFound := nextExport()
			if !exportFound {
				break
			}

			if exportMatch["obj"].Content(f.Program()) != "exports" {
				continue
			}

			if exportMatch["right"].Content(f.Program()) != internalName {
				continue
			}

			exportName = exportMatch["prop"].Content(f.Program())
			if exportName != internalName {
				zap.S().Debugf(`found export of '%s' as '%s': "%s"`, internalName, exportName, exportMatch["assign"].Content(f.Program()))
			}
			return
		}

		var err error
		if vf.requireExport {
			err = errors.Errorf(`no export found for variable "%s"`, internalName)
		}
		return internalName, "", err
	}

	var errString strings.Builder
	fmt.Fprintf(&errString, `expected to find an assignment/definition of "new %s()"`, vf.queryMatchType)
	if vf.queryMatchTypeModule != "" {
		fmt.Fprintf(&errString, ` or "%s.%s()"`, vf.queryMatchTypeModule, vf.queryMatchType)
	}
	errString.WriteString(", but could not find one")

	return "", "", core.NewCompilerError(f, annot, errors.New(errString.String()))
}

// TODO: rework this functionality to re-parsing imports on each invocation once we've added imports at the source file level
// findVarName finds the internal name for the given `varSpec` within the given file. Returns "" if the var isn't defined in the file.
func findVarName(f *core.SourceFile, spec VarSpec) string {
	log := zap.L().With(logging.FileField(f)).Sugar()

	varName := spec.InternalName
	if f.Path() != spec.DefinedIn {
		filteredImports := FindImportsInFile(f).Filter(
			filter.NewSimpleFilter(
				IsRelativeImportOfModule(spec.DefinedIn),
				predicate.AnyOf(
					IsImportOfType(ImportTypeNamespace),
					predicate.AllOf(
						IsImportOfType(ImportTypeNamed),
						ImportHasName(spec.VarName),
					))))

		if len(filteredImports) == 0 {
			return ""
		}
		if len(filteredImports) > 1 {
			log.Warnf("ambiguous imported var detected '%s': %d matches found for the supplied specification", spec, len(filteredImports))
			return ""
		}

		resolvedImport := filteredImports[0]
		if resolvedImport.Type == ImportTypeNamed {
			varName = resolvedImport.ImportedAs()
		} else { // namespace import
			varName = filteredImports[0].ImportedAs() + "." + spec.VarName
		}
	}
	return varName
}
