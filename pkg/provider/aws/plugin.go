package aws

import (
	"github.com/klothoplatform/klotho/pkg/core"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base"
	"github.com/klothoplatform/klotho/pkg/multierr"
	awsKnowledgebase "github.com/klothoplatform/klotho/pkg/provider/aws/knowledgebase"
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
			if _, ok := dep.Destination.(*core.Orm); ok {
				data.Constraint = knowledgebase.EdgeConstraint{
					NodeMustExist: []core.Resource{&resources.RdsProxy{}},
				}
			}
			for _, envVar := range construct.EnvironmentVariables {
				if envVar.Construct == dep.Destination {
					data.EnvironmentVariables = []core.EnvironmentVariable{envVar}
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
		}
		merr.Append(dag.CallConfigure(resource, configuration))
	}
	return merr.ErrOrNil()
}

func (a *AWS) Translate(result *core.ConstructGraph, dag *core.ResourceGraph) (links []core.CloudResourceLink, err error) {
	var merr multierr.Error
	merr.Append(a.ExpandConstructs(result, dag))
	merr.Append(a.CopyConstructEdgesToDag(result, dag))
	merr.Append(awsKnowledgebase.AwsKB.ExpandEdges(dag))
	merr.Append(a.configureResources(result, dag))
	merr.Append(awsKnowledgebase.AwsKB.ConfigureFromEdgeData(dag))
	return []core.CloudResourceLink{}, merr.ErrOrNil()
}
