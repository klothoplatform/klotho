package resources

import (
	"errors"
	"fmt"

	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/provider"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type (
	Pod struct {
		Name            string
		ConstructRefs   core.BaseConstructSet
		Object          *corev1.Pod
		Transformations map[string]core.IaCValue
		FilePath        string
		Cluster         core.IaCValue
	}
)

const (
	POD_TYPE = "pod"
)

func (pod *Pod) BaseConstructsRef() core.BaseConstructSet {
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

func (pod *Pod) GetFunctionality() core.Functionality {
	return core.Compute
}

func (pod *Pod) GetObject() runtime.Object {
	return pod.Object
}

func (pod *Pod) Kind() string {
	return pod.Object.Kind
}

func (pod *Pod) Path() string {
	return pod.FilePath
}

func (pod *Pod) GetServiceAccount(dag *core.ResourceGraph) *ServiceAccount {
	sa := &ServiceAccount{
		Name: pod.Object.Spec.ServiceAccountName,
	}
	graphSa := dag.GetResource(sa.Id())
	if graphSa == nil {
		return nil
	}
	return graphSa.(*ServiceAccount)
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
