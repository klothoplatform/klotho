package resources

import (
	"errors"
	"fmt"

	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/engine/classification"
	"github.com/klothoplatform/klotho/pkg/provider"
	"go.uber.org/zap"
	apps "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type (
	Deployment struct {
		Name            string
		ConstructRefs   core.BaseConstructSet
		Object          *apps.Deployment
		Transformations map[string]core.IaCValue
		FilePath        string
		Cluster         core.ResourceId
	}
)

const (
	DEPLOYMENT_TYPE = "deployment"
)

func (deployment *Deployment) BaseConstructRefs() core.BaseConstructSet {
	return deployment.ConstructRefs
}

func (deployment *Deployment) Id() core.ResourceId {
	return core.ResourceId{
		Provider: provider.KUBERNETES,
		Type:     DEPLOYMENT_TYPE,
		Name:     deployment.Name,
	}
}

func (deployment *Deployment) DeleteContext() core.DeleteContext {
	return core.DeleteContext{
		RequiresNoUpstream:     true,
		RequiresExplicitDelete: true,
	}
}

func (deployment *Deployment) GetObject() runtime.Object {
	return deployment.Object
}

func (deployment *Deployment) Kind() string {
	return deployment.Object.Kind
}

func (deployment *Deployment) Path() string {
	return deployment.FilePath
}

func (deployment *Deployment) MakeOperational(dag *core.ResourceGraph, appName string, classifier classification.Classifier) error {
	if deployment.Object == nil {
		deployment.Object = &apps.Deployment{}
		sa := &ServiceAccount{
			Name: deployment.Name,
		}
		deployment.Object.Spec.Template.Spec.ServiceAccountName = sa.Name
		dag.AddDependency(deployment, sa)
	}
	if deployment.Cluster.IsZero() {
		var downstreamClustersFound []core.Resource
		for _, res := range dag.GetAllDownstreamResources(deployment) {
			if classifier.GetFunctionality(res) == core.Cluster {
				downstreamClustersFound = append(downstreamClustersFound, res)
			}
		}
		if len(downstreamClustersFound) == 1 {
			dag.AddDependency(deployment, downstreamClustersFound[0])
			deployment.Cluster = downstreamClustersFound[0].Id()
			return nil
		}
		if len(downstreamClustersFound) > 1 {
			return fmt.Errorf("deployment %s has more than one cluster downstream", deployment.Id())
		}
		return core.NewOperationalResourceError(deployment, []string{string(core.Cluster)}, fmt.Errorf("deployment %s has no clusters to use", deployment.Id()))
	}
	return nil
}

func (deployment *Deployment) GetServiceAccount(dag *core.ResourceGraph) *ServiceAccount {
	if deployment.Object == nil {
		return nil
	}
	sa := &ServiceAccount{
		Name: deployment.Object.Spec.Template.Spec.ServiceAccountName,
	}
	graphSa := dag.GetResource(sa.Id())
	if graphSa == nil {
		return nil
	}
	return graphSa.(*ServiceAccount)
}

func (deployment *Deployment) AddEnvVar(iacVal core.IaCValue, envVarName string) error {

	log := zap.L().Sugar()
	log.Debugf("Adding environment variables to pod, %s", deployment.Name)

	if len(deployment.Object.Spec.Template.Spec.Containers) != 1 {
		return errors.New("expected one container in Deployment spec, cannot add environment variable")
	} else {
		k, v := GenerateEnvVarKeyValue(envVarName)

		newEv := corev1.EnvVar{
			Name:  k,
			Value: fmt.Sprintf("{{ .Values.%s }}", v),
		}

		deployment.Object.Spec.Template.Spec.Containers[0].Env = append(deployment.Object.Spec.Template.Spec.Containers[0].Env, newEv)
		if deployment.Transformations == nil {
			deployment.Transformations = make(map[string]core.IaCValue)
		}
		deployment.Transformations[v] = iacVal
	}
	return nil
}
