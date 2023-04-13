package kubernetes

import (
	"errors"
	"fmt"
	"regexp"

	"go.uber.org/zap"
	apps "k8s.io/api/apps/v1"
	autoscaling "k8s.io/api/autoscaling/v2beta2"
	corev1 "k8s.io/api/core/v1"
	elbv2api "sigs.k8s.io/aws-load-balancer-controller/apis/elbv2/v1beta1"
	"sigs.k8s.io/yaml"

	"github.com/klothoplatform/klotho/pkg/config"
	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/lang/dockerfile"
	"github.com/klothoplatform/klotho/pkg/logging"
)

type HelmExecUnit struct {
	Name                    string
	Namespace               string
	Service                 *core.SourceFile
	Deployment              *core.SourceFile
	Pod                     *core.SourceFile
	ServiceAccount          *core.SourceFile
	TargetGroupBinding      *core.SourceFile
	ServiceExport           *core.SourceFile
	HorizontalPodAutoscaler *core.SourceFile
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

	_, imagePlaceholder, err := unit.upsertOnlyContainer(&pod.Spec.Containers, cfg)
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
		Key:          imagePlaceholder,
	})
	return
}

func (unit *HelmExecUnit) transformDeployment(cfg config.ExecutionUnit) (values []HelmChartValue, err error) {
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

	k8sCfg, imagePlaceholder, err := unit.upsertOnlyContainer(&deployment.Spec.Template.Spec.Containers, cfg)
	if err != nil {
		return nil, err
	}
	values = append(values, HelmChartValue{
		ExecUnitName: unit.Name,
		Kind:         deployment.Kind,
		Type:         string(ImageTransformation),
		Key:          imagePlaceholder,
	})

	extraLabels := generateLabels(k8sCfg)

	if deployment.Labels == nil {
		deployment.Labels = make(map[string]string)
	}
	deployment.Labels["execUnit"] = unit.Name
	extraLabels.addTo(deployment.Labels)

	if k8sCfg.Replicas != 0 {
		*deployment.Spec.Replicas = int32(k8sCfg.Replicas)
	}

	if deployment.Spec.Template.Labels == nil {
		deployment.Spec.Template.Labels = make(map[string]string)
	}
	deployment.Spec.Template.Labels["execUnit"] = unit.Name
	deployment.Spec.Template.Spec.ServiceAccountName = unit.getServiceAccountName()
	extraLabels.addTo(deployment.Spec.Template.Labels)

	if deployment.Spec.Selector.MatchLabels == nil {
		deployment.Spec.Selector.MatchLabels = make(map[string]string)
	}
	deployment.Spec.Selector.MatchLabels["execUnit"] = unit.Name
	extraLabels.addTo(deployment.Spec.Selector.MatchLabels)

	if deployment.Spec.Template.Spec.NodeSelector == nil {
		deployment.Spec.Template.Spec.NodeSelector = make(map[string]string)
	}

	if cfg.NetworkPlacement != "" {
		deployment.Spec.Template.Spec.NodeSelector["network_placement"] = cfg.NetworkPlacement
	}
	if kconfig := cfg.GetExecutionUnitParamsAsKubernetes(); kconfig.InstanceType != "" {
		instanceTypeKey := unit.Name + "InstanceTypeKey"
		instanceTypeValue := unit.Name + "InstanceTypeValue"
		deployment.Spec.Template.Spec.NodeSelector[fmt.Sprintf("{{ .Values.%s }}", instanceTypeKey)] = fmt.Sprintf("{{ .Values.%s }}", instanceTypeValue)
		values = append(values,
			HelmChartValue{
				ExecUnitName: unit.Name,
				Kind:         deployment.Kind,
				Type:         string(InstanceTypeKey),
				Key:          instanceTypeKey,
			},
			HelmChartValue{
				ExecUnitName: unit.Name,
				Kind:         deployment.Kind,
				Type:         string(InstanceTypeValue),
				Key:          instanceTypeValue,
			},
		)
	} else if kconfig.DiskSizeGiB > 0 {
		log.Warnf("Unimplemented: disk size configured of %d ignored due to missing instance type", kconfig.DiskSizeGiB)
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

func (unit *HelmExecUnit) transformHorizontalPodAutoscaler(cfg config.ExecutionUnit) ([]HelmChartValue, error) {
	log := zap.L().Sugar().With(logging.FileField(unit.HorizontalPodAutoscaler), zap.String("unit", unit.Name))
	log.Debugf("Transforming file, %s, for exec unit, %s", unit.HorizontalPodAutoscaler.Path(), unit.Name)
	obj, err := readFile(unit.HorizontalPodAutoscaler)
	if err != nil {
		return nil, nil
	}
	hpa, ok := obj.(*autoscaling.HorizontalPodAutoscaler)
	if !ok {
		return nil, fmt.Errorf("expected file %s to contain HorizontalPodAutoscaler Kind", unit.HorizontalPodAutoscaler.Path())
	}
	k8Cfg := cfg.GetExecutionUnitParamsAsKubernetes()
	hpaCfg := k8Cfg.HorizontalPodAutoScalingConfig

	if k8Cfg.Replicas != 0 {
		minReplicas := int32(k8Cfg.Replicas)
		hpa.Spec.MinReplicas = &minReplicas
		if hpaCfg.MaxReplicas == 0 {
			hpaCfg.MaxReplicas = int(minReplicas) * 2
		}
	}

	if hpaCfg.MaxReplicas != 0 {
		maxReplicas := int32(hpaCfg.MaxReplicas)
		if maxReplicas < *hpa.Spec.MinReplicas {
			log.Errorf(`cannot set maxReplicas to %v because that's less than minReplicas (%v)`,
				hpaCfg.MaxReplicas,
				*hpa.Spec.MinReplicas,
			)
		} else {
			hpa.Spec.MaxReplicas = maxReplicas
		}
	}
	if hpaCfg.CpuUtilization != 0 {
		cpu := int32(hpaCfg.CpuUtilization)
		res := getOrCreateMetricResource(&hpa.Spec.Metrics, corev1.ResourceCPU)
		res.Target = autoscaling.MetricTarget{
			Type:               autoscaling.UtilizationMetricType,
			AverageUtilization: &cpu,
		}
	}
	if hpaCfg.MemoryUtilization != 0 {
		mem := int32(hpaCfg.MemoryUtilization)
		res := getOrCreateMetricResource(&hpa.Spec.Metrics, corev1.ResourceMemory)
		res.Target = autoscaling.MetricTarget{
			Type:               autoscaling.UtilizationMetricType,
			AverageUtilization: &mem,
		}
	}

	output, err := yaml.Marshal(hpa)
	if err != nil {
		return nil, err
	}
	manifest := string(output)
	err = unit.HorizontalPodAutoscaler.Reparse([]byte(manifest))
	if err != nil {
		return nil, err
	}
	return nil, nil
}

func getOrCreateMetricResource(metrics *[]autoscaling.MetricSpec, name corev1.ResourceName) *autoscaling.ResourceMetricSource {
	for _, spec := range *metrics {
		if spec.Type != autoscaling.ResourceMetricSourceType || spec.Resource == nil {
			continue
		}
		if spec.Resource.Name == name {
			return spec.Resource
		}
	}
	// none was there, so create one. The caller will set the target
	createdRes := &autoscaling.ResourceMetricSource{Name: name}
	createdSpec := autoscaling.MetricSpec{
		Type:     autoscaling.ResourceMetricSourceType,
		Resource: createdRes,
	}
	//created := autoscaling.ResourceMetricSource{Name: name}

	*metrics = append(*metrics, createdSpec)
	return createdRes
}

func (unit *HelmExecUnit) transformService(cfg config.ExecutionUnit) (values []HelmChartValue, err error) {
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

	k8Cfg := cfg.GetExecutionUnitParamsAsKubernetes()
	extraLabels := generateLabels(k8Cfg)

	if service.Spec.Selector == nil {
		service.Spec.Selector = make(map[string]string)
	}
	service.Spec.Selector["execUnit"] = unit.Name
	extraLabels.addTo(service.Spec.Selector)

	if service.Labels == nil {
		service.Labels = make(map[string]string)
	}
	service.Labels["execUnit"] = unit.Name
	extraLabels.addTo(service.Labels)

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
// configured. Along the way, it will also generate the image placeholder value, which it then returns. Finally, it
// also returns the k8s-specific configs it had to generate along the way.
//
// If the provided containers slice is empty, this method will create a new container and inert it into the slice; this
// modifies the call site's slice (which is why we pass in a pointer to a slice). Otherwise, we'll use the slice's
// existing container, or return an error if there is more than one.
//
// To configure the container, we:
//  1. set its image to a template for the generated placeholder value
//  2. call configureContainer on it
func (unit *HelmExecUnit) upsertOnlyContainer(containers *[]corev1.Container, cfg config.ExecutionUnit) (config.KubernetesTypeParams, string, error) {
	if len(*containers) > 1 {
		var zero config.KubernetesTypeParams
		return zero, "", errors.New("too many containers in pod spec, don't know which to replace")
	}
	if len(*containers) == 0 {
		*containers = append(*containers, corev1.Container{
			Name: unit.Name,
		})
	}
	container := &(*containers)[0]

	value := GenerateImagePlaceholder(unit.Name)
	container.Image = fmt.Sprintf("{{ .Values.%s }}", value)

	k8config, err := unit.configureContainer(container, cfg)
	if err != nil {
		return k8config, "", err
	}

	return k8config, value, nil
}

func (unit *HelmExecUnit) configureContainer(container *corev1.Container, cfg config.ExecutionUnit) (config.KubernetesTypeParams, error) {
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
		return k8sCfg, err
	}
	resourceReqs := corev1.ResourceRequirements{}
	if err = yaml.Unmarshal(limitsYaml, &resourceReqs); err != nil {
		return k8sCfg, err
	}
	for name, quantity := range resourceReqs.Limits {
		// We infer both limits and requestes from the k8sCfg limits. In order to get full utilization without overloading
		// the nodes, for now we're hard-coding the requests as being the same as limits.
		if container.Resources.Limits == nil {
			container.Resources.Limits = make(corev1.ResourceList)
		}
		if container.Resources.Requests == nil {
			container.Resources.Requests = make(corev1.ResourceList)
		}
		container.Resources.Limits[name] = quantity
		container.Resources.Requests[name] = quantity
	}

	return k8sCfg, nil
}

type kubernetesLabels map[string]string

func (k kubernetesLabels) addTo(other map[string]string) {
	for k, v := range k {
		_, inOther := other[k]
		if !inOther {
			other[k] = v
		}
	}
}

func generateLabels(cfg config.KubernetesTypeParams) kubernetesLabels {
	return map[string]string{
		"klotho-fargate-enabled": fmt.Sprintf(`%v`, cfg.NodeType == "fargate"),
	}
}
