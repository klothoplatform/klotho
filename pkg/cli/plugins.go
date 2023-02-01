package cli

import (
	"github.com/klothoplatform/klotho/pkg/config"
	"github.com/klothoplatform/klotho/pkg/core"
	envvar "github.com/klothoplatform/klotho/pkg/env_var"
	execunit "github.com/klothoplatform/klotho/pkg/exec_unit"
	"github.com/klothoplatform/klotho/pkg/infra/kubernetes"
	"github.com/klothoplatform/klotho/pkg/infra/pulumi_aws"
	"github.com/klothoplatform/klotho/pkg/lang/csharp"
	csRuntimes "github.com/klothoplatform/klotho/pkg/lang/csharp/runtimes"
	"github.com/klothoplatform/klotho/pkg/lang/golang"
	goRuntimes "github.com/klothoplatform/klotho/pkg/lang/golang/runtimes"
	"github.com/klothoplatform/klotho/pkg/lang/javascript"
	jsRuntimes "github.com/klothoplatform/klotho/pkg/lang/javascript/runtimes"
	"github.com/klothoplatform/klotho/pkg/lang/python"
	pyRuntimes "github.com/klothoplatform/klotho/pkg/lang/python/runtimes"
	"github.com/klothoplatform/klotho/pkg/multierr"
	"github.com/klothoplatform/klotho/pkg/provider"
	"github.com/klothoplatform/klotho/pkg/provider/providers"
	staticunit "github.com/klothoplatform/klotho/pkg/static_unit"
	"github.com/klothoplatform/klotho/pkg/topology"
	"github.com/klothoplatform/klotho/pkg/validation"
)

// PluginSetBuilder is a crude "plugin dependency" helper struct for managing the order of plugins via stages.
// TODO improve the flexibility and expressivity to capture the real dependencies between plugins.
type PluginSetBuilder struct {
	Cfg       *config.Application
	Parse     []core.Plugin
	Units     []core.Plugin
	Transform []core.Plugin
	Topology  []core.Plugin
	Infra     []core.Plugin

	provider provider.Provider
}

func (b PluginSetBuilder) Plugins() []core.Plugin {
	var plugins []core.Plugin
	plugins = append(plugins, b.Parse...)
	plugins = append(plugins, b.Units...)
	plugins = append(plugins, b.Transform...)
	plugins = append(plugins, b.Topology...)
	plugins = append(plugins, b.Infra...)
	return plugins
}

func (b *PluginSetBuilder) AddAll() error {
	var merr multierr.Error
	for _, f := range []func() error{
		b.AddExecUnit,
		b.AddJavascript,
		b.AddPython,
		b.AddGo,
		b.AddCSharp,
		b.AddPulumi,
		b.AddPostCompilation,
	} {
		merr.Append(f())
	}
	return merr.ErrOrNil()
}

func (b *PluginSetBuilder) AddPostCompilation() error {
	if err := b.setupProvider(); err != nil {
		return err
	}

	b.Transform = append(b.Transform, envvar.EnvVarInjection{Config: b.Cfg}, validation.Plugin{Provider: b.provider, Config: b.Cfg, UserConfigOverrides: *b.Cfg}, kubernetes.Kubernetes{Config: b.Cfg})
	b.Topology = append(b.Topology, topology.Plugin{Config: b.Cfg})
	return nil
}

func (b *PluginSetBuilder) AddExecUnit() error {
	b.Units = append(b.Units,
		staticunit.StaticUnitSplit{Config: b.Cfg},
		execunit.ExecUnitPlugin{Config: b.Cfg},
		// Configure executables and include assets after exec split
		// to make sure all input files are in the proper units for the PostSplit plugins
		javascript.NodeJSExecutable{},
		python.PythonExecutable{},
		golang.GolangExecutable{},
		csharp.CSharpExecutable{Config: b.Cfg},
		execunit.PruneUncategorizedFiles{},
		execunit.Assets{})
	return nil
}

func (b *PluginSetBuilder) AddJavascript() error {
	jsRuntime, err := jsRuntimes.GetRuntime(b.Cfg)
	if err != nil {
		return err
	}

	b.Transform = append(b.Transform, javascript.NewJavascriptPlugins(b.Cfg, jsRuntime))
	return nil
}

func (b *PluginSetBuilder) AddPython() error {
	pyRuntime, err := pyRuntimes.GetRuntime(b.Cfg)
	if err != nil {
		return err
	}

	b.Transform = append(b.Transform, python.NewPythonPlugins(b.Cfg, pyRuntime))
	return nil
}

func (b *PluginSetBuilder) AddGo() error {
	goRuntime, err := goRuntimes.GetRuntime(b.Cfg)
	if err != nil {
		return err
	}

	b.Transform = append(b.Transform, golang.NewGoPlugins(b.Cfg, goRuntime))
	return nil
}

func (b *PluginSetBuilder) AddCSharp() error {
	csRuntime, err := csRuntimes.GetRuntime(b.Cfg)
	if err != nil {
		return err
	}
	b.Transform = append(b.Transform, csharp.NewCSharpPlugins(b.Cfg, csRuntime))
	return nil
}

func (b *PluginSetBuilder) setupProvider() (err error) {
	if b.provider != nil {
		return nil
	}

	b.provider, err = providers.GetProvider(b.Cfg)
	if err == nil {
		b.Infra = append(b.Infra, b.provider)
	}
	return
}

func (b *PluginSetBuilder) AddPulumi() error {
	if err := b.setupProvider(); err != nil {
		return err
	}
	b.Parse = append(b.Parse, pulumi_aws.ConfigPlugin{Config: b.Cfg, Provider: b.provider})
	b.Infra = append(b.Infra, pulumi_aws.Plugin{Config: b.Cfg})
	return nil
}
