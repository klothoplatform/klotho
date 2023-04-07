package kubernetes

import (
	"errors"
	"fmt"
	"github.com/klothoplatform/klotho/pkg/config"
	"regexp"

	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/lang/dockerfile"
	"github.com/klothoplatform/klotho/pkg/logging"
	"go.uber.org/zap"
	apps "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	elbv2api "sigs.k8s.io/aws-load-balancer-controller/apis/elbv2/v1beta1"
	"sigs.k8s.io/yaml"
)

type HelmExecUnit struct {
	Name               string
	Namespace          string
	Service            *core.SourceFile
	Deployment         *core.SourceFile
	Pod                *core.SourceFile
	ServiceAccount     *core.SourceFile
	TargetGroupBinding *core.SourceFile
	ServiceExport      *core.SourceFile
}

var (
	EKS_ANNOTATION_KEY = "eks.amazonaws.com/role-arn"
)
var nonAlphanumericRegex = regexp.MustCompile(`[^a-zA-Z0-9 ]+`)

func sanitizeString(str string) string {
	return nonAlphanumericRegex.ReplaceAllString(str, "")
}

func GenerateRoleArnPlaceholder(unit string) string {
	return fmt.Sprintf("%sRoleArn", sanitizeString(unit))
}

func GenerateImagePlaceholder(unit string) string {
	return fmt.Sprintf("%sImage", sanitizeString(unit))
}

func GenerateTargetGroupBindingPlaceholder(unit string) string {
	return fmt.Sprintf("%sTargetGroupArn", sanitizeString(unit))
}

func GenerateEnvVarKeyValue(key string) (k string, v string) {
	k = key
	v = sanitizeString(key)
	return
}

func shouldTransformImage(unit *core.ExecutionUnit) bool {
	for _, f := range unit.Files() {
		ast, ok := f.(*core.SourceFile)
		if !ok {
			continue
		}
		if _, ok := dockerfile.DockerfileLang.CastFile(ast); ok {
			return true
		}
	}
	return false
}

func shouldTransformServiceAccount(unit *core.ExecutionUnit) bool {
	// TODO: Replace this with logic that determines if we are creating a role for the exec unit. This happens after here (the aws provider) today.
	// Ideally we should understand if we are parsing any app code and if not (only building a Dockerfile) then the permissions which we assign won't matter
	return shouldTransformImage(unit)
}

func (unit *HelmExecUnit) transformPod(cfg config.ExecutionUnit) (values []HelmChartValue, err error) {
	log := zap.L().Sugar().With(logging.FileField(unit.Pod), zap.String("unit", unit.Name))
	log.Debugf("Transforming file, %s, for exec unit, %s", unit.Pod.Path(), unit.Name)
	obj, err := readFile(unit.Pod)
	if err != nil {
		return
	}
	pod, ok := obj.(*corev1.Pod)
	if !ok {
		err = fmt.Errorf("expected file %s to contain Pod Kind", unit.Pod.Path())
		return
	}

	value, err := unit.upsertOnlyContainer(&pod.Spec.Containers, cfg)
	if err != nil {
		return nil, err
	}

	if pod.Labels == nil {
		pod.Labels = make(map[string]string)
	}
	pod.Labels["execUnit"] = unit.Name
	pod.Spec.ServiceAccountName = unit.getServiceAccountName()

	output, err := yaml.Marshal(pod)
	if err != nil {
		return
	}
	err = unit.Pod.Reparse([]byte(output))
	if err != nil {
		return
	}
	values = append(values, HelmChartValue{
		ExecUnitName: unit.Name,
		Kind:         pod.Kind,
		Type:         string(ImageTransformation),
		Key:          value,
	})
	return
}

