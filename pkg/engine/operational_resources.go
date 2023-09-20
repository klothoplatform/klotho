package engine

import (
	"errors"
	"fmt"
	"reflect"
	"sort"

	"github.com/klothoplatform/klotho/pkg/collectionutil"
	"github.com/klothoplatform/klotho/pkg/construct"
	"github.com/klothoplatform/klotho/pkg/engine/classification"
	"github.com/klothoplatform/klotho/pkg/graph"
	"go.uber.org/zap"

	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base"
)

type OperationalResource interface {
	construct.Resource
	MakeOperational(dag *construct.ResourceGraph, appName string, classifier classification.Classifier) error
}

// MakeResourcesOperational runs a set of rules to make a single resource, the parameter, operational.
//
// The rules are defined in the knowledge base template for the resource and are applied to the resource graph.
// all errors and decisions are recorded in the context.
func (e *Engine) MakeResourceOperational(context *SolveContext, resource construct.Resource) bool {
	var engineErrors []EngineError
	template := e.GetTemplateForResource(resource)
	if template != nil {
		for _, rule := range template.Rules {
			decisions, errs := e.handleOperationalRule(resource, rule, context.ResourceGraph, nil)
			for _, d := range decisions {
				d.Cause = &Cause{OperationalResource: resource}
				e.handleDecision(context, d)
			}
			for _, err := range errs {
				if err != nil {
					if ore, ok := err.(*OperationalResourceError); ok {
						decisions, err := e.handleOperationalResourceError(ore, context.ResourceGraph)
						for _, d := range decisions {
							d.Cause = &Cause{OperationalResource: resource}
							e.handleDecision(context, d)
						}
						if err != nil {
							engineErrors = append(engineErrors, &ResourceNotOperationalError{
								Resource:                 resource,
								Cause:                    err,
								OperationalResourceError: *ore,
							})
						}
						continue
					}
					engineErrors = append(engineErrors, &ResourceNotOperationalError{
						Resource: resource,
						Cause:    err,
					})
				}
			}
		}
	}

	err := callMakeOperational(context.ResourceGraph, resource, e.Context.AppName, e.ClassificationDocument)
	if err != nil {
		if ore, ok := err.(*OperationalResourceError); ok {
			// If we get a OperationalResourceError let the engine try to reconcile it, and if that fails then mark the resource as non operational so we attempt to rerun on the next loop
			decisions, herr := e.handleOperationalResourceError(ore, context.ResourceGraph)
			for _, d := range decisions {
				d.Cause = &Cause{OperationalResource: resource}
				e.handleDecision(context, d)
			}
			if herr != nil {
				engineErrors = append(engineErrors, &ResourceNotOperationalError{
					Resource:                 resource,
					Cause:                    err,
					OperationalResourceError: *ore,
				})
			}

		} else {
			engineErrors = append(engineErrors, &ResourceNotOperationalError{
				Resource: resource,
				Cause:    err,
			})
		}
	}
	if len(engineErrors) > 0 {
		context.Errors = append(context.Errors, engineErrors...)
		return false
	}
	return true
}

func callMakeOperational(rg *construct.ResourceGraph, resource construct.Resource, appName string, classifier classification.Classifier) error {
	operationalResource, ok := resource.(OperationalResource)
	if !ok {
		return nil
	}
	if rg.GetResource(resource.Id()) == nil {
		return fmt.Errorf("resource with id %s cannot be made operational since it does not exist in the ResourceGraph", resource.Id())
	}
	return operationalResource.MakeOperational(rg, appName, classifier)
}

