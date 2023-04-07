package aws

import (
	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/provider/aws/resources"
	"github.com/pkg/errors"
)

// createCDNs is responsible for generating resources for any constructs who have an id set for their content delivery network.
//
// Each constructs config will be read to determine its cdnId and will be grouped behind the same cdn if the id matches.
func (a *AWS) createCDNs(result *core.ConstructGraph, dag *core.ResourceGraph) error {
	cloudfrontMap := make(map[string][]core.Construct)
	for _, res := range result.ListConstructs() {
		switch construct := res.(type) {
		case *core.Gateway:
			cfg := a.Config.GetExpose(construct.ID)
			cfId := cfg.ContentDeliveryNetwork.Id
			if cfId != "" {
				cf, ok := cloudfrontMap[cfId]
				if ok {
					cloudfrontMap[cfId] = append(cf, res)
				} else {
					cloudfrontMap[cfId] = []core.Construct{res}
				}
			}
		case *core.StaticUnit:
			cfg := a.Config.GetStaticUnit(construct.ID)
			cfId := cfg.ContentDeliveryNetwork.Id
			if cfId != "" {
				cf, ok := cloudfrontMap[cfId]
				if ok {
					cloudfrontMap[cfId] = append(cf, res)
				} else {
					cloudfrontMap[cfId] = []core.Construct{res}
				}
			}
		}
	}

	for name, keys := range cloudfrontMap {
		distro := resources.NewCloudfrontDistribution(a.Config.AppName, name)
		for _, construct := range keys {
			switch construct := construct.(type) {
			case *core.Gateway:
				res, found := a.GetResourcesDirectlyTiedToConstruct(construct)
				if !found {
					return errors.Errorf("Could not find any resource mapped to gateway %s", construct.ID)
				}
				var apiStage *resources.ApiStage
				for _, r := range res {
					if stage, ok := r.(*resources.ApiStage); ok {
						apiStage = stage
					}
				}
				if apiStage == nil {
					return errors.Errorf("Could not find an api stage mapped to gateway %s", construct.ID)
				}
				dag.AddDependency2(distro, apiStage)
				distro.DefaultCacheBehavior.DefaultTtl = 0
				resources.CreateCustomOrigin(construct, apiStage, distro)
				distro.ConstructsRef = append(distro.ConstructsRef, construct.Provenance())
			case *core.StaticUnit:
				res, found := a.GetResourcesDirectlyTiedToConstruct(construct)
				if !found {
					return errors.Errorf("Could not find any resource mapped to static unit %s", construct.ID)
				}
				var bucket *resources.S3Bucket
				for _, r := range res {
					if b, ok := r.(*resources.S3Bucket); ok {
						bucket = b
					}
				}
				if bucket == nil {
					return errors.Errorf("Could not find an api stage mapped to gateway %s", construct.ID)
				}
				dag.AddDependency2(distro, bucket)
				distro.DefaultRootObject = bucket.IndexDocument
				resources.CreateS3Origin(construct, bucket, distro, dag)
				distro.ConstructsRef = append(distro.ConstructsRef, construct.Provenance())
			}
		}
		dag.AddDependenciesReflect(distro)
	}
	return nil
}
