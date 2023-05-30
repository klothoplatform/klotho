package aws

import (
	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/provider/aws/resources"
)

func (a *AWS) expandFs(dag *core.ResourceGraph, fs core.Construct) error {
	bucket, err := core.CreateResource[*resources.S3Bucket](dag, resources.S3BucketCreateParams{
		AppName: a.Config.AppName,
		Refs:    core.AnnotationKeySetOf(fs.Provenance()),
		Name:    fs.Provenance().ID,
	})
	if err != nil {
		return err
	}
	a.MapResourceDirectlyToConstruct(bucket, fs)
	return nil
}

func getFsConfiguration() resources.S3BucketConfigureParams {
	return resources.S3BucketConfigureParams{
		ForceDestroy: true,
	}
}
