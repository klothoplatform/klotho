package kubernetes

import (
	"fmt"
	"os"
	"path/filepath"

	"go.uber.org/zap"
	apps "k8s.io/api/apps/v1"
	autoscaling "k8s.io/api/autoscaling/v2beta2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/klothoplatform/klotho/pkg/annotation"
	"github.com/klothoplatform/klotho/pkg/config"
	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/logging"
)

const HELM_CHART_TYPE = "helm_chart"

const PROVIDER = "kubernetes"

type HelmChart struct {
	Name           string
	Chart          string
	ValuesFiles    []string
	ExecutionUnits []*HelmExecUnit
	Directory      string
	Files          []core.File
	ProviderValues []HelmChartValue

	ConstructRefs    []core.AnnotationKey
	ClustersProvider core.Resource
	Repo             string
	Version          string
	Namespace        string
	Values           map[string]any
}

// Provider returns name of the provider the resource is correlated to
func (chart *HelmChart) Provider() string { return PROVIDER }

// KlothoConstructRef returns a slice containing the ids of any Klotho constructs is correlated to
func (chart *HelmChart) KlothoConstructRef() []core.AnnotationKey { return chart.ConstructRefs }

func (chart *HelmChart) Id() string {
	return fmt.Sprintf("%s:%s:%s", chart.Provider(), HELM_CHART_TYPE, chart.Name)
}

func (t *HelmChart) OutputTo(dest string) error {
	errs := make(chan error)
	files := t.Files
	for idx := range files {
		go func(f core.File) {
			path := filepath.Join(dest, "charts", f.Path())
			dir := filepath.Dir(path)
			err := os.MkdirAll(dir, 0777)
			if err != nil {
				errs <- err
				return
			}
			file, err := os.OpenFile(path, os.O_RDWR, 0777)
			if os.IsNotExist(err) {
				file, err = os.OpenFile(path, os.O_CREATE|os.O_WRONLY, 0777)
			}
			if err != nil {
				errs <- err
				return
			}
			_, err = f.WriteTo(file)
			defer file.Close()
			errs <- err
		}(files[idx])
	}

	for i := 0; i < len(files); i++ {
		err := <-errs
		if err != nil {
			return err
		}
	}
	return nil
}

func (t *HelmChart) AssignFilesToUnits() error {
	for _, unit := range t.ExecutionUnits {
		for _, f := range t.Files {
			ast, ok := f.(*core.SourceFile)
			if !ok {
				continue
			}
			log := zap.L().Sugar().With(logging.FileField(f), zap.String("unit", unit.Name))

			// setAst sets the given *core.SourceFile field (hence the double-pointer), as long as either (a) there's
			// only one exec unit or (b) there are multiple units, but this one's name matches the k8s object name.
			// It returns whether that condition matched.
			setAst := func(k8sObject metav1.Object, handle **core.SourceFile) bool {
				if len(t.ExecutionUnits) <= 1 || (k8sObject.GetName() == unit.Name) {
					log.Debugf("Found unit, %s's, pod manifest in file, %s", unit.Name, f.Path())
					*handle = ast
					return true
				}
				return false
			}

			obj, err := readFile(ast)
			if err != nil {
				return err
			}
			switch o := obj.(type) {
			case *corev1.Pod:
				if setAst(o, &unit.Pod) && unit.Deployment != nil {
					// Don't set this pod if there's already a spec for deployment. That means there's both a deployment
					// manifest and a pod manifest for the same exec unit, which is a confusing scenario (since a
					// deployment itself contains a pod spec). For now, we just disallow it.
					return fmt.Errorf("can not support multiple pod specifications for unit %s", unit.Name)
				}
			case *apps.Deployment:
				if setAst(o, &unit.Deployment) && unit.Pod != nil {
					// Don't set this deployment if there's already a spec for this pod. See comment above.
					return fmt.Errorf("can not support multiple pod specifications for unit %s", unit.Name)
				}
			case *autoscaling.HorizontalPodAutoscaler:
				setAst(o, &unit.HorizontalPodAutoscaler)
			case *corev1.ServiceAccount:
				setAst(o, &unit.ServiceAccount)
			case *corev1.Service:
				setAst(o, &unit.Service)
			default:
				log.Debug("Unrecognized type")
			}
		}
	}
	return nil
}

func (chart *HelmChart) handleExecutionUnit(unit *HelmExecUnit, eu *core.ExecutionUnit, cfg config.ExecutionUnit, constructGraph *core.ConstructGraph) ([]HelmChartValue, error) {
	values := []HelmChartValue{}

	if shouldTransformImage(eu) {
		if unit.Deployment != nil {
			deploymentValues, err := unit.transformDeployment(cfg)
			if err != nil {
				return nil, err
			}
			values = append(values, deploymentValues...)
		} else if unit.Pod != nil {
			podValues, err := unit.transformPod(cfg)
			if err != nil {
				return nil, err
			}
			values = append(values, podValues...)
		} else {
			deploymentValues, err := chart.addDeployment(unit, cfg)
			if err != nil {
				return nil, err
			}
			values = append(values, deploymentValues...)
		}
		if unit.Deployment != nil && cfg.GetExecutionUnitParamsAsKubernetes().HorizontalPodAutoScalingConfig.NotEmpty() {
			if unit.HorizontalPodAutoscaler != nil {
				hpaValues, err := unit.transformHorizontalPodAutoscaler(cfg)
				if err != nil {
					return nil, err
				}
				values = append(values, hpaValues...)
			} else {
				hpaValues, err := chart.addHorizontalPodAutoscaler(unit, cfg)
				if err != nil {
					return nil, err
				}
				values = append(values, hpaValues...)
			}
		}
	}
	if shouldTransformServiceAccount(eu) {
		if unit.ServiceAccount != nil {
			serviceAccountValues, err := unit.transformServiceAccount()
			if err != nil {
				return nil, err
			}
			values = append(values, serviceAccountValues...)
		} else {
			serviceAccountValues, err := chart.addServiceAccount(unit)
			if err != nil {
				return nil, err
			}
			values = append(values, serviceAccountValues...)
		}
	}
	upstreamValues, err := chart.handleUpstreamUnitDependencies(unit, constructGraph, cfg)
	if err != nil {
		return nil, err
	}
	values = append(values, upstreamValues...)

	unitEnvValues, err := unit.AddUnitsEnvironmentVariables(eu)
	if err != nil {
		return nil, err
	}
	values = append(values, unitEnvValues...)

	return values, nil
}