func (unit *HelmExecUnit) transformDeployment(cfg config.ExecutionUnit) ([]HelmChartValue, error) {
	values := []HelmChartValue{}
	log := zap.L().Sugar().With(logging.FileField(unit.Deployment), zap.String("unit", unit.Name))
	log.Debugf("Transforming file, %s, for exec unit, %s", unit.Deployment.Path(), unit.Name)
	obj, err := readFile(unit.Deployment)
	if err != nil {
		return nil, err
	}
	deployment, ok := obj.(*apps.Deployment)
	if !ok {
		err = fmt.Errorf("expected file %s to contain Deployment Kind", unit.Deployment.Path())
		return nil, err
	}

	value, err := unit.upsertOnlyContainer(&deployment.Spec.Template.Spec.Containers, cfg)
	if err != nil {
		return nil, err
	}

	if deployment.Labels == nil {
		deployment.Labels = make(map[string]string)
	}
	deployment.Labels["execUnit"] = unit.Name

	if deployment.Spec.Template.Labels == nil {
		deployment.Spec.Template.Labels = make(map[string]string)
	}
	deployment.Spec.Template.Labels["execUnit"] = unit.Name

	if deployment.Spec.Selector.MatchLabels == nil {
		deployment.Spec.Selector.MatchLabels = make(map[string]string)
	}
	deployment.Spec.Selector.MatchLabels["execUnit"] = unit.Name

	deployment.Spec.Template.Spec.ServiceAccountName = unit.getServiceAccountName()

	output, err := yaml.Marshal(deployment)
	if err != nil {
		return nil, err
	}
	err = unit.Deployment.Reparse([]byte(output))
	if err != nil {
		return nil, err
	}
	values = append(values, HelmChartValue{
		ExecUnitName: unit.Name,
		Kind:         deployment.Kind,
		Type:         string(ImageTransformation),
		Key:          value,
	})
	return values, nil
}

func mapOrNew[K comparable, V any](input map[K]V) map[K]V {
	if input == nil {
		input = make(map[K]V)
	}
	return input
}

func (unit *HelmExecUnit) transformService() (values []HelmChartValue, err error) {
	log := zap.L().Sugar().With(logging.FileField(unit.Service), zap.String("unit", unit.Name))
	log.Debugf("Transforming file, %s, for exec unit, %s", unit.Service.Path(), unit.Name)
	obj, err := readFile(unit.Service)
	if err != nil {
		return
	}
	service, ok := obj.(*corev1.Service)
	if !ok {
		err = fmt.Errorf("expected file %s to contain ServiceAccount Kind", unit.ServiceAccount.Path())
		return
	}
	if service.Spec.Selector == nil {
		service.Spec.Selector = make(map[string]string)
	}
	service.Spec.Selector["execUnit"] = unit.Name

	if service.Labels == nil {
		service.Labels = make(map[string]string)
	}
	service.Labels["execUnit"] = unit.Name
	output, err := yaml.Marshal(service)
	if err != nil {
		return nil, err
	}
	manifest := string(output)
	err = unit.Service.Reparse([]byte(manifest))
	if err != nil {
		return nil, err
	}
	return
}

func (unit *HelmExecUnit) transformServiceAccount() (values []HelmChartValue, err error) {
	log := zap.L().Sugar().With(logging.FileField(unit.ServiceAccount), zap.String("unit", unit.Name))
	log.Debugf("Transforming file, %s, for exec unit, %s", unit.ServiceAccount.Path(), unit.Name)
	obj, err := readFile(unit.ServiceAccount)
	if err != nil {
		return
	}
	serviceAccount, ok := obj.(*corev1.ServiceAccount)
	if !ok {
		err = fmt.Errorf("expected file %s to contain ServiceAccount Kind", unit.ServiceAccount.Path())
		return
	}
	value := GenerateRoleArnPlaceholder(unit.Name)
	if serviceAccount.Annotations == nil {
		serviceAccount.Annotations = make(map[string]string)
	}
	serviceAccount.Annotations[EKS_ANNOTATION_KEY] = fmt.Sprintf("{{ .Values.%s }}", value)
	if serviceAccount.Labels == nil {
		serviceAccount.Labels = make(map[string]string)
	}
	serviceAccount.Labels["execUnit"] = unit.Name

	output, err := yaml.Marshal(serviceAccount)
	if err != nil {
		return nil, err
	}
	err = unit.ServiceAccount.Reparse([]byte(output))
	if err != nil {
		return nil, err
	}
	values = append(values, HelmChartValue{
		ExecUnitName: unit.Name,
		Kind:         serviceAccount.Kind,
		Type:         string(ServiceAccountAnnotationTransformation),
		Key:          value,
	})
	return
}

