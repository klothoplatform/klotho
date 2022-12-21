package pulumi_aws

import (
	"github.com/klothoplatform/klotho/pkg/config"
	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/provider"
	"github.com/klothoplatform/klotho/pkg/provider/aws"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

type ConfigPlugin struct {
	Config   *config.Application
	Provider provider.Provider
}

func (p ConfigPlugin) Name() string { return "Pulumi:AWS config-set" }

func (p ConfigPlugin) Transform(result *core.CompilationResult, deps *core.Dependencies) error {
	l := zap.S()
	provider, ok := p.Provider.(*aws.AWS)
	if !ok {
		return errors.Errorf("Invalid provider '%s' for config plugin '%s'", p.Provider.Name(), p.Name())
	}
	l.Debugf("Provided defaults: %+v", p.Config.Defaults)
	defaults := config.Defaults{}
	defaults.Merge(provider.GetDefaultConfig())
	defaults.Merge(p.Config.Defaults)
	p.Config.Defaults = defaults
	l.Debugf("Merged result: %+v", p.Config.Defaults)
	return nil
}
