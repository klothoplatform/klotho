package knowledgebase

import (
	"fmt"

	"github.com/klothoplatform/klotho/pkg/core"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base"
	"github.com/klothoplatform/klotho/pkg/provider/kubernetes/resources"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var KubernetesKB = knowledgebase.Build(
	knowledgebase.EdgeBuilder[*resources.Service, *resources.Deployment]{
		Configure: func(source *resources.Service, destination *resources.Deployment, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			return nil
		},
	},
	knowledgebase.EdgeBuilder[*resources.Service, *resources.Pod]{
		Configure: func(service *resources.Service, pod *resources.Pod, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			if service.Object == nil {
				return fmt.Errorf("service %s has no object", service.Name)
			}
			if pod.Object == nil {
				return fmt.Errorf("pod %s has no object", pod.Name)
			}
			service.Object.Spec.Selector = resources.KlothoIdSelector(pod.Object)
			return nil
		},
	},
	knowledgebase.EdgeBuilder[*resources.Pod, *resources.Namespace]{
		Configure: func(pod *resources.Pod, namespace *resources.Namespace, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			if pod.Object == nil {
				return fmt.Errorf("pod %s has no object", pod.Name)
			}
			return SetNamespace(pod.Object, namespace)
		},
	},
	knowledgebase.EdgeBuilder[*resources.Service, *resources.Namespace]{
		Configure: func(service *resources.Service, namespace *resources.Namespace, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			if service.Object == nil {
				return fmt.Errorf("service %s has no object", service.Name)
			}
			return SetNamespace(service.Object, namespace)
		},
	},
	knowledgebase.EdgeBuilder[*resources.Deployment, *resources.Namespace]{
		Configure: func(deployment *resources.Deployment, namespace *resources.Namespace, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			if deployment.Object == nil {
				return fmt.Errorf("deployment %s has no object", deployment.Name)
			}
			return SetNamespace(deployment.Object, namespace)
		},
	},
	knowledgebase.EdgeBuilder[*resources.ServiceAccount, *resources.Namespace]{
		Configure: func(serviceAccount *resources.ServiceAccount, namespace *resources.Namespace, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			if serviceAccount.Object == nil {
				return fmt.Errorf("service account %s has no object", serviceAccount.Name)
			}
			return SetNamespace(serviceAccount.Object, namespace)
		},
	},
	knowledgebase.EdgeBuilder[*resources.Pod, *resources.ServiceAccount]{
		Configure: func(pod *resources.Pod, serviceAccount *resources.ServiceAccount, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			if pod.Object == nil {
				return fmt.Errorf("pod %s has no object", pod.Name)
			}
			if serviceAccount.Object == nil {
				return fmt.Errorf("service account %s has no object", serviceAccount.Name)
			}
			pod.Object.Spec.ServiceAccountName = serviceAccount.Object.GetName()
			return nil
		},
	},
	knowledgebase.EdgeBuilder[*resources.Deployment, *resources.ServiceAccount]{
		Configure: func(deployment *resources.Deployment, serviceAccount *resources.ServiceAccount, dag *core.ResourceGraph, data knowledgebase.EdgeData) error {
			if deployment.Object == nil {
				return fmt.Errorf("deployment %s has no object", deployment.Name)
			}
			if serviceAccount.Object == nil {
				return fmt.Errorf("service account %s has no object", serviceAccount.Name)
			}
			deployment.Object.Spec.Template.Spec.ServiceAccountName = serviceAccount.Object.GetName()
			return nil
		},
	},
	knowledgebase.EdgeBuilder[*resources.TargetGroupBinding, *resources.Service]{},
	knowledgebase.EdgeBuilder[*resources.ServiceExport, *resources.Service]{},
	knowledgebase.EdgeBuilder[*resources.Pod, *resources.HorizontalPodAutoscaler]{},
	knowledgebase.EdgeBuilder[*resources.Deployment, *resources.HorizontalPodAutoscaler]{},
	knowledgebase.EdgeBuilder[*resources.Pod, *resources.KustomizeDirectory]{},
	knowledgebase.EdgeBuilder[*resources.Deployment, *resources.KustomizeDirectory]{},
	knowledgebase.EdgeBuilder[*resources.ServiceExport, *resources.KustomizeDirectory]{},
	knowledgebase.EdgeBuilder[*resources.Pod, *resources.Manifest]{},
	knowledgebase.EdgeBuilder[*resources.Deployment, *resources.Manifest]{},
)

func SetNamespace(object v1.Object, namespace *resources.Namespace) error {
	if object == nil {
		return fmt.Errorf("object has is nil")
	}
	if namespace.Object == nil {
		return fmt.Errorf("namespace %s has no object", namespace.Name)
	}
	object.SetNamespace(namespace.Object.GetName())
	return nil
}