func (chart *HelmChart) handleUpstreamUnitDependencies(unit *HelmExecUnit, constructGraph *core.ConstructGraph, cfg config.ExecutionUnit) (values []HelmChartValue, err error) {
	sources := constructGraph.GetUpstreamConstructs(&core.ExecutionUnit{AnnotationKey: core.AnnotationKey{ID: unit.Name, Capability: annotation.ExecutionUnitCapability}})
	needService := false
	needsTargetGroupBinding := false
	needsServiceExport := false
	for _, source := range sources {
		if source.Provenance().Capability == annotation.ExposeCapability {
			needService = true
			needsTargetGroupBinding = true
		}
		if source.Provenance().Capability == annotation.ExecutionUnitCapability {
			needService = true
			needsServiceExport = true
		}
	}
	if needService {
		if unit.Service != nil {
			serviceValues, err := unit.transformService(cfg)
			if err != nil {
				return nil, err
			}
			values = append(values, serviceValues...)
		} else {
			serviceValues, err := chart.addService(unit, cfg)
			if err != nil {
				return nil, err
			}
			values = append(values, serviceValues...)
		}
	}

	if needsTargetGroupBinding {
		tgbValues, err := chart.addTargetGroupBinding(unit)
		if err != nil {
			return nil, err
		}
		values = append(values, tgbValues...)
	}

	if needsServiceExport {
		err := chart.addServiceExport(unit)
		if err != nil {
			return nil, err
		}
	}
	return
}

func (chart *HelmChart) addDeployment(unit *HelmExecUnit, cfg config.ExecutionUnit) ([]HelmChartValue, error) {
	log := zap.L().Sugar().With(zap.String("unit", unit.Name))
	log.Info("Adding Deployment manifest for exec unit")
	err := addDeploymentManifest(chart, unit)
	if err != nil {
		return nil, err
	}
	values, err := unit.transformDeployment(cfg)
	if err != nil {
		return nil, err
	}
	return values, nil
}

func (chart *HelmChart) addHorizontalPodAutoscaler(unit *HelmExecUnit, cfg config.ExecutionUnit) ([]HelmChartValue, error) {
	log := zap.L().Sugar().With(zap.String("unit", unit.Name))
	log.Info("Adding HorizontalPodAutoscaler manifest for exec unit")
	err := addHorizontalPodAutoscalerManifest(chart, unit)
	if err != nil {
		return nil, err
	}
	values, err := unit.transformHorizontalPodAutoscaler(cfg)
	if err != nil {
		return nil, err
	}
	return values, nil
}

func (chart *HelmChart) addServiceAccount(unit *HelmExecUnit) ([]HelmChartValue, error) {
	log := zap.L().Sugar().With(zap.String("unit", unit.Name))
	log.Info("Adding ServiceAccount manifest for exec unit")
	err := addServiceAccountManifest(chart, unit)
	if err != nil {
		return nil, err
	}
	values, err := unit.transformServiceAccount()
	if err != nil {
		return nil, err
	}
	return values, nil
}

func (chart *HelmChart) addService(unit *HelmExecUnit, cfg config.ExecutionUnit) ([]HelmChartValue, error) {
	log := zap.L().Sugar().With(zap.String("unit", unit.Name))
	log.Info("Adding Service manifest for exec unit")
	err := addServiceManifest(chart, unit)
	if err != nil {
		return nil, err
	}

	values, err := unit.transformService(cfg)
	if err != nil {
		return nil, err
	}
	return values, nil
}

func (chart *HelmChart) addTargetGroupBinding(unit *HelmExecUnit) ([]HelmChartValue, error) {
	log := zap.L().Sugar().With(zap.String("unit", unit.Name))
	log.Info("Adding TargetGroupBinding manifest for exec unit")
	err := addTargetGroupBindingManifest(chart, unit)
	if err != nil {
		return nil, err
	}
	values, err := unit.transformTargetGroupBinding()
	if err != nil {
		return nil, err
	}
	return values, nil
}

func (chart *HelmChart) addServiceExport(unit *HelmExecUnit) error {
	log := zap.L().Sugar().With(zap.String("unit", unit.Name))
	log.Info("Adding ServiceExport manifest for exec unit")
	err := addServiceExportManifest(chart, unit)
	if err != nil {
		return err
	}
	return nil
}