func (e *Engine) handleOperationalRule(resource construct.Resource, rule knowledgebase.OperationalRule, dag *construct.ResourceGraph, downstreamParent construct.Resource) ([]Decision, []EngineError) {
	resourcesOfType := []construct.Resource{}

	if rule.If != "" {
		ctx := knowledgebase.ConfigTemplateContext{DAG: dag}
		data := knowledgebase.ConfigTemplateData{Resource: resource.Id()}
		result := false
		err := ctx.ExecuteDecode(rule.If, data, &result)
		if err != nil {
			return nil, []EngineError{&OperationalResourceError{
				Rule:     rule,
				Resource: resource,
				Cause:    err,
			}}
		}
		if !result {
			zap.S().Debugf("rule %s for resource %s did not match if condition, skippingw", rule.String(), resource.Id())
			return nil, nil
		}
	}

	var dependentResources []construct.Resource
	if rule.Direction == knowledgebase.Upstream {
		dependentResources = dag.GetUpstreamResources(resource)
		if rule.Rules != nil && rule.RemoveDirectDependency {
			dependentResources = dag.GetAllUpstreamResources(resource)
		}
	} else {
		dependentResources = dag.GetDownstreamResources(resource)
		if rule.Rules != nil && rule.RemoveDirectDependency {
			dependentResources = dag.GetAllDownstreamResources(resource)
		}
	}
	if rule.ResourceTypes != nil && rule.Classifications != nil && rule.Resources != nil {
		return nil, []EngineError{
			&InternalError{
				Child: &ResourceNotOperationalError{Resource: resource},
				Cause: fmt.Errorf("rule cannot have both resource types and classifications defined %s for resource %s", rule.String(), resource.Id()),
			},
		}
	} else if len(rule.ResourceTypes) > 0 {
		for _, res := range dependentResources {
			if collectionutil.Contains(rule.ResourceTypes, res.Id().Type) && res.Id().Provider == resource.Id().Provider {
				resourcesOfType = append(resourcesOfType, res)
			}
		}
	} else if len(rule.Resources) > 0 {
		return e.handleExactResourceEnforcement(resource, rule, dag)
	} else if len(rule.Classifications) > 0 {
		for _, res := range dependentResources {
			if e.ClassificationDocument.ResourceContainsClassifications(res, rule.Classifications) {
				resourcesOfType = append(resourcesOfType, res)
			}
		}
	} else {
		return nil, []EngineError{
			&InternalError{
				Child: &ResourceNotOperationalError{Resource: resource},
				Cause: fmt.Errorf("rule must have either a resource type or classifications defined %s for resource %s", rule.String(), resource.Id()),
			},
		}
	}
	switch rule.Enforcement {
	case knowledgebase.ExactlyOne:
		return e.handleExactlyOneEnforcement(resource, rule, resourcesOfType, downstreamParent, dag)
	case knowledgebase.Conditional:
		return e.handleConditionalEnforcement(resource, rule, resourcesOfType, downstreamParent, dag)
	case knowledgebase.AnyAvailable:
		return e.handleAnyAvailableEnforcement(resource, rule, resourcesOfType, downstreamParent, dag)
	default:
		return nil, []EngineError{
			&InternalError{
				Child: &ResourceNotOperationalError{Resource: resource},
				Cause: fmt.Errorf("unknown enforcement type %s, for resource %s", rule.Enforcement, resource.Id()),
			},
		}
	}
}

func (e *Engine) handleExactResourceEnforcement(resource construct.Resource, rule knowledgebase.OperationalRule, dag *construct.ResourceGraph) (decisions []Decision, errs []EngineError) {
	ctx := knowledgebase.ConfigTemplateContext{DAG: dag}
	data := knowledgebase.ConfigTemplateData{Resource: resource.Id()}

	addDep := func(dep construct.Resource) {
		var result DecisionResult
		if rule.Direction == knowledgebase.Upstream {
			result.Edge = &graph.Edge[construct.Resource]{
				Source:      dep,
				Destination: resource,
			}
		} else {
			result.Edge = &graph.Edge[construct.Resource]{
				Source:      resource,
				Destination: dep,
			}
		}
		decisions = append(decisions, Decision{
			Action: ActionConnect,
			Result: &result,
			Cause:  &Cause{OperationalResource: resource},
		})

		err := e.setField(dag, resource, rule, dep)
		if err != nil {
			errs = append(errs, &ResourceNotOperationalError{
				Resource: resource,
				Cause:    err,
			})
		}
	}

requiredLoop:
	for _, resStr := range rule.Resources {
		var selector construct.ResourceId
		err := ctx.ExecuteDecode(resStr, data, &selector)
		if err != nil {
			errs = append(errs, &InternalError{
				Child: &ResourceNotOperationalError{Resource: resource, Cause: err},
				Cause: err,
			})
			continue
		}
		if selector.IsZero() {
			// ? Should this error instead?
			// Make sure we don't just add arbitrary dependencies, since all resources match the zero value
			continue
		}

		if selector.Name != "" {
			if r := dag.GetResource(selector); r != nil {
				addDep(r)
				continue
			}
		} else {
			for _, r := range dag.ListResources() {
				if selector.Matches(r.Id()) {
					addDep(r)
					continue requiredLoop
				}
			}
		}

		errs = append(errs, &OperationalResourceError{
			Rule:     rule,
			Resource: resource,
			ToCreate: selector,
			Count:    1,
		})
	}
	return
}

