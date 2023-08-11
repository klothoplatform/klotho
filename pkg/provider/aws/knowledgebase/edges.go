package knowledgebase

import (
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base"
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
		SqsKB,
		SnsKB,
	}
	return knowledgebase.MergeKBs(kbsToUse)
}

var AwsExtraEdgesKB = knowledgebase.Build()
