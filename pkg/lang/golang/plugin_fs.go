package golang

import (
	"errors"
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

	args := GetArguements(result.args)

	// We need to check to make sure the path supplied to the original node content is a static string. This is because it will get erased and we dont want to leave os level orphaned code
	if !args[0].IsString() {
		return nil, errors.New("must supply static string for secret path")
	}

	args[0].Content = "nil"
	args[1].Content = fmt.Sprintf(`os.Getenv("%s")`, fsEnvVar.Name)

	newNodeContent := strings.Replace(result.args.Content(), result.args.Content(), ArgumentListToString(args), 1)

	err := f.ReplaceNodeContent(result.args, newNodeContent)
	if err != nil {
		return nil, err
	}
	err = f.ReplaceNodeContent(result.operator, "blob")
	if err != nil {
		return nil, err
	}

	err = UpdateImportsInFile(f, p.runtime.GetFsImports(), []Import{{Package: "gocloud.dev/blob/fileblob"}})
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
	varName  string
	operator *sitter.Node
	args     *sitter.Node
}

func queryFS(file *core.SourceFile, annotation *core.Annotation) *persistResult {
	log := zap.L().With(logging.FileField(file), logging.AnnotationField(annotation))

	fileBlobImport := GetNamedImportInFile(file, "gocloud.dev/blob/fileblob")

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
			if !query.NodeContentEquals(id, "fileblob") {
				return nil
			}
		}
	}

	if _, found := nextMatch(); found {
		log.Warn("too many assignments for fs_storage")
		return nil
	}

	return &persistResult{
		varName:  varName.Content(),
		operator: id,
		args:     args,
	}
}
