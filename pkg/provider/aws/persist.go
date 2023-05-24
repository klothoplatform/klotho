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
	policy := resources.NewIamInlinePolicy(fmt.Sprintf("%s-s3", construct.Id()), []core.AnnotationKey{construct.Provenance()}, policyDoc)
	upstreamResources := result.GetUpstreamConstructs(construct)
	for _, res := range upstreamResources {
		unit, ok := res.(*core.ExecutionUnit)
		if ok {
			a.PolicyGenerator.AddInlinePolicyToUnit(unit.Id(), policy)
		}
	}
	return nil
}

func (a *AWS) expandRedisNode(dag *core.ResourceGraph, construct *core.RedisNode) error {
	redis, err := core.CreateResource[*resources.ElasticacheCluster](dag, resources.ElasticacheClusterCreateParams{
		AppName: a.Config.AppName,
		Refs:    []core.AnnotationKey{construct.AnnotationKey},
		Name:    construct.ID,
	})
	if err != nil {
		return err
	}
	return a.MapResourceToConstruct(redis, construct)
}
