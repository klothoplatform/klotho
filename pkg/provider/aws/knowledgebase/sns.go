package knowledgebase

import (
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base"
	"github.com/klothoplatform/klotho/pkg/provider/aws/resources"
	kubernetes "github.com/klothoplatform/klotho/pkg/provider/kubernetes/resources"
)

var SnsKB = knowledgebase.Build(
	knowledgebase.EdgeBuilder[*resources.LambdaFunction, *resources.SnsTopic]{},
	knowledgebase.EdgeBuilder[*resources.EcsService, *resources.SnsTopic]{},
	knowledgebase.EdgeBuilder[*resources.Ec2Instance, *resources.SnsTopic]{},
	knowledgebase.EdgeBuilder[*kubernetes.Pod, *resources.SnsTopic]{},
	knowledgebase.EdgeBuilder[*kubernetes.Deployment, *resources.SnsTopic]{},
	knowledgebase.EdgeBuilder[*resources.SnsSubscription, *resources.LambdaFunction]{},
	knowledgebase.EdgeBuilder[*resources.SnsSubscription, *resources.EcsService]{},
	knowledgebase.EdgeBuilder[*resources.SnsSubscription, *resources.Ec2Instance]{},
	knowledgebase.EdgeBuilder[*resources.SnsSubscription, *kubernetes.Pod]{},
	knowledgebase.EdgeBuilder[*resources.SnsSubscription, *kubernetes.Deployment]{},
	knowledgebase.EdgeBuilder[*resources.SnsTopic, *resources.SnsSubscription]{},
)
