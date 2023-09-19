package knowledgebase

import (
	"fmt"
	"strings"

	"github.com/klothoplatform/klotho/pkg/construct"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base"
	"github.com/klothoplatform/klotho/pkg/provider/aws/resources"
)

var ApiGatewayKB = knowledgebase.Build(
	knowledgebase.EdgeBuilder[*resources.ApiIntegration, *resources.LoadBalancer]{
		Reuse: knowledgebase.ReuseDownstream,
		Configure: func(integration *resources.ApiIntegration, loadBalancer *resources.LoadBalancer, dag *construct.ResourceGraph, data knowledgebase.EdgeData) error {
			if integration.Method == nil {
				return fmt.Errorf("cannot configure integration %s, missing rest api or method", integration.Id())
			}
			integration.IntegrationHttpMethod = strings.ToUpper(integration.Method.HttpMethod)
			return nil
		},
	},
)
