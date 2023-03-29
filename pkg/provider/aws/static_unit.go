package aws

import (
	"path/filepath"

	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/provider/aws/resources/s3"
)

func (a *AWS) GenerateStaticUnitResources(unit *core.StaticUnit, dag *core.ResourceGraph) error {

	bucket := s3.NewS3Bucket(unit, a.Config.AppName)
	if unit.IndexDocument != "" {
		bucket.IndexDocument = unit.IndexDocument
	}
	dag.AddResource(bucket)
	for _, f := range unit.Files() {
		object := s3.NewS3Object(bucket, filepath.Base(f.Path()), f.Path(), filepath.Join(unit.ID, f.Path()))
		dag.AddResource(object)
		dag.AddDependency(bucket, object)
	}
	return nil
}
