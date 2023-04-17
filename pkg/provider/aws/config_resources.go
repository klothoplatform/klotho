package aws

import (
	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/provider/aws/resources"
	"github.com/pkg/errors"
)

func (a *AWS) GenerateConfigResources(construct *core.Config, result *core.ConstructGraph, dag *core.ResourceGraph) error {
	if construct.Secret {
		cfg := a.Config.GetConfig(construct.ID)
		if cfg.Path == "" {
			return errors.Errorf("'Path' required for config %s", construct.ID)
		}
		secret := resources.NewSecret(construct.Provenance(), cfg.Path, a.Config.AppName)
		dag.AddResource(secret)
		a.MapResourceDirectlyToConstruct(secret, construct)

		secretVersion := resources.NewSecretVersion(secret, cfg.Path)
		dag.AddResource(secretVersion)
		dag.AddDependency(secretVersion, secret)

		for _, upstreamCons := range result.GetUpstreamConstructs(construct) {
			unit, isUnit := upstreamCons.(*core.ExecutionUnit)
			if !isUnit {
				continue
			}

			actions := []string{`secretsmanager:DescribeSecret`, `secretsmanager:GetSecretValue`}
			policyResources := []core.IaCValue{{
				Resource: secret,
				Property: resources.ARN_IAC_VALUE,
			}}
			policyDoc := resources.CreateAllowPolicyDocument(actions, policyResources)
			policy := resources.NewIamPolicy(a.Config.AppName, construct.Id(), construct.Provenance(), policyDoc)
			if res := dag.GetResource(policy.Id()); res != nil {
				if existingPolicy, ok := res.(*resources.IamPolicy); ok {
					existingPolicy.Policy.Statement = append(existingPolicy.Policy.Statement, policyDoc.Statement...)
					dag.AddDependency(existingPolicy, secretVersion)
				} else {
					return errors.Errorf("expected resource with id, %s, to be an iam policy", res.Id())
				}
			} else {
				dag.AddResource(policy)
				dag.AddDependency(policy, secretVersion)
				a.PolicyGenerator.AddAllowPolicyToUnit(unit.Id(), policy)
			}
		}
		return nil
	}

	return errors.New("unsupported")
}
