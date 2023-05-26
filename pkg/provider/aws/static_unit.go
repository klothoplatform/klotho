package aws

import (
	"path/filepath"

	"github.com/klothoplatform/klotho/pkg/collectionutil"
	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/multierr"
	"github.com/klothoplatform/klotho/pkg/provider/aws/resources"
	"github.com/pkg/errors"
)

func (a *AWS) expandStaticUnit(dag *core.ResourceGraph, unit *core.StaticUnit) error {
	errs := multierr.Error{}
	createdBuckets := make(map[core.ResourceId]*resources.S3Bucket)
	for _, f := range unit.Files() {
		object, err := core.CreateResource[*resources.S3Object](dag, resources.S3ObjectCreateParams{
			AppName:    a.Config.AppName,
			Refs:       core.AnnotationKeySetOf(unit.AnnotationKey),
			BucketName: unit.Provenance().ID,
			Name:       filepath.Base(f.Path()),
			Key:        f.Path(),
			FilePath:   filepath.Join(unit.ID, f.Path()),
		})
		if err != nil {
			errs.Append(err)
			continue
		}
		createdBuckets[object.Bucket.Id()] = object.Bucket
	}
	if err := errs.ErrOrNil(); err != nil {
		return err
	}
	switch len(createdBuckets) {
	case 0:
		return nil
	case 1:
		_, bucket := collectionutil.GetOneEntry(createdBuckets)
		a.MapResourceDirectlyToConstruct(bucket, unit)

		cfg := a.Config.GetStaticUnit(unit.ID)
		if cfg.ContentDeliveryNetwork.Id != "" {
			distro, err := core.CreateResource[*resources.CloudfrontDistribution](dag, resources.CloudfrontDistributionCreateParams{
				CdnId:   cfg.ContentDeliveryNetwork.Id,
				AppName: a.Config.AppName,
				Refs:    core.AnnotationKeySetOf(unit.AnnotationKey),
			})
			if err != nil {
				return err
			}
			dag.AddDependency(distro, bucket)
		}
		return nil
	default:
		return errors.Errorf(`Found too many buckets for unit %s. This is an internal error.`, unit.Id())
	}
}
