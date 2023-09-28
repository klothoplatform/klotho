package path_selection

import (
	"fmt"

	"github.com/dominikbraun/graph"
	construct "github.com/klothoplatform/klotho/pkg/construct2"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base2"
	"go.uber.org/zap"
)

// ExpandEdges performs calculations to determine the proper path to be inserted into the ResourceGraph.
//
// The workflow of the edge expansion is as follows:
//   - Find shortest path given the constraints on the edge
//   - Iterate through each edge in path creating the resource if necessary
func (ctx PathSelectionContext) ExpandEdge(dep graph.Edge[*construct.Resource], validPath Path, edgeData EdgeData) ([]graph.Edge[*construct.Resource], error) {

	var result []graph.Edge[*construct.Resource]
	// It does not matter what order we go in as each edge should be expanded independently. They can still reuse resources since the create methods should be idempotent if resources are the same.
	zap.S().Debugf("Expanding Edge for %s -> %s", dep.Source.ID, dep.Target.ID)

	name := fmt.Sprintf("%s_%s", dep.Source.ID.Name, dep.Target.ID.Name)

	var previousNode *construct.Resource
	var edgeTemplate *knowledgebase.EdgeTemplate

PATH:
	for i, node := range validPath.Nodes {
		destNode := &construct.Resource{ID: node}
		if i == 0 {
			previousNode = dep.Source
			continue
		} else if i == len(validPath.Nodes)-1 {
			destNode = dep.Target
		} else {
			// Create a new interface of the destination nodes type if it does not exist
			destNode.ID.Name = fmt.Sprintf("%s_%s", destNode.ID.Type, name)
			// Determine if the destination node is the same type as what is specified in the constraints as must exist
			for _, mustExistRes := range edgeData.Constraint.NodeMustExist {
				if mustExistRes.ID.Type == destNode.ID.Type && mustExistRes.ID.Provider == destNode.ID.Provider {
					destNode = &mustExistRes
				}
			}
		}
		edgeTemplate = ctx.KB.GetEdgeTemplate(previousNode.ID, node)

		// If the edge specifies that it can reuse upstream or downstream resources, we want to find the first resource which satisfies the reuse criteria and add that as the dependency.
		// If there is no resource that satisfies the reuse criteria, we want to add the original direct dependency
		switch edgeTemplate.Reuse {
		case knowledgebase.ReuseUpstream:
			DownstreamResources, err := ctx.Graph.Downstream(dep.Source, 3)
			if err != nil {
				return nil, err
			}
			for _, res := range DownstreamResources {
				if destNode.ID.QualifiedTypeName() == res.ID.QualifiedTypeName() {
					result = append(result, graph.Edge[*construct.Resource]{
						Source: res,
						Target: dep.Target,
					})
					break PATH
				}
			}
		case knowledgebase.ReuseDownstream:
			upstreamResources, err := ctx.Graph.Upstream(dep.Target, 3)
			if err != nil {
				return nil, err
			}
			for _, res := range upstreamResources {
				fmt.Println(previousNode.ID, res.ID.QualifiedTypeName())
				if previousNode.ID.QualifiedTypeName() == res.ID.QualifiedTypeName() {
					result = []graph.Edge[*construct.Resource]{
						{
							Source: dep.Source,
							Target: res,
						},
					}
					break PATH
				}
			}
		}
		result = append(result, graph.Edge[*construct.Resource]{
			Source: previousNode,
			Target: destNode,
		})
		previousNode = destNode
	}

	return result, nil
}
