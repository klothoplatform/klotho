package resources

import (
	"errors"
	"fmt"

	"github.com/klothoplatform/klotho/pkg/construct"
	"github.com/klothoplatform/klotho/pkg/engine/classification"
	"github.com/klothoplatform/klotho/pkg/provider"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type (
	Pod struct {
		Name          string
		ConstructRefs construct.BaseConstructSet `yaml:"-"`
		Object        *corev1.Pod
		Values        map[string]construct.IaCValue
		FilePath      string
		Cluster       construct.ResourceId
	}
)

const (
	POD_TYPE = "pod"
)

func (pod *Pod) BaseConstructRefs() construct.BaseConstructSet {
	return pod.ConstructRefs
}

func (pod *Pod) Id() construct.ResourceId {
	return construct.ResourceId{
		Provider: provider.KUBERNETES,
		Type:     POD_TYPE,
		Name:     pod.Name,
	}
}

func (pod *Pod) DeleteContext() construct.DeleteContext {
	return construct.DeleteContext{
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

func (pod *Pod) GetServiceAccount(dag *construct.ResourceGraph) *ServiceAccount {
	if pod.Object == nil {
		sas := construct.GetDownstreamResourcesOfType[*ServiceAccount](dag, pod)
		if len(sas) == 1 {
			return sas[0]
		}
		return nil
	}
	for _, sa := range construct.GetDownstreamResourcesOfType[*ServiceAccount](dag, pod) {
		if sa.Object != nil && sa.Object.Name == pod.Object.Spec.ServiceAccountName {
			return sa
		}
	}
	return nil
}
func (pod *Pod) AddEnvVar(iacVal construct.IaCValue, envVarName string) error {

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
		if pod.Values == nil {
			pod.Values = make(map[string]construct.IaCValue)
		}
		pod.Values[v] = iacVal
	}
	return nil
}

func (pod *Pod) MakeOperational(dag *construct.ResourceGraph, appName string, classifier classification.Classifier) error {
	if pod.Cluster.IsZero() {
		return fmt.Errorf("%s has no cluster", pod.Id())
	}
	if pod.Object == nil {
		pod.Object = &corev1.Pod{}
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

func (pod *Pod) GetValues() map[string]construct.IaCValue {
	return pod.Values
}