func (e *Engine) handleExactlyOneEnforcement(resource construct.Resource, rule knowledgebase.OperationalRule, resourcesOfType []construct.Resource, downstreamParent construct.Resource, dag *construct.ResourceGraph) ([]Decision, []EngineError) {
	var decisions []Decision
	if len(resourcesOfType) > 1 {
		ids := make([]string, len(resourcesOfType))
		for i, res := range resourcesOfType {
			ids[i] = res.Id().String()
		}
		sort.Strings(ids)
		return decisions, []EngineError{
			&ResourceNotOperationalError{
				Resource: resource,
				Cause:    fmt.Errorf("rule with enforcement exactly one has more than one resource for rule %s for resource %s (%v)", rule.String(), resource.Id(), ids),
			},
		}
	} else if len(resourcesOfType) == 0 {
		switch rule.UnsatisfiedAction.Operation {
		case knowledgebase.CreateUnsatisfiedResource:
			var oreParent construct.Resource
			if !rule.NoParentDependency {
				oreParent = downstreamParent
			}
			return decisions, []EngineError{&OperationalResourceError{
				Rule:     rule,
				Resource: resource,
				Count:    1,
				Parent:   oreParent,
				Cause:    fmt.Errorf("rule with enforcement exactly one has less than the required number of resources of type %s  or classifications %s, %d for resource %s", rule.ResourceTypes, rule.Classifications, len(resourcesOfType), resource.Id()),
			}}

		case knowledgebase.ErrorUnsatisfiedResource:
			return decisions, []EngineError{
				&ResourceNotOperationalError{
					Resource: resource,
					Cause:    fmt.Errorf("rule with enforcement exactly one has less than the required number of resources of type %s  or classifications %s, %d for resource %s", rule.ResourceTypes, rule.Classifications, len(resourcesOfType), resource.Id()),
				},
			}
		}
	} else {
		res := resourcesOfType[0]
		if !rule.RemoveDirectDependency {
			decisions = append(decisions, addDependencyDecisionForDirection(rule.Direction, resource, res))
		}
		err := e.setField(dag, resource, rule, res)
		if err != nil {
			return decisions, []EngineError{
				&ResourceNotOperationalError{
					Resource: resource,
					Cause:    err,
				},
			}
		}
		if downstreamParent != nil && !rule.NoParentDependency {
			decisions = append(decisions, addDependencyDecisionForDirection(rule.Direction, res, downstreamParent))
		}
		if rule.RemoveDirectDependency {
			if getDependencyForDirection(dag, rule.Direction, resource, res) != nil {
				decisions = append(decisions, removeDependencyDecisionForDirection(rule.Direction, resource, res))
			}
		}
	}

	var subRuleErrors []EngineError
	for _, subRule := range rule.Rules {
		subRuleDecisions, err := e.handleOperationalRule(resource, subRule, dag, nil)
		if err != nil {
			subRuleErrors = append(subRuleErrors, err...)
		}
		decisions = append(decisions, subRuleDecisions...)
	}
	if subRuleErrors != nil {
		return decisions, subRuleErrors
	}
	return decisions, nil
}

