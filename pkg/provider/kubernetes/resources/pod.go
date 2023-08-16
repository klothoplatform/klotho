package resources

import (
	"errors"
	"fmt"
	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/engine/classification"
	"github.com/klothoplatform/klotho/pkg/provider"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type (
	Pod struct {
		Name            string
		ConstructRefs   core.BaseConstructSet `yaml:"-"`
		Object          *corev1.Pod
		Transformations map[string]core.IaCValue
		FilePath        string
		Cluster         core.ResourceId
	}
)

const (
	POD_TYPE = "pod"
)

func (pod *Pod) BaseConstructRefs() core.BaseConstructSet {
	return pod.ConstructRefs
}

func (pod *Pod) Id() core.ResourceId {
	return core.ResourceId{
		Provider: provider.KUBERNETES,
		Type:     POD_TYPE,
		Name:     pod.Name,
	}
}

func (pod *Pod) DeleteContext() core.DeleteContext {
	return core.DeleteContext{
		RequiresNoUpstream:     true,
		RequiresExplicitDelete: true,
	}
}
func (pod *Pod) GetObject() v1.Object {
	return pod.Object
}

func (pod *Pod) Kind() string {
	return pod.Object.Kind
}

func (pod *Pod) Path() string {
	return pod.FilePath
}

func (pod *Pod) GetServiceAccount(dag *core.ResourceGraph) *ServiceAccount {
	if pod.Object == nil {
		sas := core.GetDownstreamResourcesOfType[*ServiceAccount](dag, pod)
		if len(sas) == 1 {
			return sas[0]
		}
		return nil
	}
	for _, sa := range core.GetDownstreamResourcesOfType[*ServiceAccount](dag, pod) {
		if sa.Object != nil && sa.Object.Name == pod.Object.Spec.ServiceAccountName {
			return sa
		}
	}
	return nil
}
func (pod *Pod) AddEnvVar(iacVal core.IaCValue, envVarName string) error {

	log := zap.L().Sugar()
	log.Debugf("Adding environment variables to pod, %s", pod.Name)

	if len(pod.Object.Spec.Containers) != 1 {
		return errors.New("expected one container in Pod spec, cannot add environment variable")
	} else {

		k, v := GenerateEnvVarKeyValue(envVarName)

		newEv := corev1.EnvVar{
			Name:  k,
			Value: fmt.Sprintf("{{ .Values.%s }}", v),
		}

		pod.Object.Spec.Containers[0].Env = append(pod.Object.Spec.Containers[0].Env, newEv)
		if pod.Transformations == nil {
			pod.Transformations = make(map[string]core.IaCValue)
		}
		pod.Transformations[v] = iacVal
	}
	return nil
}

func (pod *Pod) MakeOperational(dag *core.ResourceGraph, appName string, classifier classification.Classifier) error {
	if pod.Cluster.IsZero() {
		return fmt.Errorf("%s has no cluster", pod.Id())
	}
	SetDefaultObjectMeta(pod, pod.Object.GetObjectMeta())
	pod.FilePath = ManifestFilePath(pod)

	// TODO: consider changing this once ports are properly configurable
	// Map default port for containers if none are specified
	for i, container := range pod.Object.Spec.Containers {
		containerP := &container
		if len(containerP.Ports) == 0 {
			containerP.Ports = append(containerP.Ports, corev1.ContainerPort{
				Name:          "default-tcp",
				ContainerPort: 3000,
				HostPort:      3000 + int32(i),
				Protocol:      corev1.ProtocolTCP,
			})
		}
		pod.Object.Spec.Containers[i] = *containerP
	}
	return nil
}

func (pod *Pod) GetValues() map[string]core.IaCValue {
	return pod.Transformations
}
