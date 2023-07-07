package engine

import (
	"github.com/klothoplatform/klotho/pkg/collectionutil"
	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/filter"
	"github.com/klothoplatform/klotho/pkg/graph"
	awsResources "github.com/klothoplatform/klotho/pkg/provider/aws/resources"
	k8sResources "github.com/klothoplatform/klotho/pkg/provider/kubernetes/resources"
	"go.uber.org/zap"
	"sort"
)

type nodeSettings struct {
	// AllowIncoming determines whether the node's incoming edges should be added to the dataflow DAG
	AllowIncoming bool
	// AllowOutgoing determines whether the node's outgoing edges should be added to the dataflow DAG
	AllowOutgoing bool
}

// resourcePostFilter is a function that determines whether a resource should remain in the final dataflow DAG
// based on its own properties and the state of the dataflow DAG.
type resourcePostFilter func(resource core.Resource, dag *core.ResourceGraph) bool

func (e *Engine) GetDataFlowDag() *core.ResourceGraph {
	dataFlowDag := core.NewResourceGraph()
	typesWeCareAbout := []string{
		awsResources.LAMBDA_FUNCTION_TYPE,
		awsResources.EC2_INSTANCE_TYPE,
		awsResources.ECS_SERVICE_TYPE,
		awsResources.API_GATEWAY_REST_TYPE,
		awsResources.S3_BUCKET_TYPE,
		awsResources.DYNAMODB_TABLE_TYPE,
		awsResources.RDS_INSTANCE_TYPE,
		awsResources.ELASTICACHE_CLUSTER_TYPE,
		awsResources.SECRET_TYPE,
		awsResources.RDS_PROXY_TYPE,
		awsResources.LOAD_BALANCER_TYPE,
		awsResources.CLOUDFRONT_DISTRIBUTION_TYPE,
		awsResources.ROUTE_53_HOSTED_ZONE_TYPE,
		k8sResources.DEPLOYMENT_TYPE,
		k8sResources.POD_TYPE,
		k8sResources.HELM_CHART_TYPE,
	}

	parentResources := map[string]nodeSettings{
		awsResources.VPC_TYPE:         {},
		awsResources.ECS_CLUSTER_TYPE: {},
		awsResources.EKS_CLUSTER_TYPE: {},
	}

	var parentResourceTypes []string
	for parentResourceType := range parentResources {
		parentResourceTypes = append(parentResourceTypes, parentResourceType)
	}

	resourcePostFilters := []resourcePostFilter{
		helmChartFilter,
	}

	// Add relevant resources to the dataflow DAG
	for _, resource := range e.Context.EndState.ListResources() {
		if collectionutil.Contains(typesWeCareAbout, resource.Id().Type) || collectionutil.Contains(parentResourceTypes, resource.Id().Type) {
			dataFlowDag.AddResource(resource)
		}
	}

	// Add summarized edges between types we care about to the dataflow DAG.
	// Only irrelevant nodes in a path of edges between the source and destination will be summarized.
	for _, src := range dataFlowDag.ListResources() {
		var srcParents []core.Resource
		hasPathWithoutOthers := false
		for _, dst := range dataFlowDag.ListResources() {
			if src == dst {
				continue
			}
			paths := e.KnowledgeBase.FindPathsInGraph(src, dst, e.Context.EndState)
			if len(paths) > 0 {
				sort.SliceStable(paths, func(i, j int) bool {
					return len(paths[i]) < len(paths[j])
				})
				addedDep := false
				for _, path := range paths {
					pathHasDep := false
					for _, edge := range path {
						if collectionutil.Contains(typesWeCareAbout, edge.Source.Id().Type) && edge.Source.Id() != src.Id() && edge.Source.Id() != dst.Id() {
							dataFlowDag.AddDependency(src, edge.Source)
							addedDep = true
							pathHasDep = true
							break
						}
						if collectionutil.Contains(typesWeCareAbout, edge.Destination.Id().Type) && edge.Destination.Id() != src.Id() && edge.Destination.Id() != dst.Id() {
							dataFlowDag.AddDependency(src, edge.Destination)
							addedDep = true
							pathHasDep = true
							break
						}
					}
					if !pathHasDep {
						hasPathWithoutOthers = true
					}
					if addedDep {
						break
					}
				}
				// Add a summarized edge if there are no relevant intermediate resources
				// or a child -> parent edge if the destination is a parent type.
				if collectionutil.Contains(parentResourceTypes, dst.Id().Type) && hasPathWithoutOthers {
					srcParents = append(srcParents, dst)
				} else if !addedDep {
					dataFlowDag.AddDependency(src, dst)
				}
			}
		}
		var parentPaths [][]graph.Edge[core.Resource]
		for _, p := range srcParents {
			parentPaths = append(parentPaths, e.KnowledgeBase.FindPathsInGraph(src, p, e.Context.EndState)...)
		}
		sort.SliceStable(parentPaths, func(i, j int) bool {
			return len(parentPaths[i]) < len(parentPaths[j])
		})

		// TODO: look into why FindPathsInGraph is returning unrelated paths. this filter is a workaround.
		parentPaths = filter.NewSimpleFilter[[]graph.Edge[core.Resource]](func(path []graph.Edge[core.Resource]) bool {
			return collectionutil.Contains(parentResourceTypes, path[len(path)-1].Destination.Id().Type)
		}).Apply(parentPaths...)

		if len(parentPaths) > 0 {
			closestParent := parentPaths[0][len(parentPaths[0])-1].Destination
			dataFlowDag.AddDependency(src, closestParent)
		}
	}

	for _, res := range dataFlowDag.ListResources() {
		for _, filter := range resourcePostFilters {
			if !filter(res, dataFlowDag) {
				err := dataFlowDag.RemoveResourceAndEdges(res)
				if err != nil {
					zap.S().Debugf("Error removing resource %s", err.Error())
					continue
				}
				break
			}
		}
	}

	// Configure Parent/Child relationships and remove child -> parent edges.
	for _, dep := range dataFlowDag.ListDependencies() {
		if collectionutil.Contains(parentResourceTypes, dep.Destination.Id().Type) {
			if core.IsResourceChild(dep.Source, dep.Destination) {
				err := dataFlowDag.RemoveDependency(dep.Source.Id(), dep.Destination.Id())
				if err != nil {
					zap.S().Debugf("Error removing dependency %s", err.Error())
					continue
				}
				dataFlowDag.AddResourceWithProperties(dep.Source, map[string]string{
					"parent": dep.Destination.Id().String(),
				})
			}
		}
	}
	filterParentEdges(dataFlowDag, parentResources)
	return dataFlowDag
}

func helmChartFilter(resource core.Resource, dag *core.ResourceGraph) bool {
	chart, ok := resource.(*k8sResources.HelmChart)
	if !ok {
		return true
	}
	return !chart.IsInternal
}

// filterParentEdges removes edges between resources categorized as parent resources and other resources depending on their AllowIncoming and AllowOutgoing settings.
func filterParentEdges(dataFlowDag *core.ResourceGraph, parentResources map[string]nodeSettings) {
	for _, dep := range dataFlowDag.ListDependencies() {
		if settings, ok := parentResources[dep.Destination.Id().Type]; ok {
			if !settings.AllowIncoming {
				err := dataFlowDag.RemoveDependency(dep.Source.Id(), dep.Destination.Id())
				if err != nil {
					zap.S().Debugf("Error removing dependency %s", err.Error())
					continue
				}
			}
		}
		if settings, ok := parentResources[dep.Source.Id().Type]; ok {
			if !settings.AllowOutgoing {
				err := dataFlowDag.RemoveDependency(dep.Source.Id(), dep.Destination.Id())
				if err != nil {
					zap.S().Debugf("Error removing dependency %s", err.Error())
					continue
				}
			}
		}
	}
}
