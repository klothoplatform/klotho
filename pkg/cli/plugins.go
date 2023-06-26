package cli

import (
	"net/http"

	compiler "github.com/klothoplatform/klotho/pkg/compiler"
	"github.com/klothoplatform/klotho/pkg/config"
	"github.com/klothoplatform/klotho/pkg/engine"
	envvar "github.com/klothoplatform/klotho/pkg/env_var"
	execunit "github.com/klothoplatform/klotho/pkg/exec_unit"
	"github.com/klothoplatform/klotho/pkg/infra/iac2"
	"github.com/klothoplatform/klotho/pkg/lang/csharp"
	csRuntimes "github.com/klothoplatform/klotho/pkg/lang/csharp/runtimes"
	"github.com/klothoplatform/klotho/pkg/lang/golang"
	goRuntimes "github.com/klothoplatform/klotho/pkg/lang/golang/runtimes"
	"github.com/klothoplatform/klotho/pkg/lang/javascript"
	jsRuntimes "github.com/klothoplatform/klotho/pkg/lang/javascript/runtimes"
	"github.com/klothoplatform/klotho/pkg/lang/python"
	pyRuntimes "github.com/klothoplatform/klotho/pkg/lang/python/runtimes"
	"github.com/klothoplatform/klotho/pkg/multierr"
	"github.com/klothoplatform/klotho/pkg/provider/providers"
	staticunit "github.com/klothoplatform/klotho/pkg/static_unit"
	"github.com/klothoplatform/klotho/pkg/visualizer"
)

// PluginSetBuilder is a crude "plugin dependency" helper struct for managing the order of plugins via stages.
// TODO improve the flexibility and expressivity to capture the real dependencies between plugins.
type PluginSetBuilder struct {
	AnalysisAndTransform []compiler.AnalysisAndTransformationPlugin
	IaC                  []compiler.IaCPlugin
	Engine               *engine.Engine
	Cfg                  *config.Application
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
		b.AddVisualizerPlugin,
		b.AddEngine,
	} {
		merr.Append(f())
	}
	return merr.ErrOrNil()
}

func (b *PluginSetBuilder) AddEngine() error {
	provider, err := providers.GetProvider(b.Cfg)
	if err != nil {
		return err
	}
	kb, err := providers.GetKnowledgeBase(b.Cfg)
	if err != nil {
		return err
	}
	b.Engine = engine.NewEngine(provider, kb)
	return nil
}

func (b *PluginSetBuilder) AddVisualizerPlugin() error {
	b.IaC = append(b.IaC, visualizer.Plugin{AppName: b.Cfg.AppName, Provider: b.Cfg.Provider, Client: http.DefaultClient})
	return nil
}

func (b *PluginSetBuilder) AddExecUnit() error {
	b.AnalysisAndTransform = append(b.AnalysisAndTransform,
		staticunit.StaticUnitSplit{Config: b.Cfg},
		execunit.ExecUnitPlugin{Config: b.Cfg},
		// Configure executables and include assets after exec split
		// to make sure all input files are in the proper units for the PostSplit plugins
		javascript.NodeJSExecutable{},
		python.PythonExecutable{},
		golang.GolangExecutable{},
		csharp.CSharpExecutable{Config: b.Cfg},
		execunit.PruneUncategorizedFiles{},
		execunit.Assets{},
		envvar.EnvVarInjection{Config: b.Cfg})
	return nil
}

func (b *PluginSetBuilder) AddJavascript() error {
	jsRuntime, err := jsRuntimes.GetRuntime(b.Cfg)
	if err != nil {
		return err
	}

	b.AnalysisAndTransform = append(b.AnalysisAndTransform, javascript.NewJavascriptPlugins(b.Cfg, jsRuntime))
	return nil
}

func (b *PluginSetBuilder) AddPython() error {
	pyRuntime, err := pyRuntimes.GetRuntime(b.Cfg)
	if err != nil {
		return err
	}

	b.AnalysisAndTransform = append(b.AnalysisAndTransform, python.NewPythonPlugins(b.Cfg, pyRuntime))
	return nil
}

func (b *PluginSetBuilder) AddGo() error {
	goRuntime, err := goRuntimes.GetRuntime(b.Cfg)
	if err != nil {
		return err
	}

	b.AnalysisAndTransform = append(b.AnalysisAndTransform, golang.NewGoPlugins(b.Cfg, goRuntime))
	return nil
}

func (b *PluginSetBuilder) AddCSharp() error {
	csRuntime, err := csRuntimes.GetRuntime(b.Cfg)
	if err != nil {
		return err
	}
	b.AnalysisAndTransform = append(b.AnalysisAndTransform, csharp.NewCSharpPlugins(b.Cfg, csRuntime))
	return nil
}

func (b *PluginSetBuilder) AddPulumi() error {
	b.IaC = append(b.IaC, iac2.Plugin{Config: b.Cfg})
	return nil
}
