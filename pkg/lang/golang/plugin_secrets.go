package golang

import (
	"fmt"
	"strings"

	"github.com/klothoplatform/klotho/pkg/annotation"
	"github.com/klothoplatform/klotho/pkg/compiler/types"
	"github.com/klothoplatform/klotho/pkg/config"
	"github.com/klothoplatform/klotho/pkg/construct"
	klotho_errors "github.com/klothoplatform/klotho/pkg/errors"
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

func (p PersistSecretsPlugin) Transform(input *types.InputFiles, fileDeps *types.FileDependencies, constructGraph *construct.ConstructGraph) error {

	var errs multierr.Error
	for _, unit := range construct.GetConstructsOfType[*types.ExecutionUnit](constructGraph) {
		for _, goSource := range unit.FilesOfLang(goLang) {
			resources, err := p.handleFile(goSource, unit)
			if err != nil {
				errs.Append(klotho_errors.WrapErrf(err, "failed to handle persist in unit %s", unit.Name))
				continue
			}

			for _, r := range resources {
				constructGraph.AddConstruct(r)
				constructGraph.AddDependency(unit.Id(), r.Id())
			}
		}
	}

	return errs.ErrOrNil()
}

func (p *PersistSecretsPlugin) handleFile(f *types.SourceFile, unit *types.ExecutionUnit) ([]construct.Construct, error) {
	resources := []construct.Construct{}
	var errs multierr.Error
	annots := f.Annotations()
	for _, annot := range annots {
		cap := annot.Capability
		if cap.Name != annotation.ConfigCapability {
			continue
		}
		isSecret, found := cap.Directives.Bool("secret")
		if !isSecret || !found {
			continue
		}
		secretsResult := querySecret(f, annot)
		if secretsResult != nil {
			secretResource, err := p.transformSecret(f, annot, secretsResult, unit)
			if err != nil {
				errs.Append(err)
			}
			p.runtime.SetConfigType(cap.ID, isSecret)
			resources = append(resources, secretResource)

		}
	}
	return resources, errs.ErrOrNil()
}

func (p *PersistSecretsPlugin) transformSecret(f *types.SourceFile, cap *types.Annotation, result *persistSecretResult, unit *types.ExecutionUnit) (construct.Construct, error) {
	secret := &types.Config{
		Name:   cap.Capability.ID,
		Secret: true,
	}

	args, found := getArguments(result.expression)
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
	secretEnvVar := types.GenerateSecretEnvVar(secret)

	unit.EnvironmentVariables.Add(secretEnvVar)

	args[1].Content = fmt.Sprintf(`"awssecretsmanager://" + os.Getenv("%s") + "?region=" + os.Getenv("AWS_REGION") + queryParams`, secretEnvVar.Name)

	newArgContent := argumentListToString(args)

	newExpressionContent := strings.ReplaceAll(result.expression.Content(), result.args.Content(), newArgContent)
	newNodeContent += newExpressionContent

	err := f.ReplaceNodeContent(result.expression, newNodeContent)
	if err != nil {
		return nil, err
	}

	err = UpdateImportsInFile(f, p.runtime.GetSecretsImports(), []Import{{Package: "gocloud.dev/runtimevar/filevar"}, {Package: "gocloud.dev/runtimevar/constantvar"}})
	if err != nil {
		return nil, err
	}

	return secret, nil
}

type persistSecretResult struct {
	varName    string
	args       *sitter.Node
	expression *sitter.Node
}

func querySecret(file *types.SourceFile, annotation *types.Annotation) *persistSecretResult {
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
