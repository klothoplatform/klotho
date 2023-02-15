package golang

import (
	"fmt"
	"strings"

	"github.com/klothoplatform/klotho/pkg/annotation"
	"github.com/klothoplatform/klotho/pkg/config"
	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/logging"
	"github.com/klothoplatform/klotho/pkg/multierr"
	"github.com/klothoplatform/klotho/pkg/query"
	sitter "github.com/smacker/go-tree-sitter"
	"go.uber.org/zap"
)

type PersistSecretsPlugin struct {
	runtime Runtime
	config  *config.Application
}

func (p PersistSecretsPlugin) Name() string { return "Persist" }

func (p PersistSecretsPlugin) Transform(result *core.CompilationResult, deps *core.Dependencies) error {

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

func (p *PersistSecretsPlugin) handleFile(f *core.SourceFile, unit *core.ExecutionUnit) ([]core.CloudResource, error) {
	resources := []core.CloudResource{}
	var errs multierr.Error
	annots := f.Annotations()
	for _, annot := range annots {
		cap := annot.Capability
		if cap.Name != annotation.PersistCapability {
			continue
		}
		secretsResult := querySecret(f, annot)
		if secretsResult != nil {
			persistResource, err := p.transformSecret(f, annot, secretsResult, unit)
			if err != nil {
				errs.Append(err)
			}
			resources = append(resources, persistResource)

		}
	}
	return resources, errs.ErrOrNil()
}

func (p *PersistSecretsPlugin) transformSecret(f *core.SourceFile, cap *core.Annotation, result *persistSecretResult, unit *core.ExecutionUnit) (core.CloudResource, error) {

	args, found := getArguements(result.expression)
	if !found {
		return nil, nil
	}
	// Generate the new node content before replacing the node.
	// We are going to set a new variable to the original file path and split to get the query params
	newNodeContent := `klothoRuntimePathSub := ` + args[1].Content
	//Split the path to get anything after ? so we can get the query params
	newNodeContent += "\nklothoRuntimePathSubChunks := strings.SplitN(klothoRuntimePathSub, \"?\", 2)\n"
	newNodeContent += `var queryParams string
	if len(klothoRuntimePathSubChunks) == 2 {
		queryParams = "&" + klothoRuntimePathSubChunks[1]
	}
	`

	args[1].Content = fmt.Sprintf(`"awssecretsmanager://%s?region=" + os.Getenv("AWS_REGION") + queryParams`, p.config.AppName+"_"+cap.Capability.ID)

	newArgContent := argumentListToString(args)

	newExpressionContent := strings.ReplaceAll(result.expression.Content(), result.args.Content(), newArgContent)
	newNodeContent += newExpressionContent

	err := f.ReplaceNodeContent(result.expression, newNodeContent)
	if err != nil {
		return nil, err
	}

	err = UpdateImportsInFile(f, p.runtime.GetSecretsImports(), []Import{})
	if err != nil {
		return nil, err
	}

	persist := &core.Persist{
		Kind: core.PersistSecretKind,
		Name: cap.Capability.ID,
	}
	return persist, nil
}

type persistSecretResult struct {
	varName    string
	args       *sitter.Node
	expression *sitter.Node
}

func querySecret(file *core.SourceFile, annotation *core.Annotation) *persistSecretResult {
	log := zap.L().With(logging.FileField(file), logging.AnnotationField(annotation))

	runtimeVarImport := GetNamedImportInFile(file, "gocloud.dev/runtimevar")

	nextMatch := doQuery(annotation.Node, openVariable)

	match, found := nextMatch()
	if !found {
		return nil
	}
	varName, args, id := match["varName"], match["args"], match["id"]

	if id != nil {
		if runtimeVarImport.Alias != "" {
			if !query.NodeContentEquals(id, runtimeVarImport.Alias) {
				return nil
			}
		} else {
			if !query.NodeContentEquals(id, "runtimevar") {
				return nil
			}
		}
	}

	if _, found := nextMatch(); found {
		log.Warn("too many assignments for fs_secrets")
		return nil
	}

	return &persistSecretResult{
		varName:    varName.Content(),
		args:       args,
		expression: match["expression"],
	}
}
