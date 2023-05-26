package aws

import (
	"path/filepath"

	"github.com/klothoplatform/klotho/pkg/collectionutil"
	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/multierr"
	"github.com/klothoplatform/klotho/pkg/provider/aws/resources"
	"github.com/pkg/errors"
	"go.uber.org/zap"
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
	nBuckets := len(createdBuckets)
	zap.L().With()
	if nBuckets > 0 {
		if nBuckets > 1 {
			errs.Append(errors.Errorf(`Found too many buckets for unit %s. This is an internal error.`, unit.Id()))
		}
		_, bucket := collectionutil.GetOneEntry(createdBuckets)
		err := a.MapResourceToConstruct(bucket, unit)
		errs.Append(err)
	}

	return errs.ErrOrNil()
}
