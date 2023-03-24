package resources

import (
	"github.com/klothoplatform/klotho/pkg/core"
	"go.uber.org/zap"
)

type CloudfrontDistribution struct {
	Id                string
	Origins           []core.AnnotationKey
	DefaultRootObject string
}

func CreateCloudfrontDistribution(resources []core.Construct) *CloudfrontDistribution {
	distribution := &CloudfrontDistribution{}

	for _, res := range resources {
		switch construct := res.(type) {
		case *core.Gateway:
			distribution.Origins = append(distribution.Origins, res.Provenance())
		case *core.StaticUnit:
			distribution.Origins = append(distribution.Origins, res.Provenance())
			if distribution.DefaultRootObject != "" {
				zap.S().Warn("Cannot have a cdn with multiple root objects")
			}
			distribution.DefaultRootObject = construct.IndexDocument
		}
	}
	return distribution
}
