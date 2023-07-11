package engine

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/klothoplatform/klotho/pkg/collectionutil"
	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/graph"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base"
	"go.uber.org/zap"
)

func (e *Engine) expandEdges(graph *core.ResourceGraph) error {
	zap.S().Debug("Engine Expanding Edges")
	var joinedErr error
	for _, dep := range graph.ListDependencies() {

		edgeData, err := getEdgeData(dep)
		if err != nil {
			zap.S().Warnf("got error when getting edge data for edge %s -> %s, err: %s", dep.Source.Id(), dep.Destination.Id(), err.Error())
			joinedErr = errors.Join(joinedErr, err)
			continue
		}
		path, err := e.determineCorrectPath(dep, edgeData)
		if err != nil {
			zap.S().Warnf("got error when determining correct path for edge %s -> %s, err: %s", dep.Source.Id(), dep.Destination.Id(), err.Error())
			joinedErr = errors.Join(joinedErr, err)
			continue
		}
		if path == nil {
			if edgeData.Attributes != nil {
				joinedErr = errors.Join(joinedErr, fmt.Errorf("no valid path found for edge %s -> %s with edge attributes %s", dep.Source.Id(), dep.Destination.Id(), edgeData.Attributes))
			} else {
				joinedErr = errors.Join(joinedErr, fmt.Errorf("no valid path found for edge %s -> %s", dep.Source.Id(), dep.Destination.Id()))
			}
			continue
		}
		err = e.KnowledgeBase.ExpandEdge(&dep, graph, path, edgeData)
		if err != nil {
			zap.S().Warnf("got error when expanding edge %s -> %s, err: %s", dep.Source.Id(), dep.Destination.Id(), err.Error())
			joinedErr = errors.Join(joinedErr, err)
			continue
		}
	}
	zap.S().Debug("Engine Done Expanding Edges")
	return joinedErr
}

// getEdgeData retrieves the edge data from the edge in the resource graph to use during expansion
func getEdgeData(dep graph.Edge[core.Resource]) (knowledgebase.EdgeData, error) {
	// We want to retrieve the edge data from the edge in the resource graph to use during expansion
	edgeData := knowledgebase.EdgeData{}
	data, ok := dep.Properties.Data.(knowledgebase.EdgeData)
	if !ok && dep.Properties.Data != nil {
		return edgeData, fmt.Errorf("edge properties for edge %s -> %s, do not satisfy edge data format during expansion", dep.Source.Id(), dep.Destination.Id())
	} else if dep.Properties.Data != nil {
		edgeData = data
	}
	// We attach the dependencies source and destination nodes for context during expansion
	edgeData.Source = dep.Source
	edgeData.Destination = dep.Destination
	// Find all possible paths given the initial source and destination node
	return edgeData, nil
}

// determineCorrectPath determines the correct path to take to get from the dependency's source node to destination node, using the knowledgebase of edges
// It first finds all possible paths given the initial source and destination node. It then filters out any paths that do not satisfy the constraints of the edge
// It then filters out any paths that contain unnecessary hops to get to the destination
func (e *Engine) determineCorrectPath(dep graph.Edge[core.Resource], edgeData knowledgebase.EdgeData) (knowledgebase.Path, error) {
	paths := e.KnowledgeBase.FindPaths(dep.Source, dep.Destination, edgeData.Constraint)
	var validPaths []knowledgebase.Path
	var satisfyAttributeData []knowledgebase.Path
	for _, p := range paths {
		satisfies := true
		for _, edge := range p {
			for k := range edgeData.Attributes {
				// If its a direct edge we need to make sure the source contains the attributes, otherwise ignore the source of the dependency
				if edge.Source != reflect.TypeOf(dep.Source) || len(p) == 1 {
					classification := e.ClassificationDocument.GetClassification(reflect.New(edge.Source.Elem()).Interface().(core.Resource))
					if !collectionutil.Contains(classification.Is, k) {
						satisfies = false
						break
					}
				}
				// If its a direct edge we need to make sure the destination contains the attributes, otherwise ignore the destination of the dependency
				if edge.Destination != reflect.TypeOf(dep.Destination) || len(p) == 1 {
					classification := e.ClassificationDocument.GetClassification(reflect.New(edge.Destination.Elem()).Interface().(core.Resource))
					if !collectionutil.Contains(classification.Is, k) {
						satisfies = false
						break
					}
				}
			}
			if !satisfies {
				break
			}
		}
		if satisfies {
			satisfyAttributeData = append(satisfyAttributeData, p)
		}
	}
	for _, p := range satisfyAttributeData {
		// Ensure we arent taking unnecessary hops to get to the destination
		if !e.containsUnneccessaryHopsInPath(dep, p) {
			validPaths = append(validPaths, p)
		}
	}
	validPath, err := findShortestPath(validPaths)
	if err != nil {
		return nil, err
	}
	zap.S().Debugf("Found valid path %s", validPath)
	return validPath, nil
}

