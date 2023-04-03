package aws

import (
	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/provider/aws/resources"
)

func (a *AWS) GenerateSecretsResources(construct *core.Secrets, result *core.ConstructGraph, dag *core.ResourceGraph) error {
	for _, single := range construct.Secrets {
		secret := resources.NewSecret(construct.Provenance(), single, a.Config.AppName)
		dag.AddResource(secret)
		a.MapResourceDirectlyToConstruct(secret, construct)

		secretVersion := resources.NewSecretVersion(secret, single)
		dag.AddResource(secretVersion)
		dag.AddDependency2(secret, secretVersion)
	}
	return nil
}
