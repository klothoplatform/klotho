package aws

import (
	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/provider/aws/resources"
)

func (a *AWS) GenerateFsResources(construct *core.Fs, result *core.ConstructGraph, dag *core.ResourceGraph) error {
	bucket := resources.NewS3Bucket(construct, a.Config.AppName)
	upstreamResources := result.GetUpstreamConstructs(construct)
	for _, res := range upstreamResources {
		unit, ok := res.(*core.ExecutionUnit)
		if ok {
			a.PolicyGenerator.AddAllowPolicyToUnit(unit.Id(), []string{"s3:*"},
				[]core.IaCValue{
					{Resource: bucket, Property: core.ARN_IAC_VALUE},
					{Resource: bucket, Property: core.ALL_BUCKET_DIRECTORY_IAC_VALUE},
				})
		}
	}
	a.MapResourceDirectlyToConstruct(bucket, construct)
	dag.AddResource(bucket)
	return nil
}

func (a *AWS) GenerateRedisResources(construct core.Construct, result *core.ConstructGraph, dag *core.ResourceGraph) error {
	ec := resources.CreateElasticache(a.Config, dag, construct)
	upstreamResources := result.GetUpstreamConstructs(construct)
	for _, res := range upstreamResources {
		unit, ok := res.(*core.ExecutionUnit)
		if ok {
			unit.EnvironmentVariables.Add(core.GenerateRedisHostEnvVar(construct))
			unit.EnvironmentVariables.Add(core.GenerateRedisPortEnvVar(construct))
		}
	}
	a.MapResourceDirectlyToConstruct(ec, construct)
	return nil
}
