package validation

import (
	"fmt"

	"github.com/klothoplatform/klotho/pkg/annotation"
	"github.com/klothoplatform/klotho/pkg/compiler/types"
	"github.com/klothoplatform/klotho/pkg/config"
	"github.com/klothoplatform/klotho/pkg/logging"
	"github.com/klothoplatform/klotho/pkg/multierr"
	"go.uber.org/zap"

	"github.com/klothoplatform/klotho/pkg/construct"
)

type ConstructValidation struct {
	Config              *config.Application
	UserConfigOverrides config.Application
}

func (p ConstructValidation) Name() string { return "Validation" }

func (p ConstructValidation) Run(input *types.InputFiles, constructGraph *construct.ConstructGraph) error {
	var errs multierr.Error
	err := p.handleAnnotations(input, constructGraph)
	errs.Append(err)
	err = p.handleResources(constructGraph)
	errs.Append(err)
	p.validateConfigOverrideResourcesExist(constructGraph, zap.L().Sugar())
	return errs.ErrOrNil()
}

// handleAnnotations ensures that every annotation has one resource and only one resource tied to the kind in which it is supposed to produce.
func (p *ConstructValidation) handleAnnotations(input *types.InputFiles, constructGraph *construct.ConstructGraph) error {
	var errs multierr.Error
	for _, f := range input.Files() {

		log := zap.L().With(logging.FileField(f)).Sugar()
		ast, ok := f.(*types.SourceFile)
		if !ok {
			// Non-source files can't have any annotations therefore we don't care about checking
			log.Debug("Skipping non-source file")
			continue
		}

		for _, annot := range ast.Annotations() {
			log = log.With(logging.AnnotationField(annot))
			p.checkAnnotationForResource(annot, constructGraph, log)
		}
	}
	return errs.ErrOrNil()
}

// handleResources ensures that every resource has a unique id and capability pair.
func (p *ConstructValidation) handleResources(constructGraph *construct.ConstructGraph) error {
	var errs multierr.Error
	err := validateNoDuplicateIds[*types.Kv](constructGraph)
	errs.Append(err)
	err = validateNoDuplicateIds[*types.Fs](constructGraph)
	errs.Append(err)
	err = validateNoDuplicateIds[*types.Secrets](constructGraph)
	errs.Append(err)
	err = validateNoDuplicateIds[*types.Orm](constructGraph)
	errs.Append(err)
	err = validateNoDuplicateIds[*types.RedisCluster](constructGraph)
	errs.Append(err)
	err = validateNoDuplicateIds[*types.RedisNode](constructGraph)
	errs.Append(err)
	err = validateNoDuplicateIds[*types.Gateway](constructGraph)
	errs.Append(err)
	err = validateNoDuplicateIds[*types.ExecutionUnit](constructGraph)
	errs.Append(err)
	err = validateNoDuplicateIds[*types.PubSub](constructGraph)
	errs.Append(err)
	err = validateNoDuplicateIds[*types.StaticUnit](constructGraph)
	errs.Append(err)
	err = validateNoDuplicateIds[*types.Config](constructGraph)
	errs.Append(err)
	return errs.ErrOrNil()
}

