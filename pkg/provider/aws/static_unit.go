package aws

import (
	"path/filepath"

	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/multierr"
	"github.com/klothoplatform/klotho/pkg/provider/aws/resources"
)

//func (a *AWS) GenerateStaticUnitResources(unit *core.StaticUnit, dag *core.ResourceGraph) error {
//
//	bucket := resources.NewS3Bucket(unit, a.Config.AppName)
//	bucket.IndexDocument = unit.IndexDocument
//	dag.AddResource(bucket)
//	for _, f := range unit.Files() {
//		object := resources.NewS3Object(bucket, filepath.Base(f.Path()), f.Path(), filepath.Join(unit.ID, f.Path()))
//		dag.AddResource(object)
//		dag.AddDependency(object, bucket)
//		a.MapResourceDirectlyToConstruct(object, unit)
//	}
//	a.MapResourceDirectlyToConstruct(bucket, unit)
//	return nil
//}

func (a *AWS) expandStaticUnit(dag *core.ResourceGraph, unit *core.StaticUnit) error {
	errs := multierr.Error{}
	for _, f := range unit.Files() {
		object, err := core.CreateResource[*resources.S3Object](dag, resources.S3ObjectCreateParams{
			AppName:  a.Config.AppName,
			Refs:     core.AnnotationKeySetOf(unit.AnnotationKey),
			UnitName: unit.Provenance().ID,
			Name:     filepath.Base(f.Path()),
			Key:      f.Path(),
			FilePath: filepath.Join(unit.ID, f.Path()),
		})
		if err != nil {
			errs.Append(err)
			continue
		}
		err = a.MapResourceToConstruct(object, unit)
		errs.Append(err)
	}
	return errs.ErrOrNil()
}
