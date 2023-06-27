package engine

import (
	"github.com/klothoplatform/klotho/pkg/collectionutil"
	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/provider/aws/resources"
	"go.uber.org/zap"
)

func (e *Engine) GetDataFlowDag() *core.ResourceGraph {
	dataFlowDag := core.NewResourceGraph()
	typesWeCareAbout := []string{
		resources.LAMBDA_FUNCTION_TYPE,
		resources.EC2_INSTANCE_TYPE,
		resources.ECS_SERVICE_TYPE,
		resources.API_GATEWAY_REST_TYPE,
		resources.S3_BUCKET_TYPE,
		resources.DYNAMODB_TABLE_TYPE,
		resources.RDS_INSTANCE_TYPE,
		resources.ELASTICACHE_CLUSTER_TYPE,
		resources.SECRET_TYPE,
		resources.RDS_PROXY_TYPE,
		resources.LOAD_BALANCER_TYPE,
		resources.CLOUDFRONT_DISTRIBUTION_TYPE,
		resources.ROUTE_53_HOSTED_ZONE_TYPE,
	}

	parentResourceTypes := []string{
		resources.VPC_TYPE,
		resources.ECS_CLUSTER_TYPE,
		resources.EKS_CLUSTER_TYPE,
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
		for _, dst := range dataFlowDag.ListResources() {
			if src == dst {
				continue
			}
			paths := e.KnowledgeBase.FindPathsInGraph(src, dst, e.Context.EndState)
			if len(paths) > 0 {
				addedDep := false
				for _, path := range paths {
					for _, edge := range path {
						if collectionutil.Contains(typesWeCareAbout, edge.Source.Id().Type) && edge.Source != src && edge.Source != dst {
							dataFlowDag.AddDependency(src, edge.Source)
							addedDep = true
							break
						}
						if collectionutil.Contains(typesWeCareAbout, edge.Destination.Id().Type) && edge.Destination != src && edge.Destination != dst {
							dataFlowDag.AddDependency(src, edge.Destination)
							addedDep = true
							break
						}
					}
					if addedDep {
						break
					}
				}

				// Add a summarized edge if there are no relevant intermediate resources
				// or a child -> parent edge if the destination is a parent type.
				if !addedDep || collectionutil.Contains(parentResourceTypes, dst.Id().Type) {
					dataFlowDag.AddDependency(src, dst)
				}
			}
		}
	}

	// Configure Parent/Child relationships and remove child -> parent edges.
	for _, dep := range dataFlowDag.ListDependencies() {
		if collectionutil.Contains(parentResourceTypes, dep.Destination.Id().Type) {
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
	return dataFlowDag
}
