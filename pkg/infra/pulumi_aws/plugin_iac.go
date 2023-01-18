package pulumi_aws

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/bmatcuk/doublestar/v4"
	"io"
	"io/fs"
	"os"
	"strings"
	"text/template"

	"github.com/klothoplatform/klotho/pkg/config"
	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/logging"
	"github.com/klothoplatform/klotho/pkg/provider/aws"
	"github.com/pelletier/go-toml/v2"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

type Plugin struct {
	Config *config.Application
}

func (p Plugin) Name() string { return "Pulumi:AWS" }

func (p Plugin) Transform(result *core.CompilationResult, deps *core.Dependencies) error {
	infraFiles := &core.InfraFiles{
		Name: "Pulumi (AWS)",
	}

	buf := new(bytes.Buffer)
	var err error
	var ext string
	switch p.Config.Format {
	case "toml":
		ext = "toml"
		enc := toml.NewEncoder(buf)
		enc.SetArraysMultiline(true)
		enc.SetIndentTables(true)
		err = enc.Encode(p.Config)

	case "json":
		ext = "json"
		err = json.NewEncoder(buf).Encode(p.Config)

	case "yaml":
		ext = "yaml"
		err = yaml.NewEncoder(buf).Encode(p.Config)

	default:
		err = errors.Errorf("unsupported config format: %s", p.Config.Format)
	}
	if err != nil {
		return err
	}
	awsData := result.GetFirstResource(aws.AwsTemplateDataKind)
	data, ok := awsData.(*aws.TemplateData)
	if !ok {
		return errors.Errorf("Invalid template data '%s' for iac plugin '%s'", awsData.Key(), p.Name())
	}

	configFile := &core.RawFile{
		FPath:   fmt.Sprintf("klotho.%s", ext),
		Content: buf.Bytes(),
	}

	infraFiles.Add(configFile)
	data.ConfigPath = configFile.Path()

	addTemplate := func(name string, t *template.Template) {
		if err != nil {
			return
		}
		buf := new(bytes.Buffer)
		err = t.Execute(buf, data)
		if err != nil {
			err = core.WrapErrf(err, "error executing template %s", name)
			return
		}
		infraFiles.Add(&core.RawFile{
			FPath:   name,
			Content: buf.Bytes(),
		})
	}
	addTemplate("index.ts", index)
	addTemplate("Pulumi.yaml", pulumiBase)

	if err != nil {
		return err
	}

	buf = new(bytes.Buffer)
	err = pulumiStack.Execute(buf, data)
	if err == nil {
		stack := &StackFile{
			RawFile: core.RawFile{
				FPath:   fmt.Sprintf("Pulumi.%s.yaml", p.Config.AppName),
				Content: buf.Bytes(),
			},
			AppName: data.AppName,
		}
		infraFiles.Add(stack)
	}

	addFile := func(name string) {
		if err != nil {
			return
		}

		var content []byte
		content, err = files.ReadFile(name)
		if err == nil {
			infraFiles.Add(&core.RawFile{
				FPath:   name,
				Content: content,
			})
		}
	}

	if len(data.CloudfrontDistributions) > 0 {
		addFile("iac/cloudfront.ts")
	}

	if len(data.StaticUnits) > 0 {
		addFile("iac/static_s3_website.ts")
	}

	addFile("deploylib.ts")
	addFile("package.json")
	addFile("tsconfig.json")
	addFile("iac/elasticache.ts")
	addFile("iac/memorydb.ts")
	addFile("iac/eks.ts")
	addFile("iac/kubernetes.ts")
	addFile("iac/cockroachdb.ts")
	addFile("iac/analytics.ts")
	addFile("iac/load_balancing.ts")
	addFile("iac/k8s/horizontal-pod-autoscaling.ts")
	addFile("iac/k8s/helm_chart.ts")
	addFile("iac/k8s/add_ons/metrics_server/index.ts")
	addFile("iac/k8s/add_ons/alb_controller/target_group_binding.yaml")
	addFile("iac/k8s/add_ons/alb_controller/index.ts")
	addFile("iac/k8s/add_ons/cloud_map_controller/cloudmap_cluster_set.yaml")
	addFile("iac/k8s/add_ons/cloud_map_controller/cloudmap_export_service.yaml")
	addFile("iac/k8s/add_ons/cloud_map_controller/index.ts")
	addFile("iac/k8s/add_ons/external_dns/index.ts")
	addFile("iac/k8s/add_ons/index.ts")

	addDir := func(dir string, exclusions ...string) {
		var unreadEntries []string
		dirContents, err := files.ReadDir(dir)

		addEntries := func(parentDir string, entries []fs.DirEntry) {
			for _, entry := range entries {
				dirSuffix := ""
				if entry.IsDir() {
					dirSuffix = "/"
				}
				path := strings.TrimSuffix(parentDir, "/") + "/" + entry.Name() + dirSuffix
				shouldInclude := true
				for _, exclusion := range exclusions {
					if shouldExclude, _ := doublestar.Match(exclusion, path); shouldExclude {
						shouldInclude = false
					}
				}
				if shouldInclude {
					unreadEntries = append(unreadEntries, path)
				}
			}
		}
		addEntries(dir, dirContents)

		for len(unreadEntries) > 0 {
			if err != nil {
				return
			}

			entry := unreadEntries[0]
			unreadEntries = unreadEntries[1:]
			if strings.HasSuffix(entry, "/") {
				var childEntries []os.DirEntry
				childEntries, err = files.ReadDir(strings.TrimSuffix(entry, "/"))
				addEntries(entry, childEntries)

			} else {
				addFile(entry)
			}
		}
	}

	addDir("iac/sanitization", "**/*.{test,spec}.{ts,js}")

	if err != nil {
		return err
	}

	result.Add(infraFiles)

	return nil
}

type StackFile struct {
	core.RawFile
	AppName string
}

func (s *StackFile) WriteTo(w io.Writer) (int64, error) {
	if f, ok := w.(*os.File); ok {
		outDir := strings.TrimSuffix(f.Name(), s.FPath)
		zap.L().
			With(logging.FileField(s)).Sugar().
			Infof("Make sure to run `pulumi config set aws:region YOUR_REGION --cwd '%s' -s '%s'` to configure the target AWS region.", outDir, s.AppName)
	}
	return s.RawFile.WriteTo(w)
}

func (s *StackFile) Overwrite(f *os.File) bool {
	if f != nil {
		zap.L().Debug("Detected existing Pulumi stack yaml, skipping")
		return false
	}
	return true
}
