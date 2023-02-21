package validation

import (
	"fmt"
	"strings"

	"github.com/klothoplatform/klotho/pkg/annotation"
	"github.com/klothoplatform/klotho/pkg/config"
	exec_unit "github.com/klothoplatform/klotho/pkg/exec_unit"
	"github.com/klothoplatform/klotho/pkg/logging"
	"github.com/klothoplatform/klotho/pkg/multierr"
	"github.com/klothoplatform/klotho/pkg/provider"
	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/klothoplatform/klotho/pkg/core"
)

type Plugin struct {
	Provider            provider.Provider
	Config              *config.Application
	UserConfigOverrides config.Application
}

func (p Plugin) Name() string { return "Validation" }

func (p Plugin) Transform(result *core.CompilationResult, deps *core.Dependencies) error {
	var errs multierr.Error
	err := p.handleAnnotations(result)
	errs.Append(err)
	err = p.handleResources(result)
	errs.Append(err)
	err = p.handleProviderValidation(result)
	errs.Append(err)
	p.validateConfigOverrideResourcesExist(result, zap.L().Sugar())
	return errs.ErrOrNil()
}

func (p *Plugin) handleProviderValidation(result *core.CompilationResult) error {

	var errs multierr.Error
	log := zap.L().Sugar()
	for _, resource := range result.Resources() {
		switch resource.Key().Kind {
		case core.InfraAsCodeKind, core.InputFilesKind, core.NetworkLoadBalancerKind, core.TopologyKind, exec_unit.FileDependenciesResourceKind:
		default:
			resourceValid := false
			mapping, shouldValidate := p.Provider.GetKindTypeMappings(resource.Key().Kind)
			resourceType := p.Config.GetResourceType(resource)
			if !shouldValidate {
				log.Debugf("Skipping kind (%s) check (for type %s)", resource.Key().Kind, resourceType)
				continue
			}
			log.Debugf("Checking if provider, %s, supports %s and type, %s, pair.", p.Provider.Name(), resource.Key().Kind, resourceType)
			for _, validType := range mapping {
				if validType == resourceType {
					resourceValid = true
				}
			}
			if !resourceValid {
				errs.Append(errors.Errorf("Provider, %s, Does not support %s and type, %s, pair.\nValid resource types are: %s", p.Provider.Name(), resource.Key().Kind, resourceType, strings.Join(mapping, ", ")))
			}
		}
	}
	return errs.ErrOrNil()
}

// handleAnnotations ensures that every annotation has one resource and only one resource tied to the kind in which it is supposed to produce.
func (p *Plugin) handleAnnotations(result *core.CompilationResult) error {
	var errs multierr.Error
	inputR := result.GetFirstResource(core.InputFilesKind)
	if inputR == nil {
		return nil
	}
	input := inputR.(*core.InputFiles)
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
			p.checkAnnotationForResource(annot, result, log)
		}
	}
	return errs.ErrOrNil()
}

// handleResources ensures that every resource has a unique id and capability pair.
func (p *Plugin) handleResources(result *core.CompilationResult) error {
	var errs multierr.Error
	err := validateNoDuplicateIds[*core.Persist](result)
	errs.Append(err)
	err = validateNoDuplicateIds[*core.Gateway](result)
	errs.Append(err)
	err = validateNoDuplicateIds[*core.ExecutionUnit](result)
	errs.Append(err)
	err = validateNoDuplicateIds[*core.PubSub](result)
	errs.Append(err)
	err = validateNoDuplicateIds[*core.StaticUnit](result)
	errs.Append(err)
	err = validateNoDuplicateIds[*core.Config](result)
	errs.Append(err)
	return errs.ErrOrNil()
}