func (e *Engine) handleConditionalEnforcement(resource construct.Resource, rule knowledgebase.OperationalRule, resourcesOfType []construct.Resource, downstreamParent construct.Resource, dag *construct.ResourceGraph) ([]Decision, []EngineError) {
	var decisions []Decision
	if len(resourcesOfType) == 0 {
		if rule.NumNeeded > 0 {
			return decisions, []EngineError{
				&ResourceNotOperationalError{
					Resource: resource,
					Cause:    fmt.Errorf("rule with enforcement conditional has less than the required number of resources of type %s  or classifications %s, %d for resource %s", rule.ResourceTypes, rule.Classifications, len(resourcesOfType), resource.Id()),
				},
			}
		}
		return decisions, nil
	} else if len(resourcesOfType) == 1 {
		err := e.setField(dag, resource, rule, resourcesOfType[0])
		if err != nil {
			return decisions, []EngineError{
				&ResourceNotOperationalError{
					Resource: resource,
					Cause:    err,
				},
			}
		}
		if rule.RemoveDirectDependency {
			if getDependencyForDirection(dag, rule.Direction, resource, resourcesOfType[0]) != nil {
				decisions = append(decisions, removeDependencyDecisionForDirection(rule.Direction, resource, resourcesOfType[0]))
			}
		}
	} else {
		setFieldErrors := []EngineError{}
		for _, res := range resourcesOfType {
			err := e.setField(dag, resource, rule, res)
			if err != nil {
				setFieldErrors = append(setFieldErrors, &ResourceNotOperationalError{
					Resource: resource,
					Cause:    err,
				})
			}
		}
		if len(setFieldErrors) > 0 {
			return decisions, setFieldErrors
		}
	}
	var subRuleErrors []EngineError
	for _, subRule := range rule.Rules {
		subRuleDecisions, err := e.handleOperationalRule(resource, subRule, dag, resourcesOfType[0])
		if err != nil {
			subRuleErrors = append(subRuleErrors, err...)
		}
		decisions = append(decisions, subRuleDecisions...)
	}
	if subRuleErrors != nil {
		return decisions, subRuleErrors
	}
	return decisions, nil
}

func (e *Engine) handleAnyAvailableEnforcement(resource construct.Resource, rule knowledgebase.OperationalRule, resourcesOfType []construct.Resource, downstreamParent construct.Resource, dag *construct.ResourceGraph) ([]Decision, []EngineError) {
	var decisions []Decision
	for _, res := range resourcesOfType {
		err := e.setField(dag, resource, rule, res)
		if err != nil {
			return decisions, []EngineError{
				&ResourceNotOperationalError{
					Resource: resource,
					Cause:    err,
				},
			}
		}
	}
	if rule.NumNeeded > len(resourcesOfType) {
		switch rule.UnsatisfiedAction.Operation {
		case knowledgebase.CreateUnsatisfiedResource:
			var oreParent construct.Resource
			if !rule.NoParentDependency {
				oreParent = downstreamParent
			}
			return decisions, []EngineError{&OperationalResourceError{
				Rule:     rule,
				Resource: resource,
				Count:    rule.NumNeeded - len(resourcesOfType),
				Parent:   oreParent,
				Cause:    fmt.Errorf("rule with enforcement any has less than the required number of resources of type %s  or classifications %s, %d for resource %s", rule.ResourceTypes, rule.Classifications, len(resourcesOfType), resource.Id()),
			}}
		case knowledgebase.ErrorUnsatisfiedResource:
			return decisions, []EngineError{
				&ResourceNotOperationalError{
					Resource: resource,
					Cause:    fmt.Errorf("unsatisfied resource error: rule with enforcement any has less than the required number of resources of type %s  or classifications %s, %d, for resource %s", rule.ResourceTypes, rule.Classifications, len(resourcesOfType), resource.Id()),
				},
			}
		}
	}
	var subRuleErrors []EngineError
	for _, subRule := range rule.Rules {
		subRuleDecisions, err := e.handleOperationalRule(resource, subRule, dag, nil)
		if err != nil {
			subRuleErrors = append(subRuleErrors, err...)
		}
		decisions = append(decisions, subRuleDecisions...)
	}
	if subRuleErrors != nil {
		return decisions, subRuleErrors
	}
	return decisions, nil
}

