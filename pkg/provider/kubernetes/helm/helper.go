package helm

import (
	"io"
	"log"
	"os"
	"strings"

	yamlLang "github.com/klothoplatform/klotho/pkg/lang/yaml"
	"go.uber.org/zap"

	klotho_io "github.com/klothoplatform/klotho/pkg/io"

	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/chartutil"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/engine"
)

type HelmHelper struct {
	cfg *action.Configuration
}

var (
	notesFileSuffix = "NOTES.txt"
)

func NewHelmHelper() (*HelmHelper, error) {
	h := &HelmHelper{}
	settings := cli.New()

	actionConfig := new(action.Configuration)
	// You can pass an empty string instead of settings.Namespace() to list
	// all namespaces
	if err := actionConfig.Init(settings.RESTClientGetter(), settings.Namespace(), os.Getenv("HELM_DRIVER"), zap.S().Debugf); err != nil {
		return nil, err
	}
	h.cfg = actionConfig
	return h, nil
}

func (h *HelmHelper) LoadChart(directory string) (*chart.Chart, error) {
	writer := log.Writer()
	// We want to discard log output, otherwise on LoadDir failures we will see a lot of logs about symlinks to stdout, making it a bad UX
	log.SetOutput(io.Discard)
	chart, err := loader.LoadDir(directory)
	log.SetOutput(writer)
	if err != nil {
		return nil, err
	}
	return chart, nil
}

func GetValues(valuesFile string) (map[string]interface{}, error) {
	values, err := chartutil.ReadValuesFile(valuesFile)
	if err != nil {
		return nil, err
	}
	return values.AsMap(), nil
}

func MergeValues(valuesFiles []string) (map[string]interface{}, error) {
	base := map[string]interface{}{}

	// User specified a values files via -f/--values
	for _, filePath := range valuesFiles {
		currentMap, err := GetValues(filePath)
		if err != nil {
			return nil, err
		}
		// Merge with the previous map
		base = mergeMaps(base, currentMap)
	}
	return base, nil
}

func mergeMaps(a, b map[string]interface{}) map[string]interface{} {
	out := make(map[string]interface{}, len(a))
	for k, v := range a {
		out[k] = v
	}
	for k, v := range b {
		if v, ok := v.(map[string]interface{}); ok {
			if bv, ok := out[k]; ok {
				if bv, ok := bv.(map[string]interface{}); ok {
					out[k] = mergeMaps(bv, v)
					continue
				}
			}
		}
		out[k] = v
	}
	return out
}

func (h *HelmHelper) GetRenderedTemplates(ch *chart.Chart, vals map[string]interface{}, namespace string) ([]klotho_io.File, error) {

	renderedFiles := []klotho_io.File{}

	client := action.NewInstall(h.cfg)
	client.DryRun = true
	client.ReleaseName = "release-name"
	client.ClientOnly = true
	client.APIVersions = chartutil.VersionSet([]string{})
	client.IncludeCRDs = true
	client.Namespace = namespace

	options := chartutil.ReleaseOptions{
		Name:      client.ReleaseName,
		Namespace: client.Namespace,
		Revision:  1,
		IsInstall: true,
		IsUpgrade: false,
	}

	valuesToRender, err := chartutil.ToRenderValues(ch, vals, options, nil)
	if err != nil {
		return nil, err
	}

	files, err := engine.Render(ch, valuesToRender)
	if err != nil {
		return nil, err
	}

	for p, f := range files {
		if strings.HasSuffix(p, notesFileSuffix) {
			continue
		}
		if strings.TrimSpace(f) == "" {
			continue
		}
		newF, err := yamlLang.NewFile(p, strings.NewReader(f))
		if err != nil {
			return nil, err
		}
		renderedFiles = append(renderedFiles, newF)
	}
	for _, crd := range ch.CRDObjects() {
		newF, err := yamlLang.NewFile(crd.Filename, strings.NewReader(string(crd.File.Data)))
		if err != nil {
			return nil, err
		}
		renderedFiles = append(renderedFiles, newF)
	}

	return renderedFiles, nil
}
