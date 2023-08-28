// Package provider provides the interface for providers which satisfy the [compiler.ProviderPlugin].
//
// A Provider is a centralized location containing all necessary logic to transform an architecture, representated as klotho constructs,
// into the necessary resources to run on the given provider and achieve the same functionality.
//
// In addition to implementing the [compiler.ProviderPlugin] interface, a provider must:
//   - Provide mappings of a Kind of resource (ex. Fs or Execution Unit) to the types supported in the provider (can be services, such as s3 or lambda)
//   - Provide a Default configuration for all of the specified types that the provider offers
//
// The Provider Plugins are responsible for translating the [construct.ConstructGraph] into a [construct.ResourceGraph] with the necessary resources defined by each provider.
// Each specific provider is responsible for generating their own internal representation's of their resources as a [construct.Resource]
//
// These internal representations are what will eventually be used by the [compiler.IaCPlugin] and their fields can be parsed if they meet the following criteria
//   - They are a native Go Type
//   - They satisfy the construct.Resource interface
//   - They are a construct.IaCValue
package provider

import (
	"github.com/klothoplatform/klotho/pkg/construct"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base"
)

type (
	Provider interface {
		Name() string
		ListResources() []construct.Resource
		GetOperationalTempaltes() map[construct.ResourceId]*construct.ResourceTemplate
		GetEdgeTempaltes() map[string]*knowledgebase.EdgeTemplate
		CreateResourceFromId(id construct.ResourceId, dag *construct.ConstructGraph) (construct.Resource, error)
	}
)

const (
	AWS        = "aws"
	DOCKER     = "docker"
	KUBERNETES = "kubernetes"
)
