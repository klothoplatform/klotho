package engine

import (
	"errors"
	"fmt"
	"reflect"
	"sort"

	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/engine/classification"
	"go.uber.org/zap"
)

func (e *Engine) MakeResourcesOperational(graph *core.ResourceGraph) (map[core.ResourceId]bool, error) {
	zap.S().Debug("Engine Making resources operational and configuring resources")
	operationalResources := map[core.ResourceId]bool{}
	var joinedErr error
	for _, resource := range graph.ListResources() {
		err := callMakeOperational(graph, resource, e.Context.AppName, e.ClassificationDocument)
		if err != nil {
			if ore, ok := err.(*core.OperationalResourceError); ok {
				// If we get a OperationalResourceError let the engine try to reconcile it, and if that fails then mark the resource as non operational so we attempt to rerun on the next loop
				herr := e.handleOperationalResourceError(ore, graph)
				if herr != nil {
					err = errors.Join(err, herr)
				}
				joinedErr = errors.Join(joinedErr, err)
			}
			continue
		}

		err = graph.CallConfigure(resource, nil)
		if err != nil {
			joinedErr = errors.Join(joinedErr, err)
			continue
		}
		operationalResources[resource.Id()] = true
	}
	zap.S().Debug("Engine done making resources operational and configuring resources")
	return operationalResources, joinedErr
}

func callMakeOperational(rg *core.ResourceGraph, resource core.Resource, appName string, classifier classification.Classifier) error {
	method := reflect.ValueOf(resource).MethodByName("MakeOperational")
	if method.IsValid() {
		if rg.GetResource(resource.Id()) == nil {
			return fmt.Errorf("resource with id %s cannot be made operational since it does not exist in the ResourceGraph", resource.Id())
		}
		var callArgs []reflect.Value
		callArgs = append(callArgs, reflect.ValueOf(rg))
		callArgs = append(callArgs, reflect.ValueOf(appName))
		callArgs = append(callArgs, reflect.ValueOf(classifier))
		eval := method.Call(callArgs)
		if eval[0].IsNil() {
			return nil
		} else {
			err, ok := eval[0].Interface().(error)
			if !ok {
				return fmt.Errorf("return type should be an error")
			}
			return err
		}
	}
	return nil
}

// handleOperationalResourceError tries to determine how to fix OperatioanlResourceErrors by adding dependencies to the resource graph where needed.
// If the error cannot be fixed, it will return an error.
func (e *Engine) handleOperationalResourceError(err *core.OperationalResourceError, dag *core.ResourceGraph) error {
	resources := e.ListResources()

	var neededResource core.Resource
	for _, res := range resources {
		if e.ClassificationDocument.ResourceContainsClassifications(res, err.Needs) {
			_, found := e.KnowledgeBase.GetEdge(err.Resource, res)
			if !found {
				continue
			}
			if neededResource != nil {
				return fmt.Errorf("multiple resources found that can satisfy the operational resource error, %s", err.Error())
			}
			neededResource = res
		}
	}
	if neededResource == nil {
		return fmt.Errorf("no resources found that can satisfy the operational resource error, %s", err.Error())
	}
	var availableResources []core.Resource
	for _, res := range dag.ListResources() {
		if res.Id().Type == neededResource.Id().Type {
			availableResources = append(availableResources, res)
		}
	}
	if len(availableResources) == 0 {
		reflect.ValueOf(neededResource).Elem().FieldByName("Name").Set(reflect.ValueOf(fmt.Sprintf("%s-%s", neededResource.Id().Type, err.Resource.Id().Name)))
		dag.AddDependency(err.Resource, neededResource)
	} else {
		resourceIds := []string{}
		for _, res := range availableResources {
			resourceIds = append(resourceIds, res.Id().Name)
		}
		sort.Strings(resourceIds)
		for _, res := range availableResources {
			if res.Id().Name == resourceIds[0] {
				dag.AddDependency(err.Resource, res)
				break
			}
		}
	}
	return nil
}
