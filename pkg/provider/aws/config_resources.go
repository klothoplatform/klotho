package aws

import (
	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/provider/aws/resources"
	"github.com/pkg/errors"
)

func (a *AWS) expandConfig(dag *core.ResourceGraph, construct *core.Config) error {
	if !construct.Secret {
		return errors.Errorf("unsupported: non-secret config for annotation '%s'", construct.ID)
	}
	secretVersion, err := core.CreateResource[*resources.SecretVersion](dag, resources.SecretVersionCreateParams{
		AppName: a.Config.AppName,
		Refs:    core.AnnotationKeySetOf(construct.AnnotationKey),
		Name:    construct.ID,
	})
	if err != nil {
		return err
	}

	a.MapResourceDirectlyToConstruct(secretVersion.Secret, construct)
	return nil
}
