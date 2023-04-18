package aws

import (
	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/pkg/errors"
)

func (a *AWS) GenerateConfigResources(construct *core.Config, result *core.ConstructGraph, dag *core.ResourceGraph) error {
	if construct.Secret {
		cfg := a.Config.GetConfig(construct.ID)
		if cfg.Path == "" {
			return errors.Errorf("'Path' required for config %s", construct.ID)
		}
		return a.generateSecret(construct, result, dag, cfg.Path)
	}

	return errors.Errorf("unsupported: non-secret config for annotation '%s'", construct.ID)
}
