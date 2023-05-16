package aws

import (
	"fmt"

	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/multierr"
	"github.com/klothoplatform/klotho/pkg/provider/aws/resources"
)

func (a *AWS) GenerateSecretsResources(construct *core.Secrets, result *core.ConstructGraph, dag *core.ResourceGraph) error {
	var merr multierr.Error
	for _, secretName := range construct.Secrets {
		merr.Append(a.generateSecret(construct, result, dag, secretName))
	}
	return merr.ErrOrNil()
}

func (a *AWS) generateSecret(construct core.Construct, result *core.ConstructGraph, dag *core.ResourceGraph, secretName string) error {
	secret := resources.NewSecret(construct.Provenance(), secretName, a.Config.AppName)
	dag.AddResource(secret)
	a.MapResourceDirectlyToConstruct(secret, construct)

	secretVersion := resources.NewSecretVersion(secret, secretName)
	dag.AddDependenciesReflect(secretVersion)

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
		policy := resources.NewIamInlinePolicy(fmt.Sprintf("%s-secretsmanager", secret.Name), []core.AnnotationKey{construct.Provenance()}, policyDoc)
		a.PolicyGenerator.AddInlinePolicyToUnit(unit.Id(), policy)
	}
	return nil
}