func (e *Engine) setField(dag *construct.ResourceGraph, resource construct.Resource, rule knowledgebase.OperationalRule, fieldResource construct.Resource) error {
	if rule.SetField == "" {
		return nil
	}
	// snapshot the ID from before any field changes
	oldId := resource.Id()

	resVal := reflect.ValueOf(resource)
	fieldValue := reflect.ValueOf(fieldResource)

	field := resVal.Elem().FieldByName(rule.SetField)

	if field.Kind() == reflect.Slice || field.Kind() == reflect.Array {
		field.Set(reflect.Append(field, fieldValue))
	} else {
		if field.Kind() == reflect.Ptr && !field.IsNil() {
			oldFieldValue := field.Interface()
			if oldRes, ok := oldFieldValue.(construct.Resource); ok && fieldResource.Id() != oldRes.Id() {
				err := dag.RemoveDependency(resource.Id(), oldRes.Id())
				if err != nil {
					return err
				}
				zap.S().Infof("Removing old field value for '%s' (%s) for %s", rule.SetField, oldRes.Id(), fieldResource.Id())
				// Remove the old field value if it's unused
				e.deleteResource(dag, oldRes, false, false)
			}
		}

		if resourceIdType.AssignableTo(field.Type()) {
			field.Set(reflect.ValueOf(fieldResource.Id()))
		} else {
			field.Set(fieldValue)
		}
	}
	zap.S().Infof("set field %s#%s to %s", resource.Id(), rule.SetField, fieldResource.Id())
	// If this sets the field driving the namespace, for example,
	// then the Id could change, so replace the resource in the graph
	// to update all the edges to the new Id.
	if oldId != resource.Id() {
		err := dag.ReplaceConstructId(oldId, resource)
		if err != nil {
			return err
		}
	}
	return nil
}

