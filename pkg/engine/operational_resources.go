package engine

import (
	"errors"
	"fmt"
	"reflect"
	"sort"

	"github.com/klothoplatform/klotho/pkg/collectionutil"
	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/engine/classification"
	"go.uber.org/zap"
)

func (e *Engine) MakeResourcesOperational(graph *core.ResourceGraph) (map[core.ResourceId]bool, error) {
	zap.S().Debug("Engine Making resources operational and configuring resources")
	operationalResources := map[core.ResourceId]bool{}
	var joinedErr error
	resources, err := graph.ReverseTopologicalSort()
	if err != nil {
		return nil, err
	}
	for _, resource := range resources {
		template := e.ResourceTemplates[fmt.Sprintf("%s:%s", resource.Id().Provider, resource.Id().Type)]
		if template != nil {
			err := e.TemplateMakeOperational(graph, resource, *template)
			if err != nil {
				joinedErr = errors.Join(joinedErr, err)
				continue
			}
			err = TemplateConfigure(resource, *template)
			if err != nil {
				joinedErr = errors.Join(joinedErr, err)
				continue
			}
		} else {
			err := callMakeOperational(graph, resource, e.Context.AppName, e.ClassificationDocument)
			if err != nil {
				if ore, ok := err.(*core.OperationalResourceError); ok {
					// If we get a OperationalResourceError let the engine try to reconcile it, and if that fails then mark the resource as non operational so we attempt to rerun on the next loop
					herr := e.handleDownstreamOperationalResourceError(ore, graph)
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

func (e *Engine) TemplateMakeOperational(dag *core.ResourceGraph, resource core.Resource, template core.ResourceTemplate) error {
	var joinedErr error
	for _, drule := range template.Rules.Downstream {
		errs := e.handleDownstreamOperationalRule(resource, drule, dag, nil)
		for _, err := range errs {
			if err != nil {
				if ore, ok := err.(*core.OperationalResourceError); ok {
					err := e.handleDownstreamOperationalResourceError(ore, dag)
					if err != nil {
						joinedErr = errors.Join(joinedErr, err)
					}
					continue
				}
				joinedErr = errors.Join(joinedErr, err)
			}
		}
	}
	return joinedErr
}

func (e *Engine) handleDownstreamOperationalRule(resource core.Resource, rule core.OperationalRule, dag *core.ResourceGraph, downstreamParent core.Resource) []error {
	downstreamResourcesOfType := []core.Resource{}
	if rule.ResourceTypes != nil && rule.Classifications != nil {
		return []error{fmt.Errorf("downstream rule cannot have both a resource type and classifications defined %s", rule.String())}
	} else if rule.ResourceTypes != nil {
		for _, down := range dag.GetAllDownstreamResources(resource) {
			if collectionutil.Contains(rule.ResourceTypes, down.Id().Type) && down.Id().Provider == resource.Id().Provider {
				downstreamResourcesOfType = append(downstreamResourcesOfType, down)
			}
		}
	} else if rule.Classifications != nil {
		for _, down := range dag.GetAllDownstreamResources(resource) {
			if e.ClassificationDocument.ResourceContainsClassifications(down, rule.Classifications) {
				downstreamResourcesOfType = append(downstreamResourcesOfType, down)
			}
		}
	} else {
		return []error{fmt.Errorf("downstream rule must have either a resource type or classifications defined %s", rule.String())}
	}

	switch rule.Enforcement {
	case core.ExactlyOne:
		var res core.Resource
		var ore *core.OperationalResourceError
		if len(downstreamResourcesOfType) > 1 {
			return []error{fmt.Errorf("downstream rule with enforcement only_one has more than one resource of types %s", rule.ResourceTypes)}
		} else if len(downstreamResourcesOfType) == 0 {
			switch rule.UnsatisfiedAction.Operation {
			case core.CreateUnsatisfiedResource:
				var needs []string
				if rule.UnsatisfiedAction.DefaultType != "" {
					needs = []string{rule.UnsatisfiedAction.DefaultType}
				} else {
					if rule.Classifications != nil {
						needs = rule.Classifications
					} else {
						needs = []string{rule.ResourceTypes[0]}
					}
				}
				ore = &core.OperationalResourceError{
					Resource:   resource,
					Parent:     downstreamParent,
					Count:      1,
					Needs:      needs,
					MustCreate: rule.UnsatisfiedAction.Unique,
					Cause:      fmt.Errorf("downstream rule with enforcement any has less than the required number of resources of type %s, %d", rule.ResourceTypes, len(downstreamResourcesOfType)),
				}
			case core.ErrorUnsatisfiedResource:
				return []error{fmt.Errorf("downstream rule with enforcement any has less than the required number of resources of type %s, %d", rule.ResourceTypes, len(downstreamResourcesOfType))}
			}
		} else {
			res = downstreamResourcesOfType[0]
			dag.AddDependency(resource, res)
			setField(resource, rule, res)
			if downstreamParent != nil {
				dag.AddDependency(res, downstreamParent)
			}
		}
		var subRuleErrors []error
		for _, subRule := range rule.Rules {
			err := e.handleDownstreamOperationalRule(resource, subRule, dag, nil)
			if err != nil {
				subRuleErrors = append(subRuleErrors, err...)
			}
		}
		if subRuleErrors != nil {
			return subRuleErrors
		}
		if ore != nil {
			return []error{ore}
		}
		if res == nil {
			return []error{fmt.Errorf("no resources found that can satisfy the operational resource rule %s, for %s", rule.String(), resource.Id())}
		}
		if rule.RemoveDirectDependency {
			err := dag.RemoveDependency(resource.Id(), res.Id())
			if err != nil {
				return []error{err}
			}
		}
	case core.Conditional:
		if len(downstreamResourcesOfType) > 1 {
			return []error{fmt.Errorf("downstream rule with enforcement if_one has more than one resource of types %s", rule.ResourceTypes)}
		}
		if len(downstreamResourcesOfType) == 1 {
			setField(resource, rule, downstreamResourcesOfType[0])
			if rule.RemoveDirectDependency {
				err := dag.RemoveDependency(resource.Id(), downstreamResourcesOfType[0].Id())
				if err != nil {
					return []error{err}
				}
			}
			var subRuleErrors []error
			for _, subRule := range rule.Rules {
				err := e.handleDownstreamOperationalRule(resource, subRule, dag, downstreamResourcesOfType[0])
				if err != nil {
					subRuleErrors = append(subRuleErrors, err...)
				}
			}
			if subRuleErrors != nil {
				return subRuleErrors
			}
		}
	case core.AnyAvailable:
		var ore *core.OperationalResourceError
		for _, res := range downstreamResourcesOfType {
			setField(resource, rule, res)
		}
		if rule.NumNeeded > len(downstreamResourcesOfType) {
			switch rule.UnsatisfiedAction.Operation {
			case core.CreateUnsatisfiedResource:
				var needs []string
				if len(downstreamResourcesOfType) > 0 {
					var existingTypes []string
					for _, res := range downstreamResourcesOfType {
						existingTypes = append(existingTypes, res.Id().Type)
					}
					if len(existingTypes) == 1 {
						needs = existingTypes
					}
				} else if rule.UnsatisfiedAction.DefaultType != "" {
					needs = []string{rule.UnsatisfiedAction.DefaultType}
				} else {
					if rule.Classifications != nil {
						needs = rule.Classifications
					} else {
						needs = rule.ResourceTypes
					}
				}
				ore = &core.OperationalResourceError{
					Resource:   resource,
					Parent:     downstreamParent,
					Count:      rule.NumNeeded - len(downstreamResourcesOfType),
					MustCreate: rule.UnsatisfiedAction.Unique,
					Needs:      needs,
					Cause:      fmt.Errorf("downstream rule with enforcement any has less than the required number of resources of type %s, %d", rule.ResourceTypes, len(downstreamResourcesOfType)),
				}
			case core.ErrorUnsatisfiedResource:
				return []error{fmt.Errorf("downstream rule with enforcement any has less than the required number of resources of type %s, %d", rule.ResourceTypes, len(downstreamResourcesOfType))}
			}
		}
		var subRuleErrors []error
		for _, subRule := range rule.Rules {
			err := e.handleDownstreamOperationalRule(resource, subRule, dag, nil)
			if err != nil {
				subRuleErrors = append(subRuleErrors, err...)
			}
		}
		if subRuleErrors != nil {
			return subRuleErrors
		}
		if ore != nil {
			return []error{ore}
		}
		if len(downstreamResourcesOfType) != rule.NumNeeded {
			return []error{fmt.Errorf("downstream rule with enforcement any available has less than the required number of resources of type %s, %d", rule.ResourceTypes, len(downstreamResourcesOfType))}
		}
	default:
		return []error{fmt.Errorf("unknown enforcement type %s", rule.Enforcement)}
	}
	return nil
}

func setField(resource core.Resource, rule core.OperationalRule, res core.Resource) {
	if rule.SetField == "" {
		return
	}
	if reflect.ValueOf(resource).Elem().FieldByName(rule.SetField).Kind() == reflect.Slice || reflect.ValueOf(resource).Elem().FieldByName(rule.SetField).Kind() == reflect.Array {
		reflect.ValueOf(resource).Elem().FieldByName(rule.SetField).Set(reflect.Append(reflect.ValueOf(resource).Elem().FieldByName(rule.SetField), reflect.ValueOf(res)))
	} else {
		reflect.ValueOf(resource).Elem().FieldByName(rule.SetField).Set(reflect.ValueOf(res))
	}
}

func nameResource(res core.Resource, resource core.Resource, addToName string) {
	reflect.ValueOf(res).Elem().FieldByName("Name").Set(reflect.ValueOf(fmt.Sprintf("%s-%s%s", res.Id().Type, resource.Id().Name, addToName)))
	reflect.ValueOf(res).Elem().FieldByName("ConstructRefs").Set(reflect.ValueOf(core.BaseConstructSetOf(resource)))
}

// handleOperationalResourceError tries to determine how to fix OperatioanlResourceErrors by adding dependencies to the resource graph where needed.
// If the error cannot be fixed, it will return an error.
func (e *Engine) handleDownstreamOperationalResourceError(err *core.OperationalResourceError, dag *core.ResourceGraph) error {

	resources := e.ListResources()

	// determine the type of resource necessary to satisfy the operational resource error
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

	// first check if the parent resource passed into the error has any upstream resources we can reuse
	numSatisfied := 0
	if err.Parent != nil {
		upstreamResources := dag.GetAllUpstreamResources(err.Parent)
		for _, res := range upstreamResources {
			if res.Id().Type == neededResource.Id().Type && res.Id().Provider == neededResource.Id().Provider && dag.GetDependency(err.Resource.Id(), res.Id()) == nil {
				dag.AddDependency(err.Resource, res)
				numSatisfied++
			}
		}
	}
	if numSatisfied == err.Count {
		return nil
	}

	// determine if there are any available resources in the graph that we can reuse
	var availableResources []core.Resource
	// we only want to look at available resources if we dont have a parent they need to be scoped to.
	// This prevents us from saying that resource_a is available if it is a child of resource_b when the error has a parent of resource_c
	if err.Parent == nil && !err.MustCreate {
		//Todo: Get nearest resource. we should look one resource upstream until we find available resources so that we have a higher chance of choosing the right one
		for _, res := range dag.ListResources() {
			if res.Id().Type == neededResource.Id().Type {
				availableResources = append(availableResources, res)
			}
		}
	}
	// if theres no available resources from us to choose from, we must create new resources
	if len(availableResources) == 0 {
		for i := numSatisfied; i < err.Count; i++ {
			newRes := reflect.New(reflect.TypeOf(neededResource).Elem()).Interface().(core.Resource)
			if err.Count-numSatisfied == 1 {
				nameResource(newRes, err.Resource, "")
			} else {
				nameResource(newRes, err.Resource, fmt.Sprintf("%d", i))
			}
			dag.AddDependency(err.Resource, newRes)
			if err.Parent != nil {
				dag.AddDependency(newRes, err.Parent)
			}
		}
	} else {
		resourceIds := []string{}
		for _, res := range availableResources {
			resourceIds = append(resourceIds, res.Id().Name)
		}
		sort.Strings(resourceIds)
		if err.Count-numSatisfied > len(resourceIds) {
			return fmt.Errorf("not enough resources found that can satisfy operational exception error, %s", err.Error())
		}
		for i := 0; i < err.Count-numSatisfied; i++ {
			for _, res := range availableResources {
				if res.Id().Name == resourceIds[i] {
					dag.AddDependency(err.Resource, res)
					break
				}
			}
		}
	}
	return nil
}

func TemplateConfigure(resource core.Resource, template core.ResourceTemplate) error {
	for _, config := range template.Configuration {
		field := reflect.ValueOf(resource).Elem().FieldByName(config.Field)
		switch field.Kind() {
		case reflect.Slice, reflect.Array:
			if field.Len() == 0 && !config.ZeroValueAllowed {
				if reflect.ValueOf(config.Value).Kind() != reflect.Slice {
					return fmt.Errorf("config template is not the correct type for resource %s. expected it to be a list, but got %s", resource.Id(), reflect.TypeOf(config.Value))
				}
				configureField(config.Value, field)
				reflect.ValueOf(resource).Elem().FieldByName(config.Field).Set(field)
			}

		case reflect.Pointer, reflect.Struct:
			if reflect.ValueOf(config.Value).Kind() != reflect.Map {
				return fmt.Errorf("config template is not the correct type for resource %s. expected it to be a map, but got %s", resource.Id(), reflect.TypeOf(config.Value))
			}
			configureField(config.Value, field)
			reflect.ValueOf(resource).Elem().FieldByName(config.Field).Set(field)
		default:
			configureField(config.Value, field)
		}
	}
	return nil
}

func configureField(val interface{}, field reflect.Value) {
	switch field.Kind() {
	case reflect.Slice, reflect.Array:
		arr := field
		for i := 0; i < reflect.ValueOf(val).Len(); i++ {
			val := reflect.ValueOf(val).Index(i).Interface()
			if field.Type().Elem().Kind() == reflect.Struct {
				subField := reflect.New(field.Type().Elem()).Interface()
				configureField(val, reflect.ValueOf(subField))
				arr = reflect.Append(arr, reflect.ValueOf(subField).Elem())
			} else if field.Type().Elem().Kind() == reflect.Ptr {
				subField := reflect.New(field.Type().Elem().Elem()).Interface()
				configureField(val, reflect.ValueOf(subField).Elem())
				arr = reflect.Append(arr, reflect.ValueOf(subField))
			} else {
				arr = reflect.Append(arr, reflect.ValueOf(val))
			}
		}
		field.Set(arr)
	case reflect.Struct, reflect.Ptr:
		if field.Kind() == reflect.Ptr && field.IsNil() {
			field.Set(reflect.New(reflect.TypeOf(field.Interface()).Elem()))
		}
		if field.Kind() == reflect.Ptr {
			field = field.Elem()
		}
		for _, key := range reflect.ValueOf(val).MapKeys() {
			for i := 0; i < field.NumField(); i++ {
				if field.Type().Field(i).Name == key.String() {
					configureField(reflect.ValueOf(val).MapIndex(key).Interface(), field.Field(i))
				}
			}
		}
	default:
		field.Set(reflect.ValueOf(val))
	}

}
