package kubernetes

import (
	"bytes"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/klothoplatform/klotho/pkg/annotation"
	"github.com/klothoplatform/klotho/pkg/config"
	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/infra/kubernetes/helm"
	yamlLang "github.com/klothoplatform/klotho/pkg/lang/yaml"
	"github.com/klothoplatform/klotho/pkg/multierr"
	"helm.sh/helm/v3/pkg/chart"
	"sigs.k8s.io/yaml"

	"go.uber.org/zap"
)

const (
	KubernetesType = "kubernetes"
)

type Kubernetes struct {
	Config     *config.Application
	log        *zap.SugaredLogger
	helmHelper *helm.HelmHelper
}

func (p Kubernetes) Name() string { return "kubernetes" }

func (p Kubernetes) Translate(constructGraph *core.ConstructGraph, dag *core.ResourceGraph) error {
	var errs multierr.Error
	p.log = zap.L().Sugar()
	helmHelper, err := helm.NewHelmHelper()
	if err != nil {
		return err
	}
	p.helmHelper = helmHelper

	klothoCharts, err := p.getKlothoCharts(constructGraph)
	if err != nil {
		return err
	}

	// For exec units that specify their own chart, we want to render and replace
	for dir, khChart := range klothoCharts {
		dirToLoad := filepath.Join(p.Config.Path, dir)
		chartContent, err := p.helmHelper.LoadChart(dirToLoad)

		if err != nil {
			if err.Error() == "Chart.yaml file is missing" {
				var unitNames []string
				for _, eu := range khChart.ExecutionUnits {
					unitNames = append(unitNames, eu.Name)
				}

				chartContent = &chart.Chart{
					Metadata: &chart.Metadata{
						Name:        strings.ReplaceAll(strings.ToLower(strings.Join(unitNames, "")), "_", "-"),
						APIVersion:  "v2",
						AppVersion:  "0.0.1",
						Version:     "0.0.1",
						KubeVersion: ">= 1.19.0-0",
						Type:        "application",
					},
				}
			} else {
				errs.Append(err)
				continue
			}
		}
		khChart.Name = chartContent.Name()
		values := make(map[string]interface{})
		if len(khChart.ValuesFiles) > 0 {
			values, err = helm.MergeValues(khChart.ValuesFiles)
			if err != nil {
				errs.Append(err)
				continue
			}
		}

		renderedFiles, err := p.helmHelper.GetRenderedTemplates(chartContent, values, "default")
		if err != nil {
			errs.Append(err)
			continue
		}
		khChart.Files = append(khChart.Files, renderedFiles...)

		err = khChart.AssignFilesToUnits()
		if err != nil {
			errs.Append(err)
			continue
		}

		for _, unit := range khChart.ExecutionUnits {
			eu, ok := core.GetConstruct[*core.ExecutionUnit](constructGraph, core.ResourceId{
				Provider: core.AbstractConstructProvider,
				Type:     annotation.ExecutionUnitCapability,
				Name:     unit.Name,
			})
			if !ok {
				return fmt.Errorf("unable to handle nonexistent execution unit: %s", unit.Name)
			}

			cfg := p.Config.GetExecutionUnit(unit.Name)
			execUnitValues, err := khChart.handleExecutionUnit(unit, eu, cfg, constructGraph)
			if err != nil {
				errs.Append(err)
				continue
			}
			khChart.ProviderValues = append(khChart.ProviderValues, execUnitValues...)
		}
		output, err := yaml.Marshal(chartContent.Metadata)
		if err != nil {
			errs.Append(err)
		}
		chartFile, err := yamlLang.NewFile(fmt.Sprintf("%s/Chart.yaml", khChart.Name), bytes.NewBuffer(output))
		if err != nil {
			errs.Append(err)
		}

		khChart.Files = append(khChart.Files, chartFile)

		dag.AddResource(khChart)
	}

	return errs.ErrOrNil()
}

func (p *Kubernetes) setHelmChartDirectory(path string, cfg *config.ExecutionUnit, unitName string) (bool, error) {
	extension := filepath.Ext(path)
	if extension != ".yaml" && extension != ".yml" {
		return false, nil
	}
	relPath := strings.TrimSuffix(path, extension)
	if strings.HasSuffix(relPath, "Chart") && cfg.HelmChartOptions.Directory == "" {
		chartDirectory, err := filepath.Rel(p.Config.Path, filepath.Dir(path))
		if err != nil {
			return false, err
		}
		p.log.Infof("Setting chart directory as %s, for execution unit %s", chartDirectory, unitName)
		cfg.HelmChartOptions.Directory = chartDirectory
		return true, nil
	}
	return false, nil
}

// getKlothoCharts gets all the main Chart.yaml Helm charts for the compilation. It returns a map of Helm charts for
// each execution unit, keyed by their directory (relative to the source root).
//
//   - Side effect: If the user has provided [config.HelmChartOptions] with an empty Directory for a given execution
//     unit, this method will look for a Chart.yaml next to whatever file declares the execution unit, and set the
//     Directory field on the exec unit's [config.HelmChartOptions] to that chart's directory.
func (p *Kubernetes) getKlothoCharts(constructGraph *core.ConstructGraph) (map[string]*HelmChart, error) {
	var errs multierr.Error
	klothoCharts := make(map[string]*HelmChart)
	for _, unit := range core.GetConstructsOfType[*core.ExecutionUnit](constructGraph) {
		cfg := p.Config.GetExecutionUnit(unit.ID)

		if cfg.HelmChartOptions.Directory == "" {
			for _, f := range unit.GetDeclaringFiles() {

				caps := f.Annotations()
				for _, annot := range caps {
					cap := annot.Capability
					if cap.Name == annotation.ExecutionUnitCapability && cap.ID == unit.ID {
						set, err := p.setHelmChartDirectory(f.Path(), &cfg, unit.ID)
						if err != nil {
							errs.Append(err)
						}
						if set {
							break
						}
					}
				}

			}
		}

		if cfg.Type == KubernetesType {
			chartDir := cfg.HelmChartOptions.Directory
			valuesFiles := cfg.HelmChartOptions.ValuesFiles
			khChart, ok := klothoCharts[chartDir]
			if !ok {
				khChart = &HelmChart{
					ValuesFiles:   valuesFiles,
					Directory:     chartDir,
					ConstructRefs: make(core.AnnotationKeySet),
					Values:        make(map[string]any),
				}
				khChart.ConstructRefs.Add(unit.Provenance())
				klothoCharts[chartDir] = khChart
			} else {
				foundDifference := false
				for _, chartFile := range khChart.ValuesFiles {
					fileFound := false
					for _, cfgFile := range valuesFiles {
						if cfgFile == chartFile {
							fileFound = true
						}
					}
					if !fileFound {
						foundDifference = true
					}
				}
				if foundDifference {
					p.log.Warnf("Found Conflicting Helm Values files, %s and %s, for helm chart in directory %s. Using %s",
						khChart.ValuesFiles, cfg.HelmChartOptions.ValuesFiles, cfg.HelmChartOptions.Directory, khChart.ValuesFiles)
				}
			}

			khChart.ExecutionUnits = append(khChart.ExecutionUnits, &HelmExecUnit{Name: unit.ID, Namespace: "default"})
			khChart.ConstructRefs.Add(unit.AnnotationKey)
		}
	}
	return klothoCharts, errs.ErrOrNil()
}
