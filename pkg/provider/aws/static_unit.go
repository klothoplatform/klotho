package aws

import (
	"path/filepath"

	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/provider/aws/resources"
)

func (a *AWS) GenerateStaticUnitResources(unit *core.StaticUnit, dag *core.ResourceGraph) error {

	bucket := resources.NewS3Bucket(unit, a.Config.AppName)
	bucket.IndexDocument = unit.IndexDocument
	dag.AddResource(bucket)
	for _, f := range unit.Files() {
		object := resources.NewS3Object(bucket, filepath.Base(f.Path()), f.Path(), filepath.Join(unit.ID, f.Path()))
		dag.AddResource(object)
		dag.AddDependency(object, bucket)
		a.MapResourceDirectlyToConstruct(object, unit)
	}
	a.MapResourceDirectlyToConstruct(bucket, unit)
	return nil
}
