package knowledgebase

import (
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base"
	"github.com/klothoplatform/klotho/pkg/provider/aws/resources"
	kubernetes "github.com/klothoplatform/klotho/pkg/provider/kubernetes/resources"
)

var SqsKB = knowledgebase.Build(
	knowledgebase.EdgeBuilder[*resources.LambdaFunction, *resources.SqsQueue]{},
	knowledgebase.EdgeBuilder[*resources.EcsService, *resources.SqsQueue]{},
	knowledgebase.EdgeBuilder[*resources.Ec2Instance, *resources.SqsQueue]{},
	knowledgebase.EdgeBuilder[*kubernetes.Pod, *resources.SqsQueue]{},
	knowledgebase.EdgeBuilder[*kubernetes.Deployment, *resources.SqsQueue]{},
	knowledgebase.EdgeBuilder[*resources.SqsQueue, *resources.LambdaFunction]{},
	knowledgebase.EdgeBuilder[*resources.SqsQueue, *resources.EcsService]{},
	knowledgebase.EdgeBuilder[*resources.SqsQueue, *resources.Ec2Instance]{},
	knowledgebase.EdgeBuilder[*resources.SqsQueue, *kubernetes.Pod]{},
	knowledgebase.EdgeBuilder[*resources.SqsQueue, *kubernetes.Deployment]{},
	knowledgebase.EdgeBuilder[*resources.SqsQueue, *resources.SqsQueuePolicy]{},
)
