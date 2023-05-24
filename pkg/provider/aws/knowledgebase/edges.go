package knowledgebase

import (
	"errors"
	"fmt"

	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base"
	"github.com/klothoplatform/klotho/pkg/provider/aws/resources"
)

func GetAwsKnowledgeBase() (knowledgebase.EdgeKB, error) {
	var err error
	kbsToUse := []knowledgebase.EdgeKB{AwsExtraEdgesKB, IamKB, NetworkingKB, RdsKB, LambdaKB, ApiGatewayKB, EksKB}
	awsKB := make(knowledgebase.EdgeKB)
	for _, kb := range kbsToUse {
		for edge, detail := range kb {
			if _, found := awsKB[edge]; found {
				err = errors.Join(err, fmt.Errorf("edge for %s -> %s is already defined in the aws knowledge base", edge.Source, edge.Destination))
			}
			awsKB[edge] = detail
		}
	}
	return awsKB, err
}

var AwsExtraEdgesKB = knowledgebase.Build(
	knowledgebase.EdgeBuilder[*resources.SecretVersion, *resources.Secret]{},
	knowledgebase.EdgeBuilder[*resources.EcrImage, *resources.EcrRepository]{},
	knowledgebase.EdgeBuilder[*resources.OpenIdConnectProvider, *resources.Region]{},
)
