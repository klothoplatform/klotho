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
	"github.com/klothoplatform/klotho/pkg/core"
)

type (
	Provider interface {
		Name() string
		LoadResources(graph core.InputGraph, resources map[core.ResourceId]core.BaseConstruct) error
		CreateResourceFromId(id core.ResourceId, dag *core.ConstructGraph) (core.Resource, error)
		ExpandConstruct(construct core.Construct, dag *core.ResourceGraph, constructType string) (directlyMappedResources []core.Resource, err error)
	}
)

const (
	AWS        = "aws"
	KUBERNETES = "kubernetes"
)
