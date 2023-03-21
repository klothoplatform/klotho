package aws_runtime

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/klothoplatform/klotho/pkg/sanitization"

	"github.com/klothoplatform/klotho/pkg/config"
	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/lang/javascript"
	"github.com/klothoplatform/klotho/pkg/provider/aws"
	"github.com/klothoplatform/klotho/pkg/provider/aws/resources"
	"github.com/klothoplatform/klotho/pkg/runtime"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

//go:generate ./compile_template.sh proxy_fargate proxy_eks dispatcher_lambda dispatcher_fargate secret keyvalue orm emitter redis_node redis_cluster fs

type (
	AwsRuntime struct {
		Config         *config.Application
		TemplateConfig aws.TemplateConfig
	}

	TemplateData struct {
		aws.TemplateConfig
		ExecUnitName       string
		Expose             ExposeTemplateData
		MainModule         string
		ProjectFilePath    string
		PayloadsBucketName string
	}

	ExposeTemplateData struct {
		ExportedAppVar string
		AppModule      string
	}
)

//go:embed keyvalue.js.tmpl
var kvRuntimeFiles embed.FS

//go:embed fs.js.tmpl
var fsRuntimeFiles embed.FS

//go:embed secret.js.tmpl
var secretRuntimeFiles embed.FS

//go:embed orm.js.tmpl
var ormRuntimeFiles embed.FS

//go:embed redis_node.js.tmpl
var redisNodeRuntimeFiles embed.FS

//go:embed redis_cluster.js.tmpl
var redisClusterRuntimeFiles embed.FS

//go:embed emitter.js.tmpl
var pubsubRuntimeFiles embed.FS

// the fs template is added here since the dispatcher needs s3. This means it technically doesn't
// need to be added later via persist or proxy as it already exists.
//
//go:embed clients.js package.json
var ExecRuntimeFiles embed.FS

//go:embed proxy_lambda.js.tmpl
var proxyLambda []byte

//go:embed proxy_fargate.js.tmpl
var proxyFargate []byte

//go:embed proxy_eks.js.tmpl
var proxyEks []byte

//go:embed proxy_apprunner.js.tmpl
var proxyApprunner []byte

//go:embed dispatcher_lambda.js.tmpl
var dispatcherLambda []byte

//go:embed dispatcher_fargate.js.tmpl
var dispatcherFargate []byte

//go:embed Lambda_Dockerfile.tmpl
var dockerfileLambda []byte

//go:embed Fargate_Dockerfile.tmpl
var dockerfileFargate []byte

var sequelizeReplaceRE = regexp.MustCompile(`new (\w+\.|\b)Sequelize\(`)

