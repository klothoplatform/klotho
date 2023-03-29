package aws

import (
	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/provider/aws/resources"
	"github.com/klothoplatform/klotho/pkg/provider/aws/resources/s3"
)

func (a *AWS) GenerateFsResources(construct *core.Fs, result *core.ConstructGraph, dag *core.ResourceGraph) error {
	accountId := resources.NewAccountId()
	bucket := s3.NewS3Bucket(construct, a.Config.AppName, accountId)
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
	a.ConstructIdToResourceId[construct.Id()] = bucket.Id()
	dag.AddResource(accountId)
	dag.AddResource(bucket)
	dag.AddDependency(accountId, bucket)
	return nil
}
