package aws

import (
	"sort"
	"strings"

	"github.com/klothoplatform/klotho/pkg/collectionutil"
	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/graph"
	"github.com/klothoplatform/klotho/pkg/infra/kubernetes"
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
	for _, construct := range core.ListConstructs[core.BaseConstruct](result) {
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
		case *core.InternalResource:
			merr.Append(a.expandFs(dag, construct))
		case *core.Kv:
			merr.Append(a.expandKv(dag, construct))
		case *core.RedisNode:
			merr.Append(a.expandRedisNode(dag, construct))
		case *core.StaticUnit:
			merr.Append(a.expandStaticUnit(dag, construct))
		case *core.Secrets:
			merr.Append(a.expandSecrets(dag, construct))
		case *core.Config:
			merr.Append(a.expandConfig(dag, construct))
		case core.Resource:
			dag.AddResource(construct)
		}
	}
	return merr.ErrOrNil()
}

// CopyConstructEdgesToDag looks at the dependencies which existed in the construct graph and copies those dependencies into the resource graph so that the edges can be later expanded on
func (a *AWS) CopyConstructEdgesToDag(result *core.ConstructGraph, dag *core.ResourceGraph) (err error) {
	var merr multierr.Error
	for _, dep := range result.ListDependencies() {
		sourceResources, ok := a.GetResourcesDirectlyTiedToConstruct(dep.Source)
		if !ok {
			merr.Append(errors.Errorf("unable to copy edge, no resource tied to construct %s", dep.Source.Id()))
			continue
		}
		for _, sourceResource := range sourceResources {
			destinationResources, ok := a.GetResourcesDirectlyTiedToConstruct(dep.Destination)
			if !ok {
				merr.Append(errors.Errorf("unable to copy edge, no resource tied to construct %s", dep.Destination.Id()))
				continue
			}
			for _, destinationResource := range destinationResources {
				merr.Append(a.copyConstructEdgeToDag(sourceResource, destinationResource, dep, dag))
			}
		}
	}
	return merr.ErrOrNil()
}

func (a *AWS) copyConstructEdgeToDag(
	sourceResource core.Resource,
	destinationResource core.Resource,
	dep graph.Edge[core.BaseConstruct],
	dag *core.ResourceGraph) error {
	data := knowledgebase.EdgeData{AppName: a.Config.AppName, Source: sourceResource, Destination: destinationResource}

	switch construct := dep.Source.(type) {
	case *core.ExecutionUnit:
		if a.Config.GetExecutionUnit(construct.ID).Type == kubernetes.KubernetesType {
			data.SourceRef = construct.AnnotationKey
		}
		for _, envVar := range construct.EnvironmentVariables {
			if envVar.Construct != nil && envVar.Construct.Id() == dep.Destination.Id() {
				data.EnvironmentVariables = append(data.EnvironmentVariables, envVar)
			}
		}
		switch dest := dep.Destination.(type) {
		case *core.Orm:
			data.Constraint = knowledgebase.EdgeConstraint{
				NodeMustExist: []core.Resource{&resources.RdsProxy{}},
			}
			// Because we dont understand resource types, our edges cant depict what is truly valid and what isnt yet. Example
			// 	found multiple paths which satisfy constraints for edge aws:lambda_function:lambda -> aws:rds_instance:rds.
			// Paths: [[{*resources.LambdaFunction *resources.RdsProxy} {*resources.RdsProxyTargetGroup *resources.RdsProxy} {*resources.RdsProxyTargetGroup *resources.RdsInstance}]
			// [{*resources.LambdaFunction *resources.RdsProxy} {*kubernetes.HelmChart *resources.RdsProxy} {*kubernetes.HelmChart *resources.RdsInstance}]]
			// When we can classify lambda and helm as compute then we can understand that there is no need for the data flow in this manner
			if a.Config.GetExecutionUnit(construct.ID).Type == kubernetes.KubernetesType {
				data.Constraint.NodeMustNotExist = append(data.Constraint.NodeMustNotExist, &resources.LambdaFunction{})
			} else {
				data.Constraint.NodeMustNotExist = append(data.Constraint.NodeMustNotExist, &kubernetes.HelmChart{})
			}
		case *core.Kv:
			data.Constraint = knowledgebase.EdgeConstraint{
				NodeMustNotExist: []core.Resource{&resources.IamPolicy{}},
			}
		case *core.ExecutionUnit:
			// We have to handle this case here since we dont understand what exists within a helm chart yet outside of the notion of constructs
			// We will be able to move this to edges once we build a better understanding of kubernetes resources
			if a.Config.GetExecutionUnit(dest.ID).Type == kubernetes.KubernetesType && a.Config.GetExecutionUnit(construct.ID).Type == kubernetes.KubernetesType {
				if chart, ok := destinationResource.(*kubernetes.HelmChart); ok {
					err := a.handleEksProxy(construct, dest, chart, dag)
					if err != nil {
						return err
					}
				} else {
					return errors.Errorf("unable to copy edge %s -> %s, target resource must be a helm chart for kubernetes proxy", dep.Source.Id(), dep.Destination.Id())
				}
			}
		}
	case *core.Kv:
		data.Constraint = knowledgebase.EdgeConstraint{
			NodeMustNotExist: []core.Resource{&resources.IamPolicy{}},
		}
	case *core.Gateway:
		for _, route := range construct.Routes {
			dstCons, ok := dep.Destination.(core.Construct)
			if !ok {
				zap.S().Warnf(`Can't connect %s to concrete resource %s`, construct.Id(), dep.Destination.Id())
				continue
			}
			if route.ExecUnitName == dstCons.Provenance().ID {
				data.Routes = append(data.Routes, route)
			}
			// Because we don't have an understanding of what exists within the helm chart we cannot expand API -> Chart (we would need API -> k8s Service)
			// To fix this we find the Target group being created for the value injected into the TargetGroupBinding Manifest and make that the destinationResources
			if chart, ok := destinationResource.(*kubernetes.HelmChart); ok {
				var destinationTG *resources.TargetGroup
				for _, val := range chart.Values {
					if iacVal, ok := val.(core.IaCValue); ok {
						if tg, ok := iacVal.Resource.(*resources.TargetGroup); ok {
							for ref := range tg.ConstructsRef {
								if ref.ID == dstCons.Provenance().ID {
									destinationTG = tg
								}
							}
						}
					}
				}
				if destinationTG == nil {
					return errors.Errorf("unable to find target group for edge, %s -> %s", dep.Source.Id(), dep.Destination.Id())
				}
				destinationResource = destinationTG
				data.Destination = destinationTG
			}
		}
	}

	dag.AddDependencyWithData(sourceResource, destinationResource, data)
	return nil
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
		case *resources.SecretVersion:
			configuration, err = a.getSecretVersionConfiguration(res, result)
			if err != nil {
				merr.Append(err)
				continue
			}
		}
		merr.Append(dag.CallConfigure(resource, configuration))
	}
	return merr.ErrOrNil()
}

func getS3BucketConfig(bucket *resources.S3Bucket, constructs *core.ConstructGraph) (resources.S3BucketConfigureParams, error) {
	staticUnits := make(map[string]*core.StaticUnit)
	for consRef := range bucket.ConstructsRef {
		cons := constructs.GetConstruct(core.ConstructId(consRef).ToRid())
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
