package aws_runtime

import (
	_ "embed"
	"fmt"
	"regexp"

	"github.com/klothoplatform/klotho/pkg/config"
	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/infra/kubernetes"
	"github.com/klothoplatform/klotho/pkg/lang/golang"
	"github.com/klothoplatform/klotho/pkg/provider/aws"
	"github.com/klothoplatform/klotho/pkg/runtime"
	"github.com/pkg/errors"
)

type (
	AwsRuntime struct {
		TemplateConfig aws.TemplateConfig
		Cfg            *config.Application
	}

	TemplateData struct {
		aws.TemplateConfig
		ExecUnitName string
		Expose       ExposeTemplateData
		MainModule   string
	}

	ExposeTemplateData struct {
		ExportedAppVar string
		AppModule      string
	}
)

//go:embed Lambda_Dockerfile
var dockerfileLambda []byte

//go:embed Exec_Dockerfile
var dockerfileExec []byte

func (r *AwsRuntime) AddExecRuntimeFiles(unit *core.ExecutionUnit, constructGraph *core.ConstructGraph) error {
	var DockerFile []byte
	unitType := r.Cfg.GetResourceType(unit)
	switch unitType {
	case aws.Lambda:
		DockerFile = dockerfileLambda
	case aws.Ecs, kubernetes.KubernetesType:
		DockerFile = dockerfileExec
	default:
		return errors.Errorf("unsupported execution unit type: '%s'", unitType)
	}

	templateData := TemplateData{
		TemplateConfig: r.TemplateConfig,
		ExecUnitName:   unit.Name,
	}

	if runtime.ShouldOverrideDockerfile(unit) {
		err := golang.AddRuntimeFile(unit, templateData, "Dockerfile", DockerFile)
		if err != nil {
			return err
		}
	}

	return nil
}

var commentRegex = regexp.MustCompile(`(?m)^(\s*)`)

func (r *AwsRuntime) ActOnExposeListener(unit *core.ExecutionUnit, f *core.SourceFile, listener *golang.HttpListener, routerName string) error {
	unitType := r.Cfg.GetResourceType(unit)
	//TODO: Move comment listen code to library logic like JS does eventually
	if unitType == aws.Lambda {
		nodeToComment := listener.Expression.Content()
		//TODO: Will likely need to move this into a separate plugin of some sort
		// Instead of having a dispatcher file, the dipatcher logic is injected into the main.go file. By having that
		// logic in the expose plugin though, it will only happen if they use the expose annotation for the lambda case.
		if len(nodeToComment) > 0 {
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
			//End - Added by Klotho`, routerName)

			newNodeContent = newNodeContent + dispatcherCode

			err := f.ReplaceNodeContent(listener.Expression, newNodeContent)
			if err != nil {
				return errors.Wrap(err, "error reparsing after substitutions")
			}
		}

		handlerRequirements := []golang.Import{
			{Package: "context"},
			{Package: "github.com/aws/aws-lambda-go/events"},
			{Package: "github.com/aws/aws-lambda-go/lambda"},
			{Package: "github.com/awslabs/aws-lambda-go-api-proxy/chi"},
			{Package: "github.com/go-chi/chi/v5"},
		}

		err := golang.UpdateImportsInFile(f, handlerRequirements, []golang.Import{{Package: "github.com/go-chi/chi"}})
		if err != nil {
			return errors.Wrap(err, "error updating imports")
		}

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
				modFile, ok := f.(*golang.GoMod)
				if !ok {
					return errors.Errorf("Unable to update %s with new requirements", f.Path())
				}
				// Some requires may be duplicated if the go.mod has similar existing modules but that shouldn't be an issue
				modFile.AddLine(requireCode)
			}
		}

		return nil
	}
	return nil
}

func (r *AwsRuntime) GetFsImports() []golang.Import {
	return []golang.Import{
		{Package: "os"},
		{Package: "gocloud.dev/blob"},
		{Package: "gocloud.dev/blob/s3blob", Alias: "_"},
	}
}

func (r *AwsRuntime) GetSecretsImports() []golang.Import {
	return []golang.Import{
		{Package: "os"},
		{Package: "strings"},
		{Package: "gocloud.dev/runtimevar"},
		{Package: "gocloud.dev/runtimevar/awssecretsmanager", Alias: "_"},
	}
}

func (r *AwsRuntime) SetConfigType(id string, isSecret bool) {
	cfg := r.Cfg.Config[id]
	if cfg == nil {
		if isSecret {
			r.Cfg.Config[id] = &config.Config{Type: aws.Secrets_manager}
		}
	} else if isSecret && cfg.Type != aws.Secrets_manager {
		cfg.Type = aws.Secrets_manager
	}
}
