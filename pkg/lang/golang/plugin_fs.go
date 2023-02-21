package golang

import (
	"fmt"
	"strings"

	"github.com/klothoplatform/klotho/pkg/annotation"
	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/logging"
	"github.com/klothoplatform/klotho/pkg/multierr"
	"github.com/klothoplatform/klotho/pkg/query"
	sitter "github.com/smacker/go-tree-sitter"
	"go.uber.org/zap"
)

type PersistFsPlugin struct {
	runtime Runtime
}

func (p PersistFsPlugin) Name() string { return "Persist" }

func (p PersistFsPlugin) Transform(result *core.CompilationResult, deps *core.Dependencies) error {

	var errs multierr.Error
	for _, res := range result.Resources() {
		unit, ok := res.(*core.ExecutionUnit)
		if !ok {
			continue
		}
		for _, goSource := range unit.FilesOfLang(goLang) {
			resources, err := p.handleFile(goSource, unit)
			if err != nil {
				errs.Append(core.WrapErrf(err, "failed to handle persist in unit %s", unit.Name))
				continue
			}

			for _, r := range resources {
				result.Add(r)

				deps.Add(core.ResourceKey{
					Name: unit.Name,
					Kind: core.ExecutionUnitKind,
				}, r.Key())
			}
		}
	}

	return errs.ErrOrNil()
}

func (p *PersistFsPlugin) handleFile(f *core.SourceFile, unit *core.ExecutionUnit) ([]core.CloudResource, error) {
	resources := []core.CloudResource{}
	var errs multierr.Error
	annots := f.Annotations()
	for _, annot := range annots {
		cap := annot.Capability
		if cap.Name != annotation.PersistCapability {
			continue
		}
		fsResult := queryFS(f, annot)
		if fsResult != nil {
			persistResource, err := p.transformFS(f, annot, fsResult, unit)
			if err != nil {
				errs.Append(err)
			}
			resources = append(resources, persistResource)

		}
	}
	return resources, errs.ErrOrNil()
}

func (p *PersistFsPlugin) transformFS(f *core.SourceFile, cap *core.Annotation, result *persistResult, unit *core.ExecutionUnit) (core.CloudResource, error) {

	fsEnvVar := core.EnvironmentVariable{
		Name:       cap.Capability.ID + "_fs_bucket",
		Kind:       string(core.PersistFileKind),
		ResourceID: cap.Capability.ID,
		Value:      "bucket_url",
	}

	unit.EnvironmentVariables = append(unit.EnvironmentVariables, fsEnvVar)

	args, _ := getArguements(result.expression)
	// Generate the new node content before replacing the node. We just set it so we can compile correctly
	newNodeContent := `var _ = ` + args[1].Content + "\n"

	args[1].Content = fmt.Sprintf(`"s3://" + os.Getenv("%s") + "?region=" + os.Getenv("AWS_REGION")`, fsEnvVar.Name)

	newArgContent := argumentListToString(args)

	newExpressionContent := strings.ReplaceAll(result.expression.Content(), result.args.Content(), newArgContent)
	newNodeContent += newExpressionContent

	err := f.ReplaceNodeContent(result.expression, newNodeContent)
	if err != nil {
		return nil, err
	}

	err = UpdateImportsInFile(f, p.runtime.GetFsImports(), []Import{})
	if err != nil {
		return nil, err
	}

	persist := &core.Persist{
		Kind: core.PersistFileKind,
		Name: cap.Capability.ID,
	}
	return persist, nil
}

type persistResult struct {
	varName    string
	expression *sitter.Node
	args       *sitter.Node
}

func queryFS(file *core.SourceFile, annotation *core.Annotation) *persistResult {
	log := zap.L().With(logging.FileField(file), logging.AnnotationField(annotation))

	fileBlobImport := GetNamedImportInFile(file, "gocloud.dev/blob")

	nextMatch := doQuery(annotation.Node, fileBucket)

	match, found := nextMatch()
	if !found {
		return nil
	}

	varName, args, id := match["varName"], match["args"], match["id"]

	if id != nil {
		if fileBlobImport.Alias != "" {
			if !query.NodeContentEquals(id, fileBlobImport.Alias) {
				return nil
			}
		} else {
			if !query.NodeContentEquals(id, "blob") {
				return nil
			}
		}
	}

	if _, found := nextMatch(); found {
		log.Warn("too many assignments for fs_storage")
		return nil
	}

	return &persistResult{
		varName:    varName.Content(),
		expression: match["expression"],
		args:       args,
	}
}