func (unit *HelmExecUnit) transformTargetGroupBinding() (values []HelmChartValue, err error) {
	log := zap.L().Sugar().With(logging.FileField(unit.TargetGroupBinding), zap.String("unit", unit.Name))
	log.Debugf("Transforming file, %s, for exec unit, %s", unit.TargetGroupBinding.Path(), unit.Name)
	obj, err := readElbv2ApiFiles(unit.TargetGroupBinding)
	if err != nil {
		return
	}
	targetGroupBinding, ok := obj.(*elbv2api.TargetGroupBinding)
	if !ok {
		err = fmt.Errorf("expected file %s to contain TargetGroupBinding Kind", unit.TargetGroupBinding.Path())
		return
	}
	value := GenerateTargetGroupBindingPlaceholder(unit.Name)

	targetGroupBinding.Spec.TargetGroupARN = fmt.Sprintf("{{ .Values.%s }}", value)

	if targetGroupBinding.Labels == nil {
		targetGroupBinding.Labels = make(map[string]string)
	}
	targetGroupBinding.Labels["execUnit"] = unit.Name
	output, err := yaml.Marshal(targetGroupBinding)
	if err != nil {
		return nil, err
	}
	err = unit.TargetGroupBinding.Reparse([]byte(output))
	if err != nil {
		return nil, err
	}
	values = append(values, HelmChartValue{
		ExecUnitName: unit.Name,
		Kind:         targetGroupBinding.Kind,
		Type:         string(TargetGroupTransformation),
		Key:          value,
	})
	return
}

func (unit *HelmExecUnit) getServiceAccountName() string {
	if unit.ServiceAccount == nil {
		return unit.Name
	}
	obj, err := readFile(unit.ServiceAccount)
	if err != nil {
		return unit.Name
	}
	serviceAccount, ok := obj.(*corev1.ServiceAccount)
	if !ok {
		zap.S().Debugf("expected file %s to contain ServiceAccount Kind", unit.ServiceAccount.Path())
		return unit.Name
	}
	return serviceAccount.Name
}

func (unit *HelmExecUnit) getServiceName() string {
	if unit.Service == nil {
		return unit.Name
	}
	obj, err := readFile(unit.Service)
	if err != nil {
		return unit.Name
	}
	service, ok := obj.(*corev1.Service)
	if !ok {
		zap.S().Debugf("expected file %s to contain Service Kind", unit.Service.Path())
		return unit.Name
	}
	if service.Name != "" {
		return service.Name
	}
	return unit.Name
}

func (unit *HelmExecUnit) AddUnitsEnvironmentVariables(eu *core.ExecutionUnit) (values []HelmChartValue, err error) {
	if unit.Deployment != nil {
		v, err := unit.addEnvsVarToDeployment(eu.EnvironmentVariables)
		if err != nil {
			return nil, err
		}

		values = append(values, v...)
	} else if unit.Pod != nil {
		v, err := unit.addEnvVarToPod(eu.EnvironmentVariables)
		if err != nil {
			return nil, err
		}

		values = append(values, v...)
	}
	return
}
func (unit *HelmExecUnit) addEnvsVarToDeployment(envVars core.EnvironmentVariables) ([]HelmChartValue, error) {
	values := []HelmChartValue{}

	log := zap.L().Sugar().With(logging.FileField(unit.Deployment), zap.String("unit", unit.Name))
	log.Debugf("Adding environment variables to file, %s, for exec unit, %s", unit.Deployment.Path(), unit.Name)
	obj, err := readFile(unit.Deployment)
	if err != nil {
		return nil, err
	}
	deployment, ok := obj.(*apps.Deployment)
	if !ok {
		err = fmt.Errorf("expected file %s to contain Deployment Kind", unit.Deployment.Path())
		return nil, err
	}

	if len(deployment.Spec.Template.Spec.Containers) != 1 {
		return nil, errors.New("expected one container in deployment spec, cannot add environment variable")
	} else {
		for _, envVar := range envVars {

			k, v := GenerateEnvVarKeyValue(envVar.Name)

			newEv := corev1.EnvVar{
				Name:  k,
				Value: fmt.Sprintf("{{ .Values.%s }}", v),
			}

			deployment.Spec.Template.Spec.Containers[0].Env = append(deployment.Spec.Template.Spec.Containers[0].Env, newEv)
			values = append(values, HelmChartValue{
				ExecUnitName:        unit.Name,
				Kind:                deployment.Kind,
				Type:                string(EnvironmentVariableTransformation),
				Key:                 v,
				EnvironmentVariable: envVar,
			})
		}
	}

	output, err := yaml.Marshal(deployment)
	if err != nil {
		return nil, err
	}
	err = unit.Deployment.Reparse([]byte(output))
	if err != nil {
		return nil, err
	}

	return values, nil
}

