package engine

import (
	"fmt"
	"reflect"

	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/graph"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base"
	"go.uber.org/zap"
)

func (e *Engine) expandEdges(i int) {
	zap.S().Debug("Engine Expanding Edges")
	for _, dep := range e.Context.EndState.ListDependencies() {
		src := dep.Source.Id()
		dst := dep.Destination.Id()
		if e.Context.ExpandedEdges[src] == nil {
			e.Context.ExpandedEdges[src] = make(map[core.ResourceId]bool)
		}
		// If we know that the edge has a direct connection but is flipped due to data flow, immediately use that edge
		if det, _ := e.KnowledgeBase.GetEdge(dep.Source, dep.Destination); det.ReverseDirection {
			dep = graph.Edge[core.Resource]{Source: dep.Destination, Destination: dep.Source}
		}
		if !e.Context.ExpandedEdges[src][dst] {
			edgeData, err := getEdgeData(dep)
			if err != nil {
				zap.S().Warnf("got error when getting edge data for edge %s -> %s, err: %s", dep.Source.Id(), dep.Destination.Id(), err.Error())
				e.Context.Errors[i] = append(e.Context.Errors[i], err)
				continue
			}
			path, err := e.determineCorrectPath(dep, edgeData)
			if err != nil {
				zap.S().Warnf("got error when determining correct path for edge %s -> %s, err: %s", dep.Source.Id(), dep.Destination.Id(), err.Error())
				e.Context.Errors[i] = append(e.Context.Errors[i], err)
				continue
			}
			err = e.KnowledgeBase.ExpandEdge(&dep, e.Context.EndState, path, edgeData)
			if err != nil {
				zap.S().Warnf("got error when expanding edge %s -> %s, err: %s", dep.Source.Id(), dep.Destination.Id(), err.Error())
				e.Context.Errors[i] = append(e.Context.Errors[i], err)
				continue
			}
		}
		e.Context.ExpandedEdges[src][dst] = true
	}
	zap.S().Debug("Engine Done Expanding Edges")
}

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

func (e *Engine) determineCorrectPath(dep graph.Edge[core.Resource], edgeData knowledgebase.EdgeData) (knowledgebase.Path, error) {
	paths := e.KnowledgeBase.FindPaths(dep.Source, dep.Destination, edgeData.Constraint)
	fmt.Println(paths)
	var validPaths []knowledgebase.Path
	for _, p := range paths {
		if len(p) == 0 {
			continue
		}
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
	fmt.Println(validPath)
	return validPath, nil
}

func (e *Engine) containsUnneccessaryHopsInPath(dep graph.Edge[core.Resource], p knowledgebase.Path) bool {
	for _, edge := range p {
		destType := reflect.TypeOf(dep.Destination)
		if e.ClassificationDocument.GetFunctionality(dep.Destination) != core.Unknown {
			if e.ClassificationDocument.GetFunctionality(dep.Destination) == e.ClassificationDocument.GetFunctionality(reflect.New(edge.Destination).Elem().Interface().(core.Resource)) && edge.Destination != destType {
				zap.S().Debugf("Found unneccessary hop in path %s, destination %s has same functionality as edge's destination %s", p, dep.Destination.Id(), edge.Destination)
				return true
			}
			if e.ClassificationDocument.GetFunctionality(dep.Destination) == e.ClassificationDocument.GetFunctionality(reflect.New(edge.Source).Elem().Interface().(core.Resource)) && edge.Source != destType {
				zap.S().Debugf("Found unneccessary hop in path %s, destination %s has same functionality as edge's destination %s", p, dep.Destination.Id(), edge.Source)
				return true
			}
		}
	}
	return false
}

// FindShortestPath determines the shortest path to get from the dependency's source node to destination node, using the knowledgebase of edges
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