func (p *ConstructValidation) validateConfigOverrideResourcesExist(constructGraph *construct.ConstructGraph, log *zap.SugaredLogger) {
	for unit := range p.UserConfigOverrides.ExecutionUnits {
		resources := constructGraph.GetResourcesOfCapability(annotation.ExecutionUnitCapability)
		resource := getResourceById(unit, resources)
		if resource == nil {
			log.Warnf("Unknown execution unit in config override, \"%s\".", unit)
		}
	}

	for persistResource := range p.UserConfigOverrides.PersistKv {
		resources := []construct.Construct{}
		resources_persist := constructGraph.GetResourcesOfCapability(annotation.PersistCapability)
		for _, res := range resources_persist {
			if _, ok := res.(*types.Kv); ok {
				resources = append(resources, res)
			}
		}
		resource := getResourceById(persistResource, resources)
		if resource == nil {
			log.Warnf("Unknown persist_kv in config override, \"%s\".", persistResource)
		}
	}

	for persistResource := range p.UserConfigOverrides.PersistFs {
		resources := []construct.Construct{}
		resources_persist := constructGraph.GetResourcesOfCapability(annotation.PersistCapability)
		for _, res := range resources_persist {
			if _, ok := res.(*types.Fs); ok {
				resources = append(resources, res)
			}
		}
		resource := getResourceById(persistResource, resources)
		if resource == nil {
			log.Warnf("Unknown persist_fs in config override, \"%s\".", persistResource)
		}
	}

	for persistResource := range p.UserConfigOverrides.PersistOrm {
		resources := []construct.Construct{}
		resources_persist := constructGraph.GetResourcesOfCapability(annotation.PersistCapability)
		for _, res := range resources_persist {
			if _, ok := res.(*types.Orm); ok {
				resources = append(resources, res)
			}
		}
		resource := getResourceById(persistResource, resources)
		if resource == nil {
			log.Warnf("Unknown persist_orm in config override, \"%s\".", persistResource)
		}
	}

	for persistResource := range p.UserConfigOverrides.PersistSecrets {
		resources := []construct.Construct{}
		resources_persist := constructGraph.GetResourcesOfCapability(annotation.PersistCapability)
		for _, res := range resources_persist {
			if _, ok := res.(*types.Secrets); ok {
				resources = append(resources, res)
			}
		}
		resource := getResourceById(persistResource, resources)
		if resource == nil {
			log.Warnf("Unknown persist in config override, \"%s\".", persistResource)
		}
	}

	for persistResource := range p.UserConfigOverrides.PersistRedisCluster {
		resources := []construct.Construct{}
		resources_persist := constructGraph.GetResourcesOfCapability(annotation.PersistCapability)
		for _, res := range resources_persist {
			if _, ok := res.(*types.RedisCluster); ok {
				resources = append(resources, res)
			}
		}
		resource := getResourceById(persistResource, resources)
		if resource == nil {
			log.Warnf("Unknown persist in config override, \"%s\".", persistResource)
		}
	}

	for persistResource := range p.UserConfigOverrides.PersistRedisNode {
		resources := []construct.Construct{}
		resources_persist := constructGraph.GetResourcesOfCapability(annotation.PersistCapability)
		for _, res := range resources_persist {
			if _, ok := res.(*types.RedisNode); ok {
				resources = append(resources, res)
			}
		}
		resource := getResourceById(persistResource, resources)
		if resource == nil {
			log.Warnf("Unknown persist in config override, \"%s\".", persistResource)
		}
	}

	for exposeResource := range p.UserConfigOverrides.Exposed {
		resources := constructGraph.GetResourcesOfCapability(annotation.ExposeCapability)
		resource := getResourceById(exposeResource, resources)
		if resource == nil {
			log.Warnf("Unknown expose in config override, \"%s\".", exposeResource)
		}
	}

	for unit := range p.UserConfigOverrides.StaticUnit {
		resources := constructGraph.GetResourcesOfCapability(annotation.StaticUnitCapability)
		resource := getResourceById(unit, resources)
		if resource == nil {
			log.Warnf("Unknown static unit in config override, \"%s\".", unit)
		}
	}

	for unit := range p.UserConfigOverrides.Config {
		resources := constructGraph.GetResourcesOfCapability(annotation.ConfigCapability)
		resource := getResourceById(unit, resources)
		if resource == nil {
			log.Warnf("Unknown config resource in config override, \"%s\".", unit)
		}
	}
}

func (p *ConstructValidation) checkAnnotationForResource(annot *types.Annotation, constructGraph *construct.ConstructGraph, log *zap.SugaredLogger) construct.Construct {
	resources := []construct.Construct{}

	switch annot.Capability.Name {
	case annotation.PersistCapability:
		resources = append(constructGraph.GetResourcesOfCapability(annotation.PersistCapability), resources...)
	case annotation.ExecutionUnitCapability:
		resources = append(constructGraph.GetResourcesOfCapability(annotation.ExecutionUnitCapability), resources...)
	case annotation.StaticUnitCapability:
		resources = append(constructGraph.GetResourcesOfCapability(annotation.StaticUnitCapability), resources...)
	case annotation.ExposeCapability:
		resources = append(constructGraph.GetResourcesOfCapability(annotation.ExposeCapability), resources...)
	case annotation.PubSubCapability:
		resources = append(constructGraph.GetResourcesOfCapability(annotation.PubSubCapability), resources...)
	case annotation.ConfigCapability:
		resources = append(constructGraph.GetResourcesOfCapability(annotation.ConfigCapability), resources...)
	case annotation.AssetCapability:
	default:
		log.Warnf("Unknown annotation capability %s.", annot.Capability.Name)
		return nil
	}

	resource := getResourceById(annot.Capability.ID, resources)
	if resource == nil && annot.Capability.Name != annotation.AssetCapability {
		log.Warn("No resource was generated for the annotation.")
	}
	return resource
}

func getResourceById(id string, resources []construct.Construct) construct.Construct {
	var resource construct.Construct
	for _, res := range resources {
		if res.Id().Name == id {
			if resource == nil {
				return res
			}
		}
	}
	return resource
}

func validateNoDuplicateIds[T construct.Construct](constructGraph *construct.ConstructGraph) error {
	unitIds := make(map[string]struct{})
	units := construct.GetConstructsOfType[T](constructGraph)
	for _, unit := range units {
		if _, ok := unitIds[unit.Id().Name]; ok {
			return fmt.Errorf(`multiple objects with the same name, "%s"`, unit.Id().Name)
		}
		unitIds[unit.Id().Name] = struct{}{}
	}
	return nil
}
