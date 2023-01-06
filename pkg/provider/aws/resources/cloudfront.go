package resources

import (
	"github.com/klothoplatform/klotho/pkg/core"
	"go.uber.org/zap"
)

type CloudfrontDistribution struct {
	Id                string
	Origins           []core.ResourceKey
	DefaultRootObject string
}

func CreateCloudfrontDistribution(resources []core.CloudResource) *CloudfrontDistribution {
	distribution := &CloudfrontDistribution{}

	for _, res := range resources {
		switch res.Key().Kind {
		case core.GatewayKind:
			distribution.Origins = append(distribution.Origins, res.Key())
		case core.StaticUnitKind:
			sunit := res.(*core.StaticUnit)
			distribution.Origins = append(distribution.Origins, res.Key())
			if distribution.DefaultRootObject != "" {
				zap.S().Warn("Cannot have a cdn with multiple root objects")
			}
			distribution.DefaultRootObject = sunit.IndexDocument
		}
	}
	return distribution
}
