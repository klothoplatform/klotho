package knowledgebase

import (
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base"
)

func GetAwsKnowledgeBase() (knowledgebase.EdgeKB, error) {
	kbsToUse := []knowledgebase.EdgeKB{
		ApiGatewayKB,
		CloudfrontKB,
		EcsKB,
		ElasticacheKB,
		IamKB,
		LambdaKB,
		Ec2KB,
		EksKB,
	}
	return knowledgebase.MergeKBs(kbsToUse)
}
