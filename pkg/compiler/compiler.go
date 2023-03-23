package compiler

import (
	"reflect"

	"github.com/klothoplatform/klotho/pkg/annotation"
	"github.com/klothoplatform/klotho/pkg/core"
	"go.uber.org/zap"
)

type (
	Plugin interface {
		Name() string
	}

	AnalysisAndTransformationPlugin interface {
		Name() string

		// Transform is expected to mutate the result and any dependencies
		Transform(*core.InputFiles, *core.ConstructGraph) error
	}

	ProviderPlugin interface {
		Name() string

		Translate(result *core.ConstructGraph, dag *core.ResourceGraph) (Links []core.CloudResourceLink, err error)
	}

	IaCPlugin interface {
		Name() string

		Translate(cloudGraph *core.ResourceGraph) ([]core.File, error)
	}

	Compiler struct {
		AnalysisAndTransformationPlugins []AnalysisAndTransformationPlugin
		ProviderPlugins                  []ProviderPlugin
		IaCPlugins                       []IaCPlugin
		Document                         CompilationDocument
	}

	// ResourcesOrErr provided as commonly used in async operations for the result channel.
	ResourcesOrErr struct {
		Resources []core.Resource
		Err       error
	}
)

func (c *Compiler) Compile() error {
	// Add our internal resource to be used for provider specific implementations. ex) aws dispatcher requires the payloads bucket and so does proxy
	// TODO: We could likely move this into runtime, but until we refactor that to be common we can keep this here so it lives in one place.
	// We previously always created the payloads bucket so the behavior is no different
	internalResource := &core.InternalResource{AnnotationKey: core.AnnotationKey{ID: core.KlothoPayloadName, Capability: annotation.InternalCapability}}
	c.Document.Constructs.AddConstruct(internalResource)

	for _, p := range c.AnalysisAndTransformationPlugins {
		if isPluginNil(p) {
			continue
		}
		log := zap.L().With(zap.String("plugin", p.Name()))
		log.Debug("starting")
		err := p.Transform(c.Document.InputFiles, c.Document.Constructs)
		if err != nil {
			return core.NewPluginError(p.Name(), err)
		}
		log.Debug("completed")
	}

	for _, p := range c.ProviderPlugins {
		if isPluginNil(p) {
			continue
		}
		log := zap.L().With(zap.String("plugin", p.Name()))
		log.Debug("starting")
		links, err := p.Translate(c.Document.Constructs, c.Document.CloudResources)
		if err != nil {
			return core.NewPluginError(p.Name(), err)
		}
		c.Document.Configuration.AddLinks(links)
		log.Debug("completed")
	}

	return nil
}

func isPluginNil(i Plugin) bool {
	if i == nil {
		return true
	}
	switch reflect.TypeOf(i).Kind() {
	case reflect.Pointer:
		return reflect.ValueOf(i).IsNil()
	}
	return false
}
