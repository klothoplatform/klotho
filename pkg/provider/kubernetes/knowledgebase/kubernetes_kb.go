package knowledgebase

import (
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base"
	"github.com/klothoplatform/klotho/pkg/provider/kubernetes/resources"
)

var KubernetesKB = knowledgebase.Build(
	knowledgebase.EdgeBuilder[*resources.Service, *resources.Deployment]{},
	knowledgebase.EdgeBuilder[*resources.Deployment, *resources.Pod]{},
	knowledgebase.EdgeBuilder[*resources.Service, *resources.Pod]{},
	knowledgebase.EdgeBuilder[*resources.Pod, *resources.Namespace]{},
	knowledgebase.EdgeBuilder[*resources.Service, *resources.Namespace]{},
	knowledgebase.EdgeBuilder[*resources.Deployment, *resources.Namespace]{},
	knowledgebase.EdgeBuilder[*resources.ServiceAccount, *resources.Namespace]{},
	knowledgebase.EdgeBuilder[*resources.Pod, *resources.ServiceAccount]{},
	knowledgebase.EdgeBuilder[*resources.Deployment, *resources.ServiceAccount]{},
	knowledgebase.EdgeBuilder[*resources.TargetGroupBinding, *resources.Service]{},
	knowledgebase.EdgeBuilder[*resources.ServiceExport, *resources.Service]{},
	knowledgebase.EdgeBuilder[*resources.Pod, *resources.HorizontalPodAutoscaler]{},
	knowledgebase.EdgeBuilder[*resources.Deployment, *resources.HorizontalPodAutoscaler]{},
	knowledgebase.EdgeBuilder[*resources.Pod, *resources.Pod]{},
	knowledgebase.EdgeBuilder[*resources.Deployment, *resources.Deployment]{},
)
