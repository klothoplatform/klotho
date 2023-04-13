package validation

import (
	"fmt"

	"github.com/klothoplatform/klotho/pkg/annotation"
	"github.com/klothoplatform/klotho/pkg/config"
	"github.com/klothoplatform/klotho/pkg/logging"
	"github.com/klothoplatform/klotho/pkg/multierr"
	"go.uber.org/zap"

	"github.com/klothoplatform/klotho/pkg/core"
)

type ConstructValidation struct {
	Config              *config.Application
	UserConfigOverrides config.Application
}

func (p ConstructValidation) Name() string { return "Validation" }

func (p ConstructValidation) Run(input *core.InputFiles, constructGraph *core.ConstructGraph) error {
	var errs multierr.Error
	err := p.handleAnnotations(input, constructGraph)
	errs.Append(err)
	err = p.handleResources(constructGraph)
	errs.Append(err)
	p.validateConfigOverrideResourcesExist(constructGraph, zap.L().Sugar())
	return errs.ErrOrNil()
}

// handleAnnotations ensures that every annotation has one resource and only one resource tied to the kind in which it is supposed to produce.
func (p *ConstructValidation) handleAnnotations(input *core.InputFiles, constructGraph *core.ConstructGraph) error {
	var errs multierr.Error
	for _, f := range input.Files() {

		log := zap.L().With(logging.FileField(f)).Sugar()
		ast, ok := f.(*core.SourceFile)
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
func (p *ConstructValidation) handleResources(constructGraph *core.ConstructGraph) error {
	var errs multierr.Error
	err := validateNoDuplicateIds[*core.Kv](constructGraph)
	errs.Append(err)
	err = validateNoDuplicateIds[*core.Fs](constructGraph)
	errs.Append(err)
	err = validateNoDuplicateIds[*core.Secrets](constructGraph)
	errs.Append(err)
	err = validateNoDuplicateIds[*core.Orm](constructGraph)
	errs.Append(err)
	err = validateNoDuplicateIds[*core.RedisCluster](constructGraph)
	errs.Append(err)
	err = validateNoDuplicateIds[*core.RedisNode](constructGraph)
	errs.Append(err)
	err = validateNoDuplicateIds[*core.Gateway](constructGraph)
	errs.Append(err)
	err = validateNoDuplicateIds[*core.ExecutionUnit](constructGraph)
	errs.Append(err)
	err = validateNoDuplicateIds[*core.PubSub](constructGraph)
	errs.Append(err)
	err = validateNoDuplicateIds[*core.StaticUnit](constructGraph)
	errs.Append(err)
	err = validateNoDuplicateIds[*core.Config](constructGraph)
	errs.Append(err)
	return errs.ErrOrNil()
}

func (p *ConstructValidation) validateConfigOverrideResourcesExist(constructGraph *core.ConstructGraph, log *zap.SugaredLogger) {
	for unit := range p.UserConfigOverrides.ExecutionUnits {
		resources := constructGraph.GetResourcesOfCapability(annotation.ExecutionUnitCapability)
		resource := getResourceById(unit, resources)
		if resource == (core.AnnotationKey{}) {
			log.Warnf("Unknown execution unit in config override, \"%s\".", unit)
		}
	}

	for persistResource := range p.UserConfigOverrides.PersistKv {
		resources := []core.Construct{}
		resources_persist := constructGraph.GetResourcesOfCapability(annotation.PersistCapability)
		for _, res := range resources_persist {
			if _, ok := res.(*core.Kv); ok {
				resources = append(resources, res)
			}
		}
		resource := getResourceById(persistResource, resources)
		if resource == (core.AnnotationKey{}) {
			log.Warnf("Unknown persist_kv in config override, \"%s\".", persistResource)
		}
	}

	for persistResource := range p.UserConfigOverrides.PersistFs {
		resources := []core.Construct{}
		resources_persist := constructGraph.GetResourcesOfCapability(annotation.PersistCapability)
		for _, res := range resources_persist {
			if _, ok := res.(*core.Fs); ok {
				resources = append(resources, res)
			}
		}
		resource := getResourceById(persistResource, resources)
		if resource == (core.AnnotationKey{}) {
			log.Warnf("Unknown persist_fs in config override, \"%s\".", persistResource)
		}
	}

	for persistResource := range p.UserConfigOverrides.PersistOrm {
		resources := []core.Construct{}
		resources_persist := constructGraph.GetResourcesOfCapability(annotation.PersistCapability)
		for _, res := range resources_persist {
			if _, ok := res.(*core.Orm); ok {
				resources = append(resources, res)
			}
		}
		resource := getResourceById(persistResource, resources)
		if resource == (core.AnnotationKey{}) {
			log.Warnf("Unknown persist_orm in config override, \"%s\".", persistResource)
		}
	}

	for persistResource := range p.UserConfigOverrides.PersistSecrets {
		resources := []core.Construct{}
		resources_persist := constructGraph.GetResourcesOfCapability(annotation.PersistCapability)
		for _, res := range resources_persist {
			if _, ok := res.(*core.Secrets); ok {
				resources = append(resources, res)
			}
		}
		resource := getResourceById(persistResource, resources)
		if resource == (core.AnnotationKey{}) {
			log.Warnf("Unknown persist in config override, \"%s\".", persistResource)
		}
	}

	for persistResource := range p.UserConfigOverrides.PersistRedisCluster {
		resources := []core.Construct{}
		resources_persist := constructGraph.GetResourcesOfCapability(annotation.PersistCapability)
		for _, res := range resources_persist {
			if _, ok := res.(*core.RedisCluster); ok {
				resources = append(resources, res)
			}
		}
		resource := getResourceById(persistResource, resources)
		if resource == (core.AnnotationKey{}) {
			log.Warnf("Unknown persist in config override, \"%s\".", persistResource)
		}
	}

	for persistResource := range p.UserConfigOverrides.PersistRedisNode {
		resources := []core.Construct{}
		resources_persist := constructGraph.GetResourcesOfCapability(annotation.PersistCapability)
		for _, res := range resources_persist {
			if _, ok := res.(*core.RedisNode); ok {
				resources = append(resources, res)
			}
		}
		resource := getResourceById(persistResource, resources)
		if resource == (core.AnnotationKey{}) {
			log.Warnf("Unknown persist in config override, \"%s\".", persistResource)
		}
	}

	for exposeResource := range p.UserConfigOverrides.Exposed {
		resources := constructGraph.GetResourcesOfCapability(annotation.ExposeCapability)
		resource := getResourceById(exposeResource, resources)
		if resource == (core.AnnotationKey{}) {
			log.Warnf("Unknown expose in config override, \"%s\".", exposeResource)
		}
	}

	for pubsubResource := range p.UserConfigOverrides.PubSub {
		resources := constructGraph.GetResourcesOfCapability(annotation.PubSubCapability)
		resource := getResourceById(pubsubResource, resources)
		if resource == (core.AnnotationKey{}) {
			log.Warnf("Unknown pubsub in config override, \"%s\".", pubsubResource)
		}
	}

	for unit := range p.UserConfigOverrides.StaticUnit {
		resources := constructGraph.GetResourcesOfCapability(annotation.StaticUnitCapability)
		resource := getResourceById(unit, resources)
		if resource == (core.AnnotationKey{}) {
			log.Warnf("Unknown static unit in config override, \"%s\".", unit)
		}
	}

	for unit := range p.UserConfigOverrides.Config {
		resources := constructGraph.GetResourcesOfCapability(annotation.ConfigCapability)
		resource := getResourceById(unit, resources)
		if resource == (core.AnnotationKey{}) {
			log.Warnf("Unknown config resource in config override, \"%s\".", unit)
		}
	}
}

func (p *ConstructValidation) checkAnnotationForResource(annot *core.Annotation, constructGraph *core.ConstructGraph, log *zap.SugaredLogger) core.AnnotationKey {
	resources := []core.Construct{}

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
		return core.AnnotationKey{}
	}

	resource := getResourceById(annot.Capability.ID, resources)
	if resource == (core.AnnotationKey{}) && annot.Capability.Name != annotation.AssetCapability {
		log.Warn("No resource was generated for the annotation.")
	}
	return resource
}

func getResourceById(id string, resources []core.Construct) core.AnnotationKey {
	var resource core.AnnotationKey
	for _, res := range resources {
		if res.Provenance().ID == id {
			if resource == (core.AnnotationKey{}) {
				return res.Provenance()
			}
		}
	}
	return resource
}

func validateNoDuplicateIds[T core.Construct](constructGraph *core.ConstructGraph) error {
	unitIds := make(map[string]struct{})
	units := core.GetResourcesOfType[T](constructGraph)
	for _, unit := range units {
		if _, ok := unitIds[unit.Provenance().ID]; ok {
			return fmt.Errorf(`multiple objects with the same name, "%s"`, unit.Provenance().ID)
		}
		unitIds[unit.Provenance().ID] = struct{}{}
	}
	return nil
}
