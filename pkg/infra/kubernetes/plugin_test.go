package kubernetes

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/klothoplatform/klotho/pkg/annotation"
	"github.com/klothoplatform/klotho/pkg/config"
	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/smacker/go-tree-sitter/javascript"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func Test_setHelmChartDirectory(t *testing.T) {

	tests := []struct {
		name     string
		path     string
		cfg      *config.ExecutionUnit
		unitName string
		want     string
		isSet    bool
	}{
		{
			name: "happy path yaml",
			path: "somedir/Chart.yaml",
			cfg: &config.ExecutionUnit{
				HelmChartOptions: &config.HelmChartOptions{
					Install: true,
				},
			},
			unitName: "testUnit",
			want:     "somedir",
			isSet:    true,
		},
		{
			name: "happy path yml",
			path: "somedir/Chart.yml",
			cfg: &config.ExecutionUnit{
				HelmChartOptions: &config.HelmChartOptions{
					Install: true,
				},
			},
			unitName: "testUnit",
			want:     "somedir",
			isSet:    true,
		},
		{
			name: "no install",
			path: "somedir/Chart.yaml",
			cfg: &config.ExecutionUnit{
				HelmChartOptions: &config.HelmChartOptions{},
			},
			unitName: "testUnit",
			want:     "",
			isSet:    false,
		},
		{
			name: "directory override",
			path: "somedir/Chart.yaml",
			cfg: &config.ExecutionUnit{
				HelmChartOptions: &config.HelmChartOptions{
					Directory: "override",
					Install:   true,
				},
			},
			unitName: "testUnit",
			want:     "override",
			isSet:    false,
		},
		{
			name: "non yaml file no action",
			path: "somedir/Chart.something",
			cfg: &config.ExecutionUnit{
				HelmChartOptions: &config.HelmChartOptions{
					Install: true,
				},
			},
			unitName: "testUnit",
			want:     "",
			isSet:    false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)

			k := &Kubernetes{Config: &config.Application{Path: "."}, log: zap.L().Sugar()}

			set, err := k.setHelmChartDirectory(tt.path, tt.cfg, tt.unitName)
			if !assert.NoError(err) {
				return
			}
			assert.Equal(tt.want, tt.cfg.HelmChartOptions.Directory)
			assert.Equal(tt.isSet, set)
		})
	}
}

func Test_setValuesFile(t *testing.T) {

	tests := []struct {
		name     string
		path     string
		cfg      *config.ExecutionUnit
		unitName string
		want     string
	}{
		{
			name: "happy path yaml",
			path: "somedir/values.yaml",
			cfg: &config.ExecutionUnit{
				HelmChartOptions: &config.HelmChartOptions{
					Install:   true,
					Directory: "somedir",
				},
			},
			unitName: "testUnit",
			want:     "somedir/values.yaml",
		},
		{
			name: "happy path yml",
			path: "somedir/values.yml",
			cfg: &config.ExecutionUnit{
				HelmChartOptions: &config.HelmChartOptions{
					Install:   true,
					Directory: "somedir",
				},
			},
			unitName: "testUnit",
			want:     "somedir/values.yml",
		},
		{
			name: "happy path no directory",
			path: "somedir/values.yml",
			cfg: &config.ExecutionUnit{
				HelmChartOptions: &config.HelmChartOptions{
					Install: true,
				},
			},
			unitName: "testUnit",
			want:     "somedir/values.yml",
		},
		{
			name: "no install",
			path: "somedir/values.yaml",
			cfg: &config.ExecutionUnit{
				HelmChartOptions: &config.HelmChartOptions{},
			},
			unitName: "testUnit",
			want:     "",
		},
		{
			name: "values override",
			path: "somedir/values.yaml",
			cfg: &config.ExecutionUnit{
				HelmChartOptions: &config.HelmChartOptions{
					ValuesFile: "override/values.yaml",
					Install:    true,
				},
			},
			unitName: "testUnit",
			want:     "override/values.yaml",
		},
		{
			name: "non yaml file no action",
			path: "somedir/values.something",
			cfg: &config.ExecutionUnit{
				HelmChartOptions: &config.HelmChartOptions{
					Install: true,
				},
			},
			unitName: "testUnit",
			want:     "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)

			k := &Kubernetes{Config: &config.Application{Path: "."}, log: zap.L().Sugar()}

			err := k.setValuesFile(tt.path, tt.cfg, tt.unitName)
			if !assert.NoError(err) {
				return
			}
			assert.Equal(tt.want, tt.cfg.HelmChartOptions.ValuesFile)
		})
	}
}

type testCapabilityFinder struct{}

var testLang = core.SourceLanguage{
	ID:               core.LanguageId("plugin_exec_split_test"),
	Sitter:           javascript.GetLanguage(), // we don't actually care about the language, but we do need a non-nil one
	CapabilityFinder: &testCapabilityFinder{},
}

func (t *testCapabilityFinder) FindAllCapabilities(sf *core.SourceFile) (core.AnnotationMap, error) {
	body := string(sf.Program())
	annots := make(core.AnnotationMap)
	if body != "" {
		annots.Add(&core.Annotation{
			Capability: &annotation.Capability{
				Name:       annotation.ExecutionUnitCapability,
				ID:         body,
				Directives: annotation.Directives{"id": body},
			},
		})
	}
	return annots, nil
}

