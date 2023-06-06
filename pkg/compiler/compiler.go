package compiler

import (
	"bytes"
	"fmt"

	"github.com/klothoplatform/klotho/pkg/annotation"
	"github.com/klothoplatform/klotho/pkg/config"
	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/validation"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

type (
	Plugin interface {
		Name() string
	}

	AnalysisAndTransformationPlugin interface {
		Plugin
		// Transform is expected to mutate the result and any dependencies
		Transform(*core.InputFiles, *core.FileDependencies, *core.ConstructGraph) error
	}

	ProviderPlugin interface {
		Plugin
		Translate(result *core.ConstructGraph, dag *core.ResourceGraph) ([]core.CloudResourceLink, error)
	}

	IaCPlugin interface {
		Plugin
		Translate(cloudGraph *core.ResourceGraph) ([]core.File, error)
	}

	ValidatingPlugin interface {
		Validate(config *config.Application, constructGraph *core.ConstructGraph) error
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

	userOverridesConfiguration := *c.Document.Configuration

	// Add our internal resource to be used for provider specific implementations. ex) aws dispatcher requires the payloads bucket and so does proxy
	// TODO: We could likely move this into runtime, but until we refactor that to be common we can keep this here so it lives in one place.
	// We previously always created the payloads bucket so the behavior is no different
	internalResource := &core.InternalResource{AnnotationKey: core.AnnotationKey{ID: core.KlothoPayloadName, Capability: annotation.InternalCapability}}
	c.Document.Constructs.AddConstruct(internalResource)

	for _, p := range c.AnalysisAndTransformationPlugins {
		log := zap.L().With(zap.String("plugin", p.Name()))
		log.Debug("starting")
		err := p.Transform(c.Document.InputFiles, c.Document.FileDependencies, c.Document.Constructs)
		if err != nil {
			return core.NewPluginError(p.Name(), err)
		}
		log.Debug("completed")
	}

	constructValidation := validation.ConstructValidation{
		Config:              c.Document.Configuration,
		UserConfigOverrides: userOverridesConfiguration,
	}
	err := constructValidation.Run(c.Document.InputFiles, c.Document.Constructs)
	if err != nil {
		return err
	}

	for _, p := range c.ProviderPlugins {
		log := zap.L().With(zap.String("plugin", p.Name()))
		log.Debug("starting")
		if validator, ok := p.(ValidatingPlugin); ok {
			err := validator.Validate(c.Document.Configuration, c.Document.Constructs)
			if err != nil {
				return core.NewPluginError(p.Name(), err)
			}
		}
		links, err := p.Translate(c.Document.Constructs, c.Document.Resources)
		if err != nil {
			return core.NewPluginError(p.Name(), err)
		}
		c.Document.Configuration.AddLinks(links)
		log.Debug("completed")
	}

	for _, p := range c.IaCPlugins {
		// TODO logging
		files, err := p.Translate(c.Document.Resources)
		if err != nil {
			return core.NewPluginError(p.Name(), err)
		}
		c.Document.OutputFiles = append(c.Document.OutputFiles, files...)
	}
	err = c.createConfigOutputFile()
	if err != nil {
		return errors.Wrap(err, "Unable to output Klotho configuration file")
	}
	return c.Document.OutputTo(c.Document.Configuration.OutDir)
}

func (c *Compiler) createConfigOutputFile() error {
	c.Document.Configuration.UpdateForResources(core.ListConstructs[core.Construct](c.Document.Constructs))
	buf := new(bytes.Buffer)
	err := c.Document.Configuration.WriteTo(buf)
	if err != nil {
		return err
	}
	c.Document.OutputFiles = append(c.Document.OutputFiles, &core.RawFile{
		FPath:   fmt.Sprintf("klotho.%s", c.Document.Configuration.Format),
		Content: buf.Bytes(),
	})
	return nil
}
