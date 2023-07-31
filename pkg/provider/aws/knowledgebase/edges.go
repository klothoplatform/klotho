package knowledgebase

import (
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base"
	"github.com/klothoplatform/klotho/pkg/provider/aws/resources"
	kubernetes "github.com/klothoplatform/klotho/pkg/provider/kubernetes/resources"
)

func GetAwsKnowledgeBase() (knowledgebase.EdgeKB, error) {
	kbsToUse := []knowledgebase.EdgeKB{
		ApiGatewayKB,
		AwsExtraEdgesKB,
		CloudfrontKB,
		EcsKB,
		EfsKB,
		ElasticacheKB,
		IamKB,
		LambdaKB,
		LbKB,
		NetworkingKB,
		RdsKB,
		S3KB,
		Ec2KB,
		EksKB,
		SqsKB,
		SnsKB,
		AppRunnerKB,
	}
	return knowledgebase.MergeKBs(kbsToUse)
}

var AwsExtraEdgesKB = knowledgebase.Build(
	knowledgebase.EdgeBuilder[*resources.SecretVersion, *resources.Secret]{
		DeletetionDependent: true,
	},
	knowledgebase.EdgeBuilder[*resources.EcrImage, *resources.EcrRepository]{},
	knowledgebase.EdgeBuilder[*resources.OpenIdConnectProvider, *resources.Region]{},
	knowledgebase.EdgeBuilder[*resources.PrivateDnsNamespace, *resources.Vpc]{},
	knowledgebase.EdgeBuilder[*kubernetes.Pod, *resources.PrivateDnsNamespace]{},
	knowledgebase.EdgeBuilder[*kubernetes.Deployment, *resources.PrivateDnsNamespace]{},
	knowledgebase.EdgeBuilder[*resources.PrivateDnsNamespace, *resources.Vpc]{},
	knowledgebase.EdgeBuilder[*resources.Route53HostedZone, *resources.Vpc]{},
)