func (r *AwsRuntime) TransformPersist(file *core.SourceFile, annot *core.Annotation, kind core.PersistKind) error {
	importModule := ""
	switch kind {
	case core.PersistFileKind:
		importModule = sanitization.IdentifierSanitizer.Apply("fs_" + annot.Capability.ID)
	case core.PersistKVKind:
		importModule = "keyvalue"
	case core.PersistSecretKind:
		importModule = "secret"
	case core.PersistORMKind:
		importModule = "orm"
	case core.PersistRedisClusterKind:
		importModule = "redis_cluster"
	case core.PersistRedisNodeKind:
		importModule = "redis_node"
	default:
		return fmt.Errorf("could not get runtime import file name for persist type: %v", kind)
	}

	err := javascript.EnsureRuntimeImportFile(importModule, importModule, file)
	if err != nil {
		return err
	}

	switch kind {
	case core.PersistORMKind:
		cfg := r.Config.GetPersistOrm(annot.Capability.ID)
		if cfg.Type == "cockroachdb_serverless" {
			oldNodeContent := annot.Node.Content()
			newNodeContent := sequelizeReplaceRE.ReplaceAllString(oldNodeContent, "new cockroachdbSequelize.Sequelize(")

			if err := file.ReplaceNodeContent(annot.Node, newNodeContent); err != nil {
				return err
			}

			importLine := `const cockroachdbSequelize = require('sequelize-cockroachdb');`
			if !strings.Contains(string(file.Program()), importLine) {
				buf := new(bytes.Buffer)
				buf.WriteString(importLine)
				buf.WriteRune('\n')
				buf.Write(file.Program())
				if err := file.Reparse(buf.Bytes()); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func (r *AwsRuntime) AddKvRuntimeFiles(unit *core.ExecutionUnit) error {
	return r.AddRuntimeFiles(unit, kvRuntimeFiles)
}

type FsTemplateData struct {
	BucketNameEnvVar string
}

func (r *AwsRuntime) AddFsRuntimeFiles(unit *core.ExecutionUnit, bucketNameEnvVar string, id string) error {
	templateData := FsTemplateData{BucketNameEnvVar: bucketNameEnvVar}
	content, err := fsRuntimeFiles.ReadFile("fs.js.tmpl")
	if err != nil {
		return err
	}
	err = javascript.AddRuntimeFile(unit, templateData, fmt.Sprintf("fs_%s.js.tmpl", sanitization.IdentifierSanitizer.Apply(id)), content)
	return err
}

func (r *AwsRuntime) AddSecretRuntimeFiles(unit *core.ExecutionUnit) error {
	return r.AddRuntimeFiles(unit, secretRuntimeFiles)
}

func (r *AwsRuntime) AddOrmRuntimeFiles(unit *core.ExecutionUnit) error {
	return r.AddRuntimeFiles(unit, ormRuntimeFiles)
}

func (r *AwsRuntime) AddRedisNodeRuntimeFiles(unit *core.ExecutionUnit) error {
	return r.AddRuntimeFiles(unit, redisNodeRuntimeFiles)
}

func (r *AwsRuntime) AddRedisClusterRuntimeFiles(unit *core.ExecutionUnit) error {
	return r.AddRuntimeFiles(unit, redisClusterRuntimeFiles)
}

func (r *AwsRuntime) AddPubsubRuntimeFiles(unit *core.ExecutionUnit) error {
	unit.EnvironmentVariables.Add(core.InternalStorageVariable)
	err := r.AddFsRuntimeFiles(unit, core.InternalStorageVariable.Name, "payload")
	if err != nil {
		return err
	}

	return r.AddRuntimeFiles(unit, pubsubRuntimeFiles)
}

func (r *AwsRuntime) AddProxyRuntimeFiles(unit *core.ExecutionUnit, proxyType string) error {
	var proxyFile []byte
	unitType := r.Config.GetResourceType(unit)
	switch proxyType {
	case "eks":
		proxyFile = proxyEks
	case "ecs":
		proxyFile = proxyFargate
	case "apprunner":
		proxyFile = proxyApprunner
	case "lambda":
		proxyFile = proxyLambda

		// We also need to add the Fs files because exec to exec calls in aws use s3
		unit.EnvironmentVariables.Add(core.InternalStorageVariable)
		err := r.AddFsRuntimeFiles(unit, core.InternalStorageVariable.Name, "payload")
		if err != nil {
			return err
		}
	default:
		return errors.Errorf("unsupported execution unit type: '%s'", unitType)
	}

	err := r.AddRuntimeFile(unit, proxyType+"_proxy.js.tmpl", proxyFile)
	if err != nil {
		return err
	}

	return nil
}

func (r *AwsRuntime) AddExecRuntimeFiles(unit *core.ExecutionUnit, result *core.CompilationResult, deps *core.Dependencies) error {
	var DockerFile, Dispatcher []byte
	unitType := r.Config.GetResourceType(unit)
	switch unitType {
	case "ecs", "eks", "apprunner":
		DockerFile = dockerfileFargate
		Dispatcher = dispatcherFargate
	case "lambda":
		DockerFile = dockerfileLambda
		Dispatcher = dispatcherLambda

		unit.EnvironmentVariables.Add(core.InternalStorageVariable)
		err := r.AddFsRuntimeFiles(unit, core.InternalStorageVariable.Name, "payload")
		if err != nil {
			return err
		}
	default:
		return errors.Errorf("unsupported execution unit type: '%s'", unitType)
	}

	templateData := TemplateData{
		TemplateConfig: r.TemplateConfig,
		ExecUnitName:   unit.Name,
	}

	exposeData, err := getExposeTemplateData(unit, result, deps)
	if err != nil {
		return err
	}
	templateData.Expose = exposeData

	pjsonPath := ""
	for path, f := range unit.Files() {
		if filepath.Base(f.Path()) == "package.json" {
			pjsonPath = path
		}
	}
	if pjsonPath == "" {
		return errors.Errorf("No `package.json` found for execution unit %s", unit.Name)
	}
	templateData.ProjectFilePath = pjsonPath
	if pjson := unit.Get(pjsonPath); pjson != nil {
		pfile := pjson.(*javascript.PackageFile)
		if mainRaw, ok := pfile.Content.OtherFields["main"]; ok {
			err := json.Unmarshal(mainRaw, &templateData.MainModule)
			if err != nil {
				return errors.Wrap(err, "could not unmarshal 'main' from package.json")
			}
			files := make(map[string]core.File)
			for _, f := range unit.Files() {
				files[f.Path()] = f
			}
			f, _ := javascript.FindFileForImport(files, ".", templateData.MainModule)
			if f != nil {
				zap.S().Debugf("Found 'main' from package.json: %s", templateData.MainModule)
			} else {
				// The main file isn't for this execution unit. This can happen if the main module
				// has a specific execution unit annotation. In that case, just skip its import
				// by zeroing out the field.
				templateData.MainModule = ""
				zap.S().Debugf("Skipping 'main' from package.json: %s due to not in unit %s", templateData.MainModule, unit.Name)
			}
		}
	}

	err = javascript.AddRuntimeFiles(unit, ExecRuntimeFiles, templateData)
	if err != nil {
		return err
	}

	if runtime.ShouldOverrideDockerfile(unit) {
		err = javascript.AddRuntimeFile(unit, templateData, "Dockerfile.tmpl", DockerFile)
		if err != nil {
			return err
		}
	}

	err = javascript.AddRuntimeFile(unit, templateData, "dispatcher.js.tmpl", Dispatcher)
	return err
}

func getExposeTemplateData(unit *core.ExecutionUnit, result *core.CompilationResult, deps *core.Dependencies) (ExposeTemplateData, error) {
	upstreamGateways := core.FindUpstreamGateways(unit, result, deps)

	var sourceGateway *core.Gateway
	for _, gw := range upstreamGateways {
		if sourceGateway != nil && (sourceGateway.DefinedIn != gw.DefinedIn || sourceGateway.ExportVarName != gw.ExportVarName) {
			return ExposeTemplateData{},
				errors.Errorf("multiple gateways cannot target different web applications in the same execution unit: [%s -> %s],[%s -> %s]",
					gw.Name, unit.Name,
					sourceGateway.Name, unit.Name)
		}
		sourceGateway = gw
	}

	exposeData := ExposeTemplateData{}
	if sourceGateway != nil {
		exposeData.AppModule = sourceGateway.DefinedIn
		exposeData.ExportedAppVar = sourceGateway.ExportVarName
	}
	return exposeData, nil
}

func (r *AwsRuntime) AddRuntimeFiles(unit *core.ExecutionUnit, files embed.FS) error {
	templateData := TemplateData{
		TemplateConfig:     r.TemplateConfig,
		ExecUnitName:       unit.Name,
		PayloadsBucketName: resources.SanitizeS3BucketName(r.Config.AppName),
	}
	err := javascript.AddRuntimeFiles(unit, files, templateData)
	return err
}

func (r *AwsRuntime) AddRuntimeFile(unit *core.ExecutionUnit, path string, content []byte) error {
	templateData := TemplateData{
		TemplateConfig:     r.TemplateConfig,
		ExecUnitName:       unit.Name,
		PayloadsBucketName: resources.SanitizeS3BucketName(r.Config.AppName),
	}
	err := javascript.AddRuntimeFile(unit, templateData, path, content)
	return err
}
