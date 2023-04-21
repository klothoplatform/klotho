package aws

import (
	"fmt"

	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/provider/aws/resources"
)

func (a *AWS) GenerateFsResources(construct core.Construct, result *core.ConstructGraph, dag *core.ResourceGraph) error {
	bucket := resources.NewS3Bucket(construct, a.Config.AppName)
	dag.AddResource(bucket)
	a.MapResourceDirectlyToConstruct(bucket, construct)
	actions := []string{"s3:*"}
	policyResources := []core.IaCValue{
		{Resource: bucket, Property: resources.ARN_IAC_VALUE},
		{Resource: bucket, Property: resources.ALL_BUCKET_DIRECTORY_IAC_VALUE},
	}
	policyDoc := resources.CreateAllowPolicyDocument(actions, policyResources)
	policy := resources.NewIamInlinePolicy(fmt.Sprintf("%s-s3", construct.Id()), construct.Provenance(), policyDoc)
	upstreamResources := result.GetUpstreamConstructs(construct)
	for _, res := range upstreamResources {
		unit, ok := res.(*core.ExecutionUnit)
		if ok {
			a.PolicyGenerator.AddInlinePolicyToUnit(unit.Id(), policy)
		}
	}
	return nil
}

func (a *AWS) GenerateRedisResources(construct core.Construct, result *core.ConstructGraph, dag *core.ResourceGraph) error {
	ec := resources.CreateElasticache(a.Config, dag, construct)
	a.MapResourceDirectlyToConstruct(ec, construct)
	return nil
}