func (p *Plugin) validateConfigOverrideResourcesExist(result *core.CompilationResult, log *zap.SugaredLogger) {
	for unit := range p.UserConfigOverrides.ExecutionUnits {
		resources := result.GetResourcesOfType(core.ExecutionUnitKind)
		resource := getResourceById(unit, resources)
		if resource == (core.ResourceKey{}) {
			log.Warnf("Unknown execution unit in config override, \"%s\".", unit)
		}
	}

	for persistResource := range p.UserConfigOverrides.Persisted {
		resources := result.GetResourcesOfType(string(core.PersistFileKind))
		resources = append(result.GetResourcesOfType(string(core.PersistKVKind)), resources...)
		resources = append(result.GetResourcesOfType(string(core.PersistORMKind)), resources...)
		resources = append(result.GetResourcesOfType(string(core.PersistRedisClusterKind)), resources...)
		resources = append(result.GetResourcesOfType(string(core.PersistRedisNodeKind)), resources...)
		resources = append(result.GetResourcesOfType(string(core.PersistSecretKind)), resources...)
		resource := getResourceById(persistResource, resources)
		if resource == (core.ResourceKey{}) {
			log.Warnf("Unknown persist in config override, \"%s\".", persistResource)
		}

	}
	for exposeResource := range p.UserConfigOverrides.Exposed {
		resources := result.GetResourcesOfType(core.GatewayKind)
		resource := getResourceById(exposeResource, resources)
		if resource == (core.ResourceKey{}) {
			log.Warnf("Unknown expose in config override, \"%s\".", exposeResource)
		}
	}

	for pubsubResource := range p.UserConfigOverrides.PubSub {
		resources := result.GetResourcesOfType(core.PubSubKind)
		resource := getResourceById(pubsubResource, resources)
		if resource == (core.ResourceKey{}) {
			log.Warnf("Unknown pubsub in config override, \"%s\".", pubsubResource)
		}
	}

	for unit := range p.UserConfigOverrides.StaticUnit {
		resources := result.GetResourcesOfType(core.StaticUnitKind)
		resource := getResourceById(unit, resources)
		if resource == (core.ResourceKey{}) {
			log.Warnf("Unknown static unit in config override, \"%s\".", unit)
		}
	}

	for unit := range p.UserConfigOverrides.Config {
		resources := result.GetResourcesOfType(core.ConfigKind)
		resource := getResourceById(unit, resources)
		if resource == (core.ResourceKey{}) {
			log.Warnf("Unknown config resource in config override, \"%s\".", unit)
		}
	}
}

func (p *Plugin) checkAnnotationForResource(annot *core.Annotation, result *core.CompilationResult, log *zap.SugaredLogger) core.ResourceKey {
	resources := []core.CloudResource{}

	switch annot.Capability.Name {
	case annotation.PersistCapability:
		resources = append(result.GetResourcesOfType(string(core.PersistFileKind)), resources...)
		resources = append(result.GetResourcesOfType(string(core.PersistKVKind)), resources...)
		resources = append(result.GetResourcesOfType(string(core.PersistORMKind)), resources...)
		resources = append(result.GetResourcesOfType(string(core.PersistRedisClusterKind)), resources...)
		resources = append(result.GetResourcesOfType(string(core.PersistRedisNodeKind)), resources...)
		resources = append(result.GetResourcesOfType(string(core.PersistSecretKind)), resources...)
	case annotation.ExecutionUnitCapability:
		resources = append(result.GetResourcesOfType(core.ExecutionUnitKind), resources...)
	case annotation.StaticUnitCapability:
		resources = append(result.GetResourcesOfType(core.StaticUnitKind), resources...)
	case annotation.ExposeCapability:
		resources = append(result.GetResourcesOfType(core.GatewayKind), resources...)
	case annotation.PubSubCapability:
		resources = append(result.GetResourcesOfType(core.PubSubKind), resources...)
	case annotation.ConfigCapability:
		resources = append(result.GetResourcesOfType(core.ConfigKind), resources...)
	case annotation.AssetCapability:
	default:
		log.Warnf("Unknown annotation capability %s.", annot.Capability.Name)
		return core.ResourceKey{}
	}

	resource := getResourceById(annot.Capability.ID, resources)
	if resource == (core.ResourceKey{}) && annot.Capability.Name != annotation.AssetCapability {
		log.Warn("No resource was generated for the annotation.")
	}
	return resource
}

func getResourceById(id string, resources []core.CloudResource) core.ResourceKey {
	var resource core.ResourceKey
	for _, res := range resources {
		if res.Key().Name == id {
			if resource == (core.ResourceKey{}) {
				return res.Key()
			}
		}
	}
	return resource
}

func validateNoDuplicateIds[T core.CloudResource](result *core.CompilationResult) error {
	unitIds := make(map[string]struct{})
	units := core.GetResourcesOfType[T](result)
	for _, unit := range units {
		if _, ok := unitIds[unit.Key().Name]; ok {
			return fmt.Errorf(`multiple Persist objects with the same name, "%s"`, unit.Key().Name)
		}
		unitIds[unit.Key().Name] = struct{}{}
	}
	return nil
}