func Test_getKlothoCharts(t *testing.T) {
	type result struct {
		klothoCharts map[string]KlothoHelmChart
		chartsUnits  map[string][]string
	}
	tests := []struct {
		name      string
		fileUnits []map[string]string
		cfg       *config.Application
		want      result
	}{
		{
			name: "single unit test",
			fileUnits: []map[string]string{{
				"chart/Chart.yaml":         `main0`,
				"chart/templates/unitFile": ``,
				"unitFile":                 `main0`,
				"chart/crds/crd.yaml":      ``,
				"chart/values.yaml":        ``,
			}},
			want: result{
				klothoCharts: map[string]KlothoHelmChart{
					"chart": {
						Directory:  "chart",
						ValuesFile: "chart/values.yaml",
					},
				},
				chartsUnits: map[string][]string{
					"chart": {"main0"},
				},
			},
			cfg: &config.Application{
				Path: ".",
				ExecutionUnits: map[string]*config.ExecutionUnit{
					"main0": {
						HelmChartOptions: &config.HelmChartOptions{
							Install: true,
						},
					},
				},
			},
		},
		{
			name: "bundled unit tests",
			fileUnits: []map[string]string{{
				"chart/Chart.yaml":         ``,
				"chart/templates/unitFile": ``,
				"unitFile":                 `main0`,
				"chart/crds/crd.yaml":      ``,
				"chart/values.yaml":        ``,
			}, {
				"chart/Chart.yaml":         ``,
				"chart/templates/unitFile": ``,
				"otherUnitFile":            `main1`,
				"chart/crds/crd.yaml":      ``,
				"chart/values.yaml":        ``,
			}},
			want: result{
				klothoCharts: map[string]KlothoHelmChart{
					"chart": {
						Directory:  "chart",
						ValuesFile: "chart/values.yaml",
					},
				},
				chartsUnits: map[string][]string{
					"chart": {"main0", "main1"},
				},
			},
			cfg: &config.Application{
				Path: ".",
				ExecutionUnits: map[string]*config.ExecutionUnit{
					"main0": {
						HelmChartOptions: &config.HelmChartOptions{
							Install:   true,
							Directory: "chart",
						},
					},
					"main1": {
						HelmChartOptions: &config.HelmChartOptions{
							Install:   true,
							Directory: "chart",
						},
					},
				},
			},
		},
		{
			name: "seperate unit tests",
			fileUnits: []map[string]string{{
				"chart/Chart.yaml":         ``,
				"chart/templates/unitFile": ``,
				"unitFile":                 `main0`,
				"chart/crds/crd.yaml":      ``,
				"chart/values.yaml":        ``,
			}, {
				"chart2/Chart.yaml":         ``,
				"chart2/templates/unitFile": ``,
				"otherUnitFile":             `main1`,
				"chart2/crds/crd.yaml":      ``,
				"chart2/values.yaml":        ``,
			}},
			want: result{
				klothoCharts: map[string]KlothoHelmChart{
					"": {},
				},
				chartsUnits: map[string][]string{
					"": {"main0", "main1"},
				},
			},
			cfg: &config.Application{
				Path: ".",
				ExecutionUnits: map[string]*config.ExecutionUnit{
					"main0": {
						HelmChartOptions: &config.HelmChartOptions{
							Install: true,
						},
					},
					"main1": {
						HelmChartOptions: &config.HelmChartOptions{
							Install: true,
						},
					},
				},
			},
		},
		{
			name: "no install no chart",
			fileUnits: []map[string]string{{
				"chart/Chart.yaml":         ``,
				"chart/templates/unitFile": ``,
				"unitFile":                 `main0`,
				"chart/crds/crd.yaml":      ``,
				"chart/values.yaml":        ``,
			}},
			want: result{},
			cfg: &config.Application{
				Path: ".",
				ExecutionUnits: map[string]*config.ExecutionUnit{
					"main0": {
						HelmChartOptions: &config.HelmChartOptions{},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)

			result := &core.CompilationResult{}
			inputFiles := &core.InputFiles{}
			for idx, fileUnit := range tt.fileUnits {
				execUnitName := fmt.Sprintf("main%s", strconv.Itoa(idx))
				testUnit := core.ExecutionUnit{Name: execUnitName}
				for path, file := range fileUnit {
					f, err := core.NewSourceFile(path, strings.NewReader(file), testLang)
					if assert.Nil(err) {
						testUnit.Add(f)
						inputFiles.Add(f)
					}
				}
				result.Add(&testUnit)
			}
			result.Add(inputFiles)

			k := &Kubernetes{Config: tt.cfg, log: zap.L().Sugar()}

			klothoCharts, err := k.getKlothoCharts(result)
			if !assert.NoError(err) {
				return
			}
			fmt.Println(klothoCharts)
			for dir, c := range tt.want.klothoCharts {
				resultChart := klothoCharts[dir]
				assert.Equal(c.Directory, resultChart.Directory)
				assert.Equal(c.ValuesFile, resultChart.ValuesFile)

				assert.Len(resultChart.ExecutionUnits, len(tt.want.chartsUnits[dir]))
				for _, u := range resultChart.ExecutionUnits {
					found := false
					for _, expectedU := range tt.want.chartsUnits[dir] {
						if expectedU == u.Name {
							found = true
						}
						assert.Equal("default", u.Namespace)
					}
					if !assert.True(found) {
						return
					}
					res := result.Get(core.ResourceKey{Kind: core.ExecutionUnitKind, Name: u.Name})
					unit, ok := res.(*core.ExecutionUnit)
					if !assert.True(ok) {
						return
					}
					for _, f := range unit.Files() {
						assert.True(!strings.HasPrefix(f.Path(), c.Directory+string(os.PathSeparator)))
					}
				}
			}
		})
	}
}