func (unit *HelmExecUnit) addEnvVarToPod(envVars core.EnvironmentVariables) ([]HelmChartValue, error) {
	values := []HelmChartValue{}

	log := zap.L().Sugar().With(logging.FileField(unit.Pod), zap.String("unit", unit.Name))
	log.Debugf("Adding environment variables to file, %s, for exec unit, %s", unit.Pod.Path(), unit.Name)
	obj, err := readFile(unit.Pod)
	if err != nil {
		return nil, err
	}
	pod, ok := obj.(*corev1.Pod)
	if !ok {
		err = fmt.Errorf("expected file %s to contain Pod Kind", unit.Pod.Path())
		return nil, err
	}

	if len(pod.Spec.Containers) != 1 {
		return nil, errors.New("expected one container in Pod spec, cannot add environment variable")
	} else {
		for _, envVar := range envVars {

			k, v := GenerateEnvVarKeyValue(envVar.Name)

			newEv := corev1.EnvVar{
				Name:  k,
				Value: fmt.Sprintf("{{ .Values.%s }}", v),
			}

			pod.Spec.Containers[0].Env = append(pod.Spec.Containers[0].Env, newEv)
			values = append(values, HelmChartValue{
				ExecUnitName:        unit.Name,
				Kind:                pod.Kind,
				Type:                string(EnvironmentVariableTransformation),
				Key:                 v,
				EnvironmentVariable: envVar,
			})
		}
	}

	output, err := yaml.Marshal(pod)
	if err != nil {
		return nil, err
	}
	err = unit.Pod.Reparse([]byte(output))
	if err != nil {
		return nil, err
	}

	return values, nil
}

// upsertOnlyContainer ensures that there is exactly one container in the given slice, and that it's correctly
// configured. Along the way, it will also generate the image placeholder value, which it then returns.
//
// If the provided containers slice is empty, this method will create a new container and inert it into the slice; this
// modifies the call site's slice (which is why we pass in a pointer to a slice). Otherwise, we'll use the slice's
// existing container, or return an error if there is more than one.
//
// To configure the container, we:
//  1. set its image to a template for the generated placeholder value
//  2. call configureContainer on it
func (unit *HelmExecUnit) upsertOnlyContainer(containers *[]corev1.Container, cfg config.ExecutionUnit) (string, error) {
	if len(*containers) > 1 {
		return "", errors.New("too many containers in pod spec, don't know which to replace")
	}
	if len(*containers) == 0 {
		*containers = append(*containers, corev1.Container{
			Name: unit.Name,
		})
	}
	container := &(*containers)[0]

	value := GenerateImagePlaceholder(unit.Name)
	container.Image = fmt.Sprintf("{{ .Values.%s }}", value)

	if err := unit.configureContainer(container, cfg); err != nil {
		return "", err
	}

	return value, nil
}

func (unit *HelmExecUnit) configureContainer(container *corev1.Container, cfg config.ExecutionUnit) error {
	k8sCfg := cfg.GetExecutionUnitParamsAsKubernetes()

	limits := make(map[corev1.ResourceName]any)
	if k8sCfg.Limits.Cpu != nil {
		limits[corev1.ResourceCPU] = k8sCfg.Limits.Cpu
	}
	if k8sCfg.Limits.Memory != nil {
		limits[corev1.ResourceMemory] = k8sCfg.Limits.Memory
	}
	limitsYaml, err := yaml.Marshal(map[string]any{"limits": limits})
	if err != nil {
		return err
	}
	resourceReqs := corev1.ResourceRequirements{}
	if err = yaml.Unmarshal(limitsYaml, &resourceReqs); err != nil {
		return err
	}
	for name, quantity := range resourceReqs.Limits {
		container.Resources.Limits = mapOrNew(container.Resources.Limits)
		container.Resources.Limits[name] = quantity
	}

	return nil
}
