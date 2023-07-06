package engine

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/engine/classification"
	"go.uber.org/zap"
)

func (e *Engine) MakeResourcesOperational(i int) {
	zap.S().Debug("Engine Making resources operational and configuring resources")
	for _, resource := range e.Context.EndState.ListResources() {
		err := callMakeOperational(e.Context.EndState, resource, e.Context.AppName, e.ClassificationDocument)
		if err != nil {
			if ore, ok := err.(*core.OperationalResourceError); ok {
				// If we get a OperationalResourceError let the engine try to reconcile it, and if that fails then mark the resource as non operational so we attempt to rerun on the next loop
				herr := e.handleOperationalResourceError(ore, e.Context.EndState)
				if herr != nil {
					err = errors.Join(err, herr)
				}
				e.Context.Errors[i] = append(e.Context.Errors[i], err)
				e.Context.OperationalResources[resource.Id()] = false
			}
			continue
		}
		e.Context.OperationalResources[resource.Id()] = true

		if !e.Context.ConfiguredResources[resource.Id()] {
			err := e.Context.EndState.CallConfigure(resource, nil)
			if err != nil {
				e.Context.Errors[i] = append(e.Context.Errors[i], err)
				continue
			}
			e.Context.ConfiguredResources[resource.Id()] = true
		}
	}
	zap.S().Debug("Engine done making resources operational and configuring resources")
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
