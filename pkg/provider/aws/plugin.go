package aws

import (
	"sort"
	"strings"

	"github.com/klothoplatform/klotho/pkg/collectionutil"
	"github.com/klothoplatform/klotho/pkg/core"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base"
	"github.com/klothoplatform/klotho/pkg/multierr"
	"github.com/klothoplatform/klotho/pkg/provider/aws/resources"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

// ExpandConstructs looks at all existing constructs in the construct graph and turns them into their respective AWS Resources
func (a *AWS) ExpandConstructs(result *core.ConstructGraph, dag *core.ResourceGraph) (err error) {
	log := zap.S()
	var merr multierr.Error
	for _, construct := range result.ListConstructs() {
		log.Debugf("Converting construct with id, %s, to aws resources", construct.Id())
		switch construct := construct.(type) {
		case *core.ExecutionUnit:
			merr.Append(a.expandExecutionUnit(dag, construct))
		case *core.Gateway:
			merr.Append(a.expandExpose(dag, construct))
		case *core.Orm:
			merr.Append(a.expandOrm(dag, construct))
		case *core.Fs:
			merr.Append(a.expandFs(dag, construct))
		case *core.Kv:
			merr.Append(a.expandKv(dag, construct))
		case *core.RedisNode:
			merr.Append(a.expandRedisNode(dag, construct))
		case *core.StaticUnit:
			merr.Append(a.expandStaticUnit(dag, construct))
		}
	}
	return merr.ErrOrNil()
}

// CopyConstructEdgesToDag looks at the dependencies which existed in the construct graph and copies those dependencies into the resource graph so that the edges can be later expanded on
func (a *AWS) CopyConstructEdgesToDag(result *core.ConstructGraph, dag *core.ResourceGraph) (err error) {
	var merr multierr.Error
	for _, dep := range result.ListDependencies() {
		sourceResource := a.GetResourceTiedToConstruct(dep.Source)
		if sourceResource == nil {
			merr.Append(errors.Errorf("unable to copy edge, no resource tied to construct %s", dep.Source.Id()))
			continue
		}
		targetResource := a.GetResourceTiedToConstruct(dep.Destination)
		if targetResource == nil {
			merr.Append(errors.Errorf("unable to copy edge, no resource tied to construct %s", dep.Destination.Id()))
			continue
		}

		data := knowledgebase.EdgeData{AppName: a.Config.AppName, Source: sourceResource, Destination: targetResource}
		switch construct := dep.Source.(type) {
		case *core.ExecutionUnit:
			switch dep.Destination.(type) {
			case *core.Orm:
				data.Constraint = knowledgebase.EdgeConstraint{
					NodeMustExist: []core.Resource{&resources.RdsProxy{}},
				}
			case *core.Kv:
				data.Constraint = knowledgebase.EdgeConstraint{
					NodeMustNotExist: []core.Resource{&resources.IamPolicy{}},
				}
			}
			for _, envVar := range construct.EnvironmentVariables {
				if envVar.Construct == dep.Destination {
					data.EnvironmentVariables = append(data.EnvironmentVariables, envVar)
				}
			}
		case *core.Gateway:
			for _, route := range construct.Routes {
				if route.ExecUnitName == dep.Destination.Provenance().ID {
					data.Routes = append(data.Routes, route)
				}
			}
		}
		dag.AddDependencyWithData(sourceResource, targetResource, data)
	}
	return merr.ErrOrNil()
}

// configureResources calls every resource's Configure method, for resources that exist in the graph
func (a *AWS) configureResources(result *core.ConstructGraph, dag *core.ResourceGraph) (err error) {
	var merr multierr.Error
	for _, resource := range dag.ListResources() {
		var configuration any
		switch res := resource.(type) {
		case *resources.LambdaFunction:
			configuration, err = a.getLambdaConfiguration(result, dag, res.ConstructsRef)
			if err != nil {
				merr.Append(err)
				continue
			}
		case *resources.RdsInstance:
			configuration, err = a.getRdsConfiguration(result, dag, res.ConstructsRef)
			if err != nil {
				merr.Append(err)
				continue
			}
		case *resources.EcrImage:
			configuration, err = a.getImageConfiguration(result, dag, res.ConstructsRef)
			if err != nil {
				merr.Append(err)
				continue
			}
		case *resources.DynamodbTable:
			configuration = a.getKvConfiguration()
		case *resources.EksNodeGroup:
			configuration, err = a.getNodeGroupConfiguration(result, dag, res.ConstructsRef)
			if err != nil {
				merr.Append(err)
				continue
			}
		case *resources.ElasticacheCluster:
			configuration, err = a.getElasticacheConfiguration(result, res.ConstructsRef)
			if err != nil {
				merr.Append(err)
				continue
			}
		case *resources.S3Bucket:
			configuration, err = getS3BucketConfig(res, result)
			if err != nil {
				merr.Append(err)
				continue
			}
		case *resources.S3Object:
			configuration = a.getStaticUnitObjectConfiguration()
		}
		merr.Append(dag.CallConfigure(resource, configuration))
	}
	return merr.ErrOrNil()
}

func getS3BucketConfig(bucket *resources.S3Bucket, constructs *core.ConstructGraph) (resources.S3BucketConfigureParams, error) {
	staticUnits := make(map[string]*core.StaticUnit)
	for consRef, _ := range bucket.ConstructsRef {
		cons := constructs.GetConstruct(consRef.ToId())
		if oneUnit, isUnit := cons.(*core.StaticUnit); isUnit {
			staticUnits[oneUnit.ID] = oneUnit
		}
	}
	switch len(staticUnits) {
	case 0:
		// None of the bucket's constructs were static unit; assume it's an FS
		return getFsConfiguration(), nil
	case 1:
		// The bucket came from a single static unit; gets its params
		_, unit := collectionutil.GetOneEntry(staticUnits)
		params := resources.S3BucketConfigureParams{
			ForceDestroy:  true,
			IndexDocument: unit.IndexDocument,
		}
		return params, nil
	default:
		// The bucket came from multiple static units; this is an error
		ids := collectionutil.Keys(staticUnits)
		sort.Strings(ids)
		return resources.S3BucketConfigureParams{}, errors.Errorf(
			`couldn't resolve configuration for bucket "%s" because I found multiple static units for it: %s`,
			bucket.Id().String(),
			strings.Join(ids, ","),
		)
	}
}

func (a *AWS) Translate(result *core.ConstructGraph, dag *core.ResourceGraph) (links []core.CloudResourceLink, err error) {
	err = a.ExpandConstructs(result, dag)
	if err != nil {
		return
	}
	err = a.CopyConstructEdgesToDag(result, dag)
	if err != nil {
		return
	}
	err = a.KnowledgeBase.ExpandEdges(dag, a.Config.AppName)
	if err != nil {
		return
	}
	err = a.configureResources(result, dag)
	if err != nil {
		return
	}
	err = a.KnowledgeBase.ConfigureFromEdgeData(dag)
	return
}