// handleOperationalResourceError tries to determine how to fix OperatioanlResourceErrors by adding dependencies to the resource graph where needed.
// If the error cannot be fixed, it will return an error.
func (e *Engine) handleOperationalResourceError(operation *OperationalResourceError, dag *construct.ResourceGraph) ([]Decision, error) {
	var decisions []Decision
	if !operation.ToCreate.IsZero() && operation.ToCreate.Name != "" {
		if operation.Count > 1 {
			return nil, fmt.Errorf("cannot create multiple resources for a specific resource id %s", operation.ToCreate)
		}
		r, err := e.CreateResourceFromId(operation.ToCreate)
		if err != nil {
			return nil, err
		}
		err = e.setField(dag, operation.Resource, operation.Rule, r)
		if err != nil {
			return nil, err
		}
		var edge *graph.Edge[construct.Resource]
		if operation.Rule.Direction == knowledgebase.Downstream {
			edge = &graph.Edge[construct.Resource]{
				Source:      operation.Resource,
				Destination: r,
			}
		} else {
			edge = &graph.Edge[construct.Resource]{
				Source:      r,
				Destination: operation.Resource,
			}
		}
		return []Decision{{
			Level:  LevelInfo,
			Action: ActionConnect,
			Cause:  &Cause{},
			Result: &DecisionResult{
				Edge: edge,
			},
		}}, nil
	}

	resources := e.ListResources()
	var needs []string
	switch {
	case len(operation.Rule.Classifications) > 0:
		needs = operation.Rule.Classifications

	case len(operation.Rule.ResourceTypes) > 0:
		// Pick the first one, assume the template writer prioritized which one should be created
		needs = []string{operation.Rule.ResourceTypes[0]}

	case operation.ToCreate.Type != "":
		needs = []string{operation.ToCreate.Type}

	case operation.Rule.UnsatisfiedAction.DefaultType != "":
		needs = []string{operation.Rule.UnsatisfiedAction.DefaultType}
	}
	// determine the type of resource necessary to satisfy the operational resource error
	var neededResource construct.Resource
	for _, res := range resources {
		if !e.ClassificationDocument.ResourceContainsClassifications(res, needs) {
			continue
		}
		var hasPath bool
		if operation.Rule.Direction == knowledgebase.Downstream {
			hasPath = e.KnowledgeBase.HasPath(operation.Resource, res)
		} else {
			hasPath = e.KnowledgeBase.HasPath(res, operation.Resource)
		}
		// if a type is explicilty stated as needed, we will consider it even if there isnt a direct p
		if !hasPath {
			continue
		}
		neededResource = res
		break
	}
	if neededResource == nil {
		return nil, fmt.Errorf("no resources found that can satisfy the operational resource error")
	}

	// first check if the parent resource passed into the error has any upstream resources we can reuse
	numSatisfied := 0
	if operation.Parent != nil {
		var resources []construct.Resource
		// The direction here is flipped since we are looking at the resources relative to the parent, not relative to the resource used in the error
		if operation.Rule.Direction == knowledgebase.Upstream {
			resources = dag.GetAllDownstreamResources(operation.Parent)
		} else {
			resources = dag.GetAllUpstreamResources(operation.Parent)
		}
		var errs error
		for _, res := range resources {
			if res.Id().Type == neededResource.Id().Type && res.Id().Provider == neededResource.Id().Provider && dag.GetDependency(operation.Resource.Id(), res.Id()) == nil {
				decisions = append(decisions, addDependencyDecisionForDirection(operation.Rule.Direction, operation.Resource, res))
				errs = errors.Join(errs, e.setField(dag, operation.Resource, operation.Rule, res))
				numSatisfied++
			}
		}
		if errs != nil {
			return decisions, errs
		}
	}
	if numSatisfied == operation.Count {
		return decisions, nil
	}

	// determine if there are any available resources in the graph that we can reuse
	var availableResources []construct.Resource
	// we only want to look at available resources if we dont have a parent they need to be scoped to.
	// This prevents us from saying that resource_a is available if it is a child of resource_b when the error has a parent of resource_c
	if operation.Parent == nil && !operation.Rule.UnsatisfiedAction.Unique {
		//Todo: Get nearest resource. we should look one resource upstream until we find available resources so that we have a higher chance of choosing the right one
		for _, res := range dag.ListResources() {
			if res.Id().Type == neededResource.Id().Type {
				availableResources = append(availableResources, res)
			}
		}
	}
	resourceIds := []string{}
	for _, res := range availableResources {
		resourceIds = append(resourceIds, res.Id().Name)
	}
	sort.Strings(resourceIds)

	currNumSatisfied := numSatisfied
	var errs error
	for i := 0; i < operation.Count-currNumSatisfied; i++ {
		for _, res := range availableResources {
			if len(resourceIds) > i && res.Id().Name == resourceIds[i] {
				decisions = append(decisions, addDependencyDecisionForDirection(operation.Rule.Direction, operation.Resource, res))
				errs = errors.Join(errs, e.setField(dag, operation.Resource, operation.Rule, res))
				numSatisfied++
				break
			}
		}
	}
	if errs != nil {
		return decisions, errs
	}

	// if theres no available resources from us to choose from, we must create new resources
	if len(availableResources) < operation.Count-numSatisfied {
		// We track the number of resources of the same type here for naming purposes, since we dont actually create new resources in this method we need to increment when we detect our decision will create a new resource
		numResources := 0
		for _, res := range dag.ListResources() {
			if res.Id().Type == neededResource.Id().Type {
				numResources++
			}
		}
		var errs error
		for i := numSatisfied; i < operation.Count; i++ {
			id := neededResource.Id()
			id.Name = generateResourceName(numResources, neededResource, operation.Resource, operation.Rule.UnsatisfiedAction.Unique)
			newRes, err := e.CreateResourceFromId(id)
			if err != nil {
				errs = errors.Join(errs, err)
				continue
			}
			newRes.BaseConstructRefs().Add(operation.Resource)

			decisions = append(decisions, addDependencyDecisionForDirection(operation.Rule.Direction, operation.Resource, newRes))
			if operation.Parent != nil {
				decisions = append(decisions, addDependencyDecisionForDirection(operation.Rule.Direction, newRes, operation.Parent))
			}
			errs = errors.Join(errs, e.setField(dag, operation.Resource, operation.Rule, newRes))
			numResources++
		}
		if errs != nil {
			return decisions, errs
		}
	}

	return decisions, nil
}

