package aws

import (
	"fmt"

	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/provider/aws/resources"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

func (a *AWS) expandSecrets(dag *core.ResourceGraph, construct *core.Secrets) error {
	for _, secretName := range construct.Secrets {
		secretVersion, err := core.CreateResource[*resources.SecretVersion](dag, resources.SecretVersionCreateParams{
			AppName:      a.Config.AppName,
			Refs:         core.BaseConstructSetOf(construct),
			Name:         secretName,
			DetectedPath: secretName,
		})

		if err != nil {
			return err
		}

		a.MapResourceDirectlyToConstruct(secretVersion.Secret, construct)
	}
	return nil
}

func (a *AWS) getSecretVersionConfiguration(secretVersion *resources.SecretVersion, result *core.ConstructGraph) (resources.SecretVersionConfigureParams, error) {
	secretVersionConfig := resources.SecretVersionConfigureParams{
		// use unmodified config by default
		Type: secretVersion.Type,
		Path: secretVersion.Path,
	}
	if len(secretVersion.ConstructsRef) > 1 {
		zap.L().Sugar().Debugf("skipping resource configuration: secret version %s has multiple refs, using unmodified config", secretVersion.Id())
		return secretVersionConfig, nil
	}
	var ref core.BaseConstruct
	for r := range secretVersion.ConstructsRef {
		ref = r
	}
	constructR := result.GetConstruct(ref.Id())
	if constructR == nil {
		return secretVersionConfig, fmt.Errorf("construct with id %s does not exist", ref.Id())
	}
	switch construct := constructR.(type) {
	case *core.Config:
		cfg := a.Config.GetConfig(construct.Name)
		if cfg.Path == "" {
			return secretVersionConfig, errors.Errorf("'Path' required for config %s", construct.Name)
		}
		secretVersionConfig.Path = cfg.Path
		secretVersionConfig.Type = "string"
	case *core.Secrets:
		secretVersionConfig.Path = secretVersion.DetectedPath
		secretVersionConfig.Type = "binary"
	default:
		zap.L().Sugar().Debugf("skipping resource configuration: secret version %s has unsupported ref type %T, using unmodified config", secretVersion.Id(), constructR)
		return secretVersionConfig, nil
	}

	return secretVersionConfig, nil
}