// containsUnneccessaryHopsInPath determines if the path contains any unnecessary hops to get to the destination
//
// We check if the source and destination of the dependency have a functionality. If they do, we check if the functionality of the source or destination
// is the same as the functionality of the source or destination of the edge in the path. If it is then we ensure that the source or destination of the edge
// in the path is not the same as the source or destination of the dependency. If it is then we know that the edge in the path is an unnecessary hop to get to the destination
func (e *Engine) containsUnneccessaryHopsInPath(dep graph.Edge[core.Resource], p knowledgebase.Path) bool {
	for _, edge := range p {
		destType := reflect.TypeOf(dep.Destination)
		srcType := reflect.TypeOf(dep.Source)
		if e.ClassificationDocument.GetFunctionality(dep.Destination) != core.Unknown {
			if e.ClassificationDocument.GetFunctionality(dep.Destination) == e.ClassificationDocument.GetFunctionality(reflect.New(edge.Destination).Elem().Interface().(core.Resource)) && edge.Destination != destType && edge.Destination != srcType {
				return true
			}
			if e.ClassificationDocument.GetFunctionality(dep.Destination) == e.ClassificationDocument.GetFunctionality(reflect.New(edge.Source).Elem().Interface().(core.Resource)) && edge.Source != destType && edge.Source != srcType {
				return true
			}
		}
		if e.ClassificationDocument.GetFunctionality(dep.Source) != core.Unknown {
			if e.ClassificationDocument.GetFunctionality(dep.Source) == e.ClassificationDocument.GetFunctionality(reflect.New(edge.Destination).Elem().Interface().(core.Resource)) && edge.Destination != srcType && edge.Destination != destType {
				return true
			}
			if e.ClassificationDocument.GetFunctionality(dep.Source) == e.ClassificationDocument.GetFunctionality(reflect.New(edge.Source).Elem().Interface().(core.Resource)) && edge.Source != srcType && edge.Source != destType {
				return true
			}
		}
	}
	return false
}

// findShortestPath determines the shortest path to get from the dependency's source node to destination node, using the knowledgebase of edges
func findShortestPath(paths []knowledgebase.Path) (knowledgebase.Path, error) {
	var validPath []knowledgebase.Edge

	var sameLengthPaths []knowledgebase.Path
	// Get the shortest route that satisfied constraints
	for _, path := range paths {
		if len(validPath) == 0 {
			validPath = path
		} else if len(path) < len(validPath) {
			validPath = path
			sameLengthPaths = []knowledgebase.Path{}
		} else if len(path) == len(validPath) {
			sameLengthPaths = append(sameLengthPaths, path, validPath)
		}
	}
	if len(sameLengthPaths) > 0 {
		return nil, fmt.Errorf("found multiple paths which are the same length. \n Paths: %s", sameLengthPaths)
	}
	return validPath, nil
}
