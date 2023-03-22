package kubernetes

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/klothoplatform/klotho/pkg/annotation"
	"github.com/klothoplatform/klotho/pkg/config"
	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/logging"
	"go.uber.org/zap"
	apps "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

type KlothoHelmChart struct {
	Name           string
	ValuesFiles    []string
	ExecutionUnits []*HelmExecUnit
	Directory      string
	Files          []core.File
	Values         []Value
	AnnotationKeys []core.AnnotationKey
}

// Provider returns name of the provider the resource is correlated to
func (chart *KlothoHelmChart) Provider() string { return "kubernetes" }

// KlothoResource returns AnnotationKey of the klotho resource the cloud resource is correlated to
func (chart *KlothoHelmChart) KlothoConstructRef() []core.AnnotationKey { return chart.AnnotationKeys }

// ID returns the id of the cloud resource
func (chart *KlothoHelmChart) Id() string { return fmt.Sprintf("klotho_helm_chart-%s", chart.Name) }

var HelmChartKind = "helm_chart"

func (t *KlothoHelmChart) OutputTo(dest string) error {
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

func (t *KlothoHelmChart) AssignFilesToUnits() error {
	needsMetadataName := len(t.ExecutionUnits) > 1
	for _, unit := range t.ExecutionUnits {
		for _, f := range t.Files {
			ast, ok := f.(*core.SourceFile)
			if !ok {
				continue
			}
			log := zap.L().Sugar().With(logging.FileField(f), zap.String("unit", unit.Name))

			obj, err := readFile(ast)
			if err != nil {
				return err
			}
			// now use switch over the type of the object
			// and match each type-case
			switch o := obj.(type) {
			case *corev1.Pod:
				pod := o
				if needsMetadataName {
					if pod.Name == unit.Name {
						log.Debugf("Found unit, %s's, pod manifest in file, %s", unit.Name, f.Path())
						if unit.Deployment != nil {
							return fmt.Errorf("can not support multiple pod specifications for unit %s", unit.Name)
						}
						unit.Pod = ast
					}
				} else {
					log.Debug("Found unit, %s's, pod manifest in file, %s", unit.Name, f.Path())
					if unit.Deployment != nil {
						return fmt.Errorf("can not support multiple pod specifications for unit %s", unit.Name)
					}
					unit.Pod = ast
				}
			case *apps.Deployment:
				deployment := o
				if needsMetadataName {
					if deployment.Name == unit.Name {
						log.Debugf("Found unit, %s's, pod manifest in file, %s", unit.Name, f.Path())
						if unit.Pod != nil {
							return fmt.Errorf("can not support multiple pod specifications for unit %s", unit.Name)
						}
						unit.Deployment = ast
					}
				} else {
					log.Debugf("Found unit, %s's, pod manifest in file, %s", unit.Name, f.Path())
					if unit.Pod != nil {
						return fmt.Errorf("can not support multiple pod specifications for unit %s", unit.Name)
					}
					unit.Deployment = ast
				}
			case *corev1.ServiceAccount:
				serviceAccount := o
				if needsMetadataName {
					if serviceAccount.Name == unit.Name {
						log.Debugf("Found unit, %s's, pod manifest in file, %s", unit.Name, f.Path())
						unit.ServiceAccount = ast
					}
				} else {
					log.Debugf("Found unit, %s's, pod manifest in file, %s", unit.Name, f.Path())
					unit.ServiceAccount = ast
				}
			case *corev1.Service:
				service := o
				if needsMetadataName {
					if service.Name == unit.Name {
						log.Debugf("Found unit, %s's, pod manifest in file, %s", unit.Name, f.Path())
						unit.Service = ast
					}
				} else {
					log.Debugf("Found unit, %s's, pod manifest in file, %s", unit.Name, f.Path())
					unit.Service = ast
				}
			default:
				log.Debug("Unrecognized type")
			}
		}
	}
	return nil
}

func (chart *KlothoHelmChart) handleExecutionUnit(unit *HelmExecUnit, eu *core.ExecutionUnit, cfg config.ExecutionUnit, constructGraph *core.ConstructGraph) ([]Value, error) {
	values := []Value{}

	if shouldTransformImage(eu) {
		if unit.Deployment != nil {
			deploymentValues, err := unit.transformDeployment()
			if err != nil {
				return nil, err
			}
			values = append(values, deploymentValues...)
		} else if unit.Pod != nil {
			podValues, err := unit.transformPod()
			if err != nil {
				return nil, err
			}
			values = append(values, podValues...)
		} else {
			deploymentValues, err := chart.addDeployment(unit)
			if err != nil {
				return nil, err
			}
			values = append(values, deploymentValues...)
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
	upstreamValues, err := chart.handleUpstreamUnitDependencies(unit, constructGraph)
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

func (chart *KlothoHelmChart) handleUpstreamUnitDependencies(unit *HelmExecUnit, constructGraph *core.ConstructGraph) (values []Value, err error) {
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
			serviceValues, err := unit.transformService()
			if err != nil {
				return nil, err
			}
			values = append(values, serviceValues...)
		} else {
			serviceValues, err := chart.addService(unit)
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

func (chart *KlothoHelmChart) addDeployment(unit *HelmExecUnit) ([]Value, error) {
	log := zap.L().Sugar().With(zap.String("unit", unit.Name))
	log.Info("Adding Deployment manifest for exec unit")
	err := addDeploymentManifest(chart, unit)
	if err != nil {
		return nil, err
	}
	values, err := unit.transformDeployment()
	if err != nil {
		return nil, err
	}
	return values, nil
}

func (chart *KlothoHelmChart) addServiceAccount(unit *HelmExecUnit) ([]Value, error) {
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

func (chart *KlothoHelmChart) addService(unit *HelmExecUnit) ([]Value, error) {
	log := zap.L().Sugar().With(zap.String("unit", unit.Name))
	log.Info("Adding Service manifest for exec unit")
	err := addServiceManifest(chart, unit)
	if err != nil {
		return nil, err
	}

	values, err := unit.transformService()
	if err != nil {
		return nil, err
	}
	return values, nil
}

func (chart *KlothoHelmChart) addTargetGroupBinding(unit *HelmExecUnit) ([]Value, error) {
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

func (chart *KlothoHelmChart) addServiceExport(unit *HelmExecUnit) error {
	log := zap.L().Sugar().With(zap.String("unit", unit.Name))
	log.Info("Adding ServiceExport manifest for exec unit")
	err := addServiceExportManifest(chart, unit)
	if err != nil {
		return err
	}
	return nil
}
