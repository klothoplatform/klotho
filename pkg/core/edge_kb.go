package core

import (
	"fmt"
	"reflect"

	"github.com/klothoplatform/klotho/pkg/multierr"
	"go.uber.org/zap"
)

type (
	Edge struct {
		From reflect.Type
		To   reflect.Type
	}

	EdgeDetails struct {
		ExpansionFunc EdgeExpander
		Configure     EdgeConfigurer
	}

	EdgeKB map[Edge]EdgeDetails

	EdgeExpander   func(from, to Resource, dag *ResourceGraph, data EdgeData) error
	EdgeConfigurer func(from, to Resource, data EdgeData) error

	EdgeConstraint struct {
		NodeMustExist    Resource
		NodeMustNotExist Resource
	}

	EdgeData struct {
		AppName             string
		EnvironmentVariable EnvironmentVariable
		Constraint          EdgeConstraint
	}
)

func GetEdgeDetails(kb EdgeKB, source reflect.Type, target reflect.Type) EdgeDetails {
	return kb[Edge{From: source, To: target}]
}

func GetEdgesWithSource(kb EdgeKB, source reflect.Type) []Edge {
	result := []Edge{}
	for edge, _ := range kb {
		if edge.From == source {
			result = append(result, edge)
		}
	}
	return result
}

func GetEdgesWithTarget(kb EdgeKB, target reflect.Type) []Edge {
	result := []Edge{}
	for edge, _ := range kb {
		if edge.To == target {
			result = append(result, edge)
		}
	}
	return result
}

func FindPaths(kb EdgeKB, source reflect.Type, dest reflect.Type) [][]reflect.Type {
	zap.S().Debugf("Finding Paths from %s -> %s", source.String(), dest.String())
	result := [][]reflect.Type{}
	visitedEdges := map[reflect.Type]bool{}
	stack := []reflect.Type{}
	findPaths(kb, source, dest, stack, visitedEdges, &result)
	return result
}

func findPaths(kb EdgeKB, source reflect.Type, dest reflect.Type, stack []reflect.Type, visited map[reflect.Type]bool, result *[][]reflect.Type) {
	visited[source] = true
	stack = append(stack, source)
	if source == dest {
		*result = append(*result, stack)
	} else {
		for _, e := range GetEdgesWithSource(kb, source) {
			if reflect.TypeOf(e.From) == reflect.TypeOf(source) && !visited[e.To] {
				findPaths(kb, e.To, dest, stack, visited, result)
			}
		}
	}
	delete(visited, source)
	stack = stack[:len(stack)-1]
}

func ExpandEdges(kb EdgeKB, dag *ResourceGraph) (err error) {
	zap.S().Debug("Expanding Edges")
	var merr multierr.Error
	for _, dep := range dag.ListDependencies() {
		zap.S().Debug("Expanding Edge for %s -> %s", dep.Source.Id().String(), dep.Destination.Id().String())
		edgeData := EdgeData{}
		data, ok := dep.Properties.Data.(EdgeData)
		if !ok && dep.Properties.Data != nil {
			merr.Append(fmt.Errorf("edge properties for edge %s -> %s, do not satisfy edge data format", dep.Source.Id().String(), dep.Destination.Id().String()))
		} else if dep.Properties.Data != nil {
			edgeData = data
		}
		paths := FindPaths(kb, reflect.TypeOf(dep.Source), reflect.TypeOf(dep.Destination))
		validPaths := [][]reflect.Type{}
		for _, path := range paths {
			if edgeData.Constraint.NodeMustExist != nil {
				nodeFound := false
				for _, res := range path {
					if res == reflect.TypeOf(edgeData.Constraint.NodeMustExist) {
						nodeFound = true
					}
				}
				if !nodeFound {
					continue
				}
			}
			if edgeData.Constraint.NodeMustNotExist != nil {
				nodeFound := false
				for _, res := range path {
					if res == reflect.TypeOf(edgeData.Constraint.NodeMustNotExist) {
						nodeFound = true
					}
				}
				if nodeFound {
					continue
				}
			}
			validPaths = append(validPaths, path)
		}
		if len(validPaths) > 1 {
			merr.Append(fmt.Errorf("found multiple paths which satisfy constraints for edge %s -> %s. \n Paths: %s", dep.Source.Id().String(), dep.Destination.Id().String(), validPaths))
		} else {
			validPath := validPaths[0]
			if len(validPath) > 2 {
				zap.S().Debugf("Removing dependency from %s -> %s", dep.Source.Id().String(), dep.Destination.Id().String())
				err := dag.RemoveDependency(dep.Source.Id().String(), dep.Destination.Id().String())
				if err != nil {
					merr.Append(err)
					continue
				}
			}
			for i := 1; i < len(validPath); i++ {
				from := validPath[i-1]
				to := validPath[i]
				edgeDetail := GetEdgeDetails(kb, from, to)
				fromNode := reflect.New(from.Elem()).Interface().(Resource)
				if from == reflect.TypeOf(dep.Source) {
					fromNode = dep.Source
				}
				toNode := reflect.New(to.Elem()).Interface().(Resource)
				if to == reflect.TypeOf(dep.Destination) {
					toNode = dep.Destination
				}
				if edgeDetail.ExpansionFunc != nil {
					err := edgeDetail.ExpansionFunc(fromNode, toNode, dag, edgeData)
					merr.Append(err)
				}
			}
		}

	}
	return merr.ErrOrNil()
}

func ConfigureFromEdgeData(kb EdgeKB, dag *ResourceGraph) (err error) {
	zap.S().Debug("Configuring Edges")
	var merr multierr.Error
	for _, dep := range dag.ListDependencies() {
		zap.S().Debugf("Configuring Edge for %s -> %s", dep.Source.Id().String(), dep.Destination.Id().String())
		to := reflect.TypeOf(dep.Source)
		from := reflect.TypeOf(dep.Destination)
		edgeData := EdgeData{}
		data, ok := dep.Properties.Data.(EdgeData)
		if !ok && dep.Properties.Data != nil {
			merr.Append(fmt.Errorf("edge properties for edge %s -> %s, do not satisfy edge data format", dep.Source.Id().String(), dep.Destination.Id().String()))
		} else if dep.Properties.Data != nil {
			edgeData = data
		}
		edgeDetail := GetEdgeDetails(kb, to, from)
		if edgeDetail.Configure != nil {
			err := edgeDetail.Configure(dep.Source, dep.Destination, edgeData)
			merr.Append(err)
		}
	}
	return merr.ErrOrNil()
}
