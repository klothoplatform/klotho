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
			Refs:         core.AnnotationKeySetOf(construct.AnnotationKey),
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
	secretVersionConfig := resources.SecretVersionConfigureParams{}
	if len(secretVersion.ConstructsRef) == 0 {
		// this case may occur when a secret is created as part of edge expansion and is configured as part of that process
		zap.L().Sugar().Debugf("skipping resource configuration: secret version %s has no construct references", secretVersion.Id())
		return secretVersionConfig, nil
	}
	ref, oneRef := secretVersion.ConstructsRef.GetSingle()
	if !oneRef {
		return secretVersionConfig, fmt.Errorf("secret resource may only have one construct reference")
	}
	constructR := result.GetConstruct(ref.ToId())
	if constructR == nil {
		return secretVersionConfig, fmt.Errorf("construct with id %s does not exist", ref.ToId())
	}
	switch construct := constructR.(type) {
	case *core.Config:
		cfg := a.Config.GetConfig(construct.ID)
		if cfg.Path == "" {
			return secretVersionConfig, errors.Errorf("'Path' required for config %s", construct.ID)
		}
		secretVersionConfig.Path = cfg.Path
		secretVersionConfig.Type = "string"
	case *core.Secrets:
		secretVersionConfig.Path = secretVersion.DetectedPath
		secretVersionConfig.Type = "binary"
	default:
		return secretVersionConfig, fmt.Errorf("secret resource must have a construct reference to a config or secrets")
	}

	return secretVersionConfig, nil
}