func cloneResource(resource construct.Resource) construct.Resource {
	newRes := reflect.New(reflect.TypeOf(resource).Elem()).Interface().(construct.Resource)
	for i := 0; i < reflect.ValueOf(newRes).Elem().NumField(); i++ {
		field := reflect.ValueOf(newRes).Elem().Field(i)
		field.Set(reflect.ValueOf(resource).Elem().Field(i))
	}
	return newRes
}

func generateResourceName(numResources int, resourceToSet construct.Resource, resource construct.Resource, unique bool) string {
	if unique {
		return fmt.Sprintf("%s-%s-%d", resourceToSet.Id().Type, resource.Id().Name, numResources)
	}
	return fmt.Sprintf("%s-%d", resourceToSet.Id().Type, numResources)
}

func addDependencyDecisionForDirection(direction knowledgebase.Direction, resource construct.Resource, dependentResource construct.Resource) Decision {
	if direction == knowledgebase.Upstream {
		return Decision{
			Action: ActionConnect,
			Result: &DecisionResult{Edge: &graph.Edge[construct.Resource]{Source: dependentResource, Destination: resource}},
		}
	} else {
		return Decision{
			Action: ActionConnect,
			Result: &DecisionResult{Edge: &graph.Edge[construct.Resource]{Source: resource, Destination: dependentResource}},
		}
	}
}

func removeDependencyDecisionForDirection(direction knowledgebase.Direction, resource construct.Resource, dependentResource construct.Resource) Decision {
	if direction == knowledgebase.Upstream {
		return Decision{
			Action: ActionDisconnect,
			Result: &DecisionResult{Edge: &graph.Edge[construct.Resource]{Source: dependentResource, Destination: resource}},
		}
	} else {
		return Decision{
			Action: ActionDisconnect,
			Result: &DecisionResult{Edge: &graph.Edge[construct.Resource]{Source: resource, Destination: dependentResource}},
		}
	}
}

func getDependencyForDirection(dag *construct.ResourceGraph, direction knowledgebase.Direction, resource construct.Resource, dependentResource construct.Resource) *graph.Edge[construct.Resource] {
	if direction == knowledgebase.Upstream {
		return dag.GetDependency(dependentResource.Id(), resource.Id())
	} else {
		return dag.GetDependency(resource.Id(), dependentResource.Id())
	}
}

func (e *Engine) isSideEffect(dag *construct.ResourceGraph, resource construct.Resource, sideEffect construct.Resource) bool {
	template := e.GetTemplateForResource(resource)
	if template == nil {
		return false
	}
	for _, rule := range template.Rules {
		if rule.ResourceTypes != nil && collectionutil.Contains(rule.ResourceTypes, sideEffect.Id().Type) || rule.Classifications != nil && e.ClassificationDocument.ResourceContainsClassifications(sideEffect, rule.Classifications) {
			if rule.Direction == knowledgebase.Upstream {
				resources, err := dag.ShortestPath(sideEffect.Id(), resource.Id())
				if len(resources) == 0 || err != nil {
					return false
				}
			} else {
				resources, err := dag.ShortestPath(resource.Id(), sideEffect.Id())
				if len(resources) == 0 || err != nil {
					return false
				}
			}
			if rule.SetField != "" {
				val, _, err := parseFieldName(resource, rule.SetField, dag, false)
				if err != nil {
					return false
				}
				if val.Kind() == reflect.Array || val.Kind() == reflect.Slice {
					for i := 0; i < val.Len(); i++ {
						if val.Index(i).Interface().(construct.Resource).Id() == sideEffect.Id() {
							return true
						}
					}
				} else {
					if val.Interface().(construct.Resource).Id() == sideEffect.Id() {
						return true
					}
				}
			} else {
				return true
			}

		}
	}
	return false
}
