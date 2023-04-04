package aws

import (
	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/provider/aws/resources"
)

func (a *AWS) GenerateFsResources(construct *core.Fs, result *core.ConstructGraph, dag *core.ResourceGraph) error {
	bucket := resources.NewS3Bucket(construct, a.Config.AppName)
	dag.AddResource(bucket)
	a.MapResourceDirectlyToConstruct(bucket, construct)
	upstreamResources := result.GetUpstreamConstructs(construct)
	for _, res := range upstreamResources {
		unit, ok := res.(*core.ExecutionUnit)
		if ok {
			actions := []string{"s3:*"}
			policyResources := []core.IaCValue{
				{Resource: bucket, Property: core.ARN_IAC_VALUE},
				{Resource: bucket, Property: core.ALL_BUCKET_DIRECTORY_IAC_VALUE},
			}
			policyDoc := resources.CreateAllowPolicyDocument(actions, policyResources)
			policy := resources.NewIamPolicy(a.Config.AppName, construct.Id(), construct.Provenance(), policyDoc)
			dag.AddResource(policy)
			dag.AddDependency2(policy, bucket)
			a.PolicyGenerator.AddAllowPolicyToUnit(unit.Id(), policy)
		}
	}
	return nil
}

func (a *AWS) GenerateRedisResources(construct core.Construct, result *core.ConstructGraph, dag *core.ResourceGraph) error {
	ec := resources.CreateElasticache(a.Config, dag, construct)
	a.MapResourceDirectlyToConstruct(ec, construct)
	return nil
}
