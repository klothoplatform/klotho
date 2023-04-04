package aws

import (
	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/provider/aws/resources"
	"github.com/pkg/errors"
)

func (a *AWS) GenerateSecretsResources(construct *core.Secrets, result *core.ConstructGraph, dag *core.ResourceGraph) error {
	for _, single := range construct.Secrets {
		secret := resources.NewSecret(construct.Provenance(), single, a.Config.AppName)
		dag.AddResource(secret)
		a.MapResourceDirectlyToConstruct(secret, construct)

		secretVersion := resources.NewSecretVersion(secret, single)
		dag.AddResource(secretVersion)
		dag.AddDependency2(secret, secretVersion)

		for _, upstreamCons := range result.GetUpstreamConstructs(construct) {
			unit, isUnit := upstreamCons.(*core.ExecutionUnit)
			if !isUnit {
				continue
			}

			actions := []string{`secretsmanager:DescribeSecret`, `secretsmanager:GetSecretValue`}
			policyResources := []core.IaCValue{{
				Resource: secretVersion,
				Property: core.ARN_IAC_VALUE,
			}}
			policyDoc := resources.CreateAllowPolicyDocument(actions, policyResources)
			policy := resources.NewIamPolicy(a.Config.AppName, construct.Id(), construct.Provenance(), policyDoc)
			if res := dag.GetResource(policy.Id()); res != nil {
				if existingPolicy, ok := res.(*resources.IamPolicy); ok {
					existingPolicy.Policy.Statement = append(existingPolicy.Policy.Statement, policyDoc.Statement...)
					dag.AddDependency2(existingPolicy, secretVersion)
				} else {
					return errors.Errorf("expected resource with id, %s, to be an iam policy", res.Id())
				}
			} else {
				dag.AddResource(policy)
				dag.AddDependency2(policy, secretVersion)
				a.PolicyGenerator.AddAllowPolicyToUnit(unit.Id(), policy)
			}
		}
	}
	return nil
}
