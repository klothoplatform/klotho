// Package provider provides the interface for providers which satisfy the [compiler.ProviderPlugin].
//
// A Provider is a centralized location containing all necessary logic to transform an architecture, representated as klotho constructs,
// into the necessary resources to run on the given provider and achieve the same functionality.
//
// In addition to implementing the [compiler.ProviderPlugin] interface, a provider must:
//   - Provide mappings of a Kind of resource (ex. Fs or Execution Unit) to the types supported in the provider (can be services, such as s3 or lambda)
//   - Provide a Default configuration for all of the specified types that the provider offers
//
// The Provider Plugins are responsible for translating the [core.ConstructGraph] into a [core.ResourceGraph] with the necessary resources defined by each provider.
// Each specific provider is responsible for generating their own internal representation's of their resources as a [core.Resource]
//
// These internal representations are what will eventually be used by the [compiler.IaCPlugin] and their fields can be parsed if they meet the following criteria
//   - They are a native Go Type
//   - They satisfy the core.Resource interface
//   - They are a core.IaCValue
package provider

import (
	"reflect"

	"github.com/klothoplatform/klotho/pkg/compiler"
	"github.com/klothoplatform/klotho/pkg/config"
	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/multierr"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

type (
	Provider interface {
		compiler.ProviderPlugin
		GetKindTypeMappings(construct core.Construct) []string
		GetDefaultConfig() config.Defaults
		compiler.ValidatingPlugin
		CreateResourceFromId(id core.ResourceId, dag *core.ResourceGraph) (core.Resource, error)
		ExpandConstruct(construct core.Construct, dag *core.ResourceGraph, constructType string) (directlyMappedResources []core.Resource, err error)
	}

	TemplateConfig struct {
		Datadog bool
		Lumigo  bool
		AppName string
	}
)

// HandleProviderValidation ensures that the klotho configuration and construct graph are valid for the provider
//
// The current checks consist of:
//   - types defined in klotho configuration for each construct is valid for the provider
func HandleProviderValidation(p Provider, config *config.Application, constructGraph *core.ConstructGraph) error {

	var errs multierr.Error
	log := zap.L().Sugar()
	for _, resource := range core.ListConstructs[core.Construct](constructGraph) {
		if _, ok := resource.(*core.InternalResource); ok {
			continue
		}
		resourceValid := false
		mapping := p.GetKindTypeMappings(resource)
		if len(mapping) == 0 {
			errs.Append(errors.Errorf(`Provider "%s" does not support %s `, p.Name(), reflect.ValueOf(resource).Type()))
			continue
		}
		resourceType := config.GetResourceType(resource)
		log.Debugf("Checking if provider, %s, supports %s and type, %s, pair.", p.Name(), resource.AnnotationCapability(), resourceType)
		for _, validType := range mapping {
			if validType == resourceType {
				resourceValid = true
			}
		}
		if !resourceValid {
			errs.Append(errors.Errorf(`Provider "%s" does not support %s of type %s.\nValid resource types are: %v`, p.Name(), reflect.ValueOf(resource).Type(), resourceType, mapping))
		}
	}

	return errs.ErrOrNil()
}
