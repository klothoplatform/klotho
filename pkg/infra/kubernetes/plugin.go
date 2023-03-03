package kubernetes

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/klothoplatform/klotho/pkg/annotation"
	"github.com/klothoplatform/klotho/pkg/config"
	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/infra/kubernetes/helm"
	yamlLang "github.com/klothoplatform/klotho/pkg/lang/yaml"
	"github.com/klothoplatform/klotho/pkg/multierr"
	"sigs.k8s.io/yaml"

	"go.uber.org/zap"
)

type Kubernetes struct {
	Config     *config.Application
	log        *zap.SugaredLogger
	helmHelper *helm.HelmHelper
}

func (p Kubernetes) Name() string { return "kubernetes" }

func (p Kubernetes) Transform(result *core.CompilationResult, deps *core.Dependencies) error {
	var errs multierr.Error
	p.log = zap.L().Sugar()
	helmHelper, err := helm.NewHelmHelper()
	if err != nil {
		return err
	}
	p.helmHelper = helmHelper

	klothoCharts, err := p.getKlothoCharts(result)
	if err != nil {
		return err
	}

	// For exec units that specify their own chart, we want to render and replace
	for dir, khChart := range klothoCharts {
		dirToLoad := filepath.Join(p.Config.Path, dir)
		chart, err := p.helmHelper.LoadChart(dirToLoad)
		if err != nil {
			errs.Append(err)
			continue
		}
		metadata := chart.Metadata
		khChart.Name = chart.Name()
		values := make(map[string]interface{})
		if len(khChart.ValuesFiles) > 0 {
			values, err = helm.MergeValues(khChart.ValuesFiles)
			if err != nil {
				errs.Append(err)
				continue
			}
		}

		renderedFiles, err := p.helmHelper.GetRenderedTemplates(chart, values, "default")
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
			res := result.Get(core.ResourceKey{Kind: core.ExecutionUnitKind, Name: unit.Name})
			eu, ok := res.(*core.ExecutionUnit)
			if !ok {
				return fmt.Errorf("unable to handle nonexistent execution unit: %s", unit.Name)
			}

			cfg := p.Config.GetExecutionUnit(unit.Name)
			execUnitValues, err := khChart.handleExecutionUnit(unit, eu, cfg, deps)
			if err != nil {
				errs.Append(err)
			}
			khChart.Values = append(khChart.Values, execUnitValues...)

		}
		output, err := yaml.Marshal(metadata)
		if err != nil {
			errs.Append(err)
		}
		chartFile, err := yamlLang.NewFile(fmt.Sprintf("%s/Chart.yaml", khChart.Name), bytes.NewBuffer(output))
		if err != nil {
			errs.Append(err)
		}
		khChart.Files = append(khChart.Files, chartFile)

		result.Add(&khChart)
	}

	return errs.ErrOrNil()
}

func (p *Kubernetes) setHelmChartDirectory(path string, cfg *config.ExecutionUnit, unitName string) (bool, error) {
	extension := filepath.Ext(path)
	if extension != ".yaml" && extension != ".yml" {
		return false, nil
	}
	relPath := strings.TrimSuffix(path, extension)
	if strings.HasSuffix(relPath, "Chart") &&
		(cfg.HelmChartOptions.Install && cfg.HelmChartOptions.Directory == "") {
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

func (p *Kubernetes) setValuesFile(path string, cfg *config.ExecutionUnit, unitName string) error {
	extension := filepath.Ext(path)
	if extension != ".yaml" && extension != ".yml" {
		return nil
	}
	relPath := strings.TrimSuffix(path, extension)
	if strings.HasSuffix(relPath, "values") &&
		(cfg.HelmChartOptions.Install && len(cfg.HelmChartOptions.ValuesFiles) == 0) {
		valuesFile, err := filepath.Rel(p.Config.Path, path)
		if err != nil {
			return err
		}
		p.log.Infof("Setting values file as %s, for execution unit %s", valuesFile, unitName)
		cfg.HelmChartOptions.ValuesFiles = []string{valuesFile}
	}
	return nil
}

func (p *Kubernetes) getKlothoCharts(result *core.CompilationResult) (map[string]KlothoHelmChart, error) {
	var errs multierr.Error
	klothoCharts := make(map[string]KlothoHelmChart)
	inputFiles := result.GetFirstResource(core.InputFilesKind)
	inputF, ok := inputFiles.(*core.InputFiles)
	if inputFiles == nil || !ok {
		return nil, nil
	}
	for _, res := range result.Resources() {
		key := res.Key()
		u, ok := res.(*core.ExecutionUnit)
		if !ok {
			continue
		}
		cfg := p.Config.GetExecutionUnit(key.Name)

		if cfg.HelmChartOptions == nil {
			continue
		}

		if cfg.HelmChartOptions.Directory == "" {
			for _, f := range u.GetDeclaringFiles() {

				caps := f.Annotations()
				for _, annot := range caps {
					cap := annot.Capability
					if cap.Name == annotation.ExecutionUnitCapability && cap.ID == u.Name {
						set, err := p.setHelmChartDirectory(f.Path(), &cfg, u.Name)
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

		for _, f := range inputF.Files() {
			chartDir := filepath.Clean(cfg.HelmChartOptions.Directory) + string(os.PathSeparator)
			if strings.HasPrefix(f.Path(), chartDir) {
				if len(cfg.HelmChartOptions.ValuesFiles) == 0 {
					err := p.setValuesFile(f.Path(), &cfg, u.Name)
					if err != nil {
						errs.Append(err)
					}
				}
				// We are removing any remaining chart files from the execution unit, since they are not necessary for our executable code and will be added to our generated helm chart.
				u.Remove(f.Path())
			}
		}

		if cfg.HelmChartOptions.Install {
			khChart, ok := klothoCharts[cfg.HelmChartOptions.Directory]
			if !ok {
				klothoCharts[cfg.HelmChartOptions.Directory] = KlothoHelmChart{
					ValuesFiles:    cfg.HelmChartOptions.ValuesFiles,
					ExecutionUnits: []*HelmExecUnit{{Name: u.Name, Namespace: "default"}},
					Directory:      cfg.HelmChartOptions.Directory,
				}
			} else {
				foundDifference := false
				for _, chartFile := range khChart.ValuesFiles {
					fileFound := false
					for _, cfgFile := range cfg.HelmChartOptions.ValuesFiles {
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
				khChart.ExecutionUnits = append(khChart.ExecutionUnits, &HelmExecUnit{Name: u.Name, Namespace: "default"})
				klothoCharts[cfg.HelmChartOptions.Directory] = khChart
			}
		}
	}
	return klothoCharts, errs.ErrOrNil()
}
