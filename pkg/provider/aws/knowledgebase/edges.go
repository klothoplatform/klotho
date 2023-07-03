package knowledgebase

import (
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base"
	"github.com/klothoplatform/klotho/pkg/provider/aws/resources"
)

func GetAwsKnowledgeBase() (knowledgebase.EdgeKB, error) {
	kbsToUse := []knowledgebase.EdgeKB{
		ApiGatewayKB,
		AwsExtraEdgesKB,
		CloudfrontKB,
		EcsKB,
		ElasticacheKB,
		IamKB,
		LambdaKB,
		LbKB,
		NetworkingKB,
		RdsKB,
		S3KB,
		Ec2KB,
		EksKB,
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
	knowledgebase.EdgeBuilder[*resources.Route53HostedZone, *resources.Vpc]{},
)
