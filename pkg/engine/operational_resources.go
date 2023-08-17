package engine

import (
	"errors"
	"fmt"
	"reflect"
	"sort"

	"github.com/klothoplatform/klotho/pkg/collectionutil"
	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/engine/classification"
	"github.com/klothoplatform/klotho/pkg/graph"

	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base"
	"go.uber.org/zap"
)

type OperationalResource interface {
	core.Resource
	MakeOperational(dag *core.ResourceGraph, appName string, classifier classification.Classifier) error
}

func (e *Engine) MakeResourcesOperational(graph *core.ResourceGraph) (map[core.ResourceId]bool, error) {
	zap.S().Debug("Engine Making resources operational and configuring resources")
	operationalResources := map[core.ResourceId]bool{}
	var joinedErr error
	resources, err := graph.ReverseTopologicalSort()
	if err != nil {
		return nil, err
	}
	for _, resource := range resources {
		err := e.MakeResourceOperational(graph, resource)
		if err != nil {
			joinedErr = errors.Join(joinedErr, err)
		} else {
			operationalResources[resource.Id()] = true
		}
	}
	zap.S().Debug("Engine done making resources operational and configuring resources")
	return operationalResources, joinedErr
}

func (e *Engine) MakeResourceOperational(graph *core.ResourceGraph, resource core.Resource) error {
	template := e.ResourceTemplates[core.ResourceId{Provider: resource.Id().Provider, Type: resource.Id().Type}]
	if template != nil {
		err := e.TemplateMakeOperational(graph, resource, *template)
		if err != nil {
			return err
		}
		err = TemplateConfigure(resource, *template, graph)
		if err != nil {
			return err

		}
	}

	err := callMakeOperational(graph, resource, e.Context.AppName, e.ClassificationDocument)
	if err != nil {
		if ore, ok := err.(*core.OperationalResourceError); ok {
			// If we get a OperationalResourceError let the engine try to reconcile it, and if that fails then mark the resource as non operational so we attempt to rerun on the next loop
			herr := e.handleOperationalResourceError(ore, graph)
			if herr != nil {
				err = errors.Join(err, herr)
			}
			return err
		} else {
			return err
		}
	}

	err = graph.CallConfigure(resource, nil)
	if err != nil {
		return err

	}

	return nil
}

func callMakeOperational(rg *core.ResourceGraph, resource core.Resource, appName string, classifier classification.Classifier) error {
	operationalResource, ok := resource.(OperationalResource)
	if !ok {
		return nil
	}
	if rg.GetResource(resource.Id()) == nil {
		return fmt.Errorf("resource with id %s cannot be made operational since it does not exist in the ResourceGraph", resource.Id())
	}
	return operationalResource.MakeOperational(rg, appName, classifier)
}

func (e *Engine) TemplateMakeOperational(dag *core.ResourceGraph, resource core.Resource, template core.ResourceTemplate) error {
	var joinedErr error
	for _, rule := range template.Rules {
		errs := e.handleOperationalRule(resource, rule, dag, nil)
		for _, err := range errs {
			if err != nil {
				if ore, ok := err.(*core.OperationalResourceError); ok {
					err := e.handleOperationalResourceError(ore, dag)
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

func (e *Engine) handleOperationalRule(resource core.Resource, rule core.OperationalRule, dag *core.ResourceGraph, downstreamParent core.Resource) []error {
	resourcesOfType := []core.Resource{}

	// if we are supposed to set a field and the field is already set and has the number of resources needed, we dont need to run this function
	// Also make sure theres no sub rules so we dont short circuit
	if rule.SetField != "" && len(rule.Rules) == 0 {
		field := reflect.ValueOf(resource).Elem().FieldByName(rule.SetField)
		if field.IsValid() {
			if (field.Kind() == reflect.Slice || field.Kind() == reflect.Array) && field.Len() > rule.NumNeeded {
				return nil
			} else if field.Kind() == reflect.Ptr && !field.IsNil() {
				return nil
			}
		}
	}

	var dependentResources []core.Resource
	if rule.Direction == core.Upstream {
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
	if rule.ResourceTypes != nil && rule.Classifications != nil {
		return []error{fmt.Errorf("rule cannot have both a resource type and classifications defined %s for resource %s", rule.String(), resource.Id())}
	} else if rule.ResourceTypes != nil {
		for _, res := range dependentResources {
			if collectionutil.Contains(rule.ResourceTypes, res.Id().Type) && res.Id().Provider == resource.Id().Provider {
				resourcesOfType = append(resourcesOfType, res)
			}
		}
	} else if rule.Classifications != nil {
		for _, res := range dependentResources {
			if e.ClassificationDocument.ResourceContainsClassifications(res, rule.Classifications) {
				resourcesOfType = append(resourcesOfType, res)
			}
		}
	} else {
		return []error{fmt.Errorf("rule must have either a resource type or classifications defined %s for resource %s", rule.String(), resource.Id())}
	}
	switch rule.Enforcement {
	case core.ExactlyOne:
		var res core.Resource
		var ore *core.OperationalResourceError
		if len(resourcesOfType) > 1 {
			return []error{fmt.Errorf("rule with enforcement only_one has more than one resource for rule %s for resource %s", rule.String(), resource.Id())}
		} else if len(resourcesOfType) == 0 {
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
				var oreParent core.Resource
				if !rule.NoParentDependency {
					oreParent = downstreamParent
				}
				ore = &core.OperationalResourceError{
					Resource:   resource,
					Parent:     oreParent,
					Direction:  rule.Direction,
					Count:      1,
					Needs:      needs,
					MustCreate: rule.UnsatisfiedAction.Unique,
					Cause:      fmt.Errorf("rule with enforcement exactly one has less than the required number of resources of type %s  or classifications %s, %d for resource %s", rule.ResourceTypes, rule.Classifications, len(resourcesOfType), resource.Id()),
				}
			case core.ErrorUnsatisfiedResource:
				return []error{fmt.Errorf("rule with enforcement exactly one has less than the required number of resources of type %s  or classifications %s, %d for resource %s", rule.ResourceTypes, rule.Classifications, len(resourcesOfType), resource.Id())}
			}
		} else {
			res = resourcesOfType[0]
			if !rule.RemoveDirectDependency {
				addDependencyForDirection(dag, rule.Direction, resource, res)
			}
			err := setField(dag, resource, rule, res)
			if err != nil {
				return []error{err}
			}
			if downstreamParent != nil && !rule.NoParentDependency {
				addDependencyForDirection(dag, rule.Direction, res, downstreamParent)
			}
		}
		// This has to come before running sub rules since we are not running this rule if its only conditional. Running sub rules first may cause side effects
		if ore != nil {
			return []error{ore}
		}
		var subRuleErrors []error
		for _, subRule := range rule.Rules {
			err := e.handleOperationalRule(resource, subRule, dag, nil)
			if err != nil {
				subRuleErrors = append(subRuleErrors, err...)
			}
		}
		if subRuleErrors != nil {
			return subRuleErrors
		}

		if res == nil {
			return []error{fmt.Errorf("no resources found that can satisfy the operational resource rule %s, for %s for resource %s", rule.String(), resource.Id(), resource.Id())}
		}
		if rule.RemoveDirectDependency {
			if getDependencyForDirection(dag, rule.Direction, resource, res) != nil {
				err := removeDependencyForDirection(dag, rule.Direction, resource, res)
				if err != nil {
					return []error{err}
				}
			}
		}
	case core.Conditional:
		if len(resourcesOfType) == 0 {
			if rule.NumNeeded > 0 {
				return []error{fmt.Errorf("rule with enforcement conditional has less than the required number of resources of type %s  or classifications %s, %d for resource %s", rule.ResourceTypes, rule.Classifications, len(resourcesOfType), resource.Id())}
			}
			return nil
		}
		if len(resourcesOfType) == 1 {
			err := setField(dag, resource, rule, resourcesOfType[0])
			if err != nil {
				return []error{err}
			}
			if rule.RemoveDirectDependency {
				if getDependencyForDirection(dag, rule.Direction, resource, resourcesOfType[0]) != nil {
					err := removeDependencyForDirection(dag, rule.Direction, resource, resourcesOfType[0])
					if err != nil {
						return []error{err}
					}
				}
			}
		}
		var subRuleErrors []error
		for _, subRule := range rule.Rules {
			err := e.handleOperationalRule(resource, subRule, dag, resourcesOfType[0])
			if err != nil {
				subRuleErrors = append(subRuleErrors, err...)
			}
		}
		if subRuleErrors != nil {
			return subRuleErrors
		}
	case core.AnyAvailable:
		var ore *core.OperationalResourceError
		for _, res := range resourcesOfType {
			err := setField(dag, resource, rule, res)
			if err != nil {
				return []error{err}
			}
		}
		if rule.NumNeeded > len(resourcesOfType) {
			switch rule.UnsatisfiedAction.Operation {
			case core.CreateUnsatisfiedResource:
				var needs []string
				if len(resourcesOfType) > 0 {
					var existingTypes []string
					for _, res := range resourcesOfType {
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
				var oreParent core.Resource
				if !rule.NoParentDependency {
					oreParent = downstreamParent
				}
				ore = &core.OperationalResourceError{
					Resource:   resource,
					Parent:     oreParent,
					Direction:  rule.Direction,
					Count:      rule.NumNeeded - len(resourcesOfType),
					MustCreate: rule.UnsatisfiedAction.Unique,
					Needs:      needs,
					Cause:      fmt.Errorf("rule with enforcement any has less than the required number of resources of type %s  or classifications %s, %d for resource %s", rule.ResourceTypes, rule.Classifications, len(resourcesOfType), resource.Id()),
				}
			case core.ErrorUnsatisfiedResource:
				return []error{fmt.Errorf("unsatisfied resource error: rule with enforcement any has less than the required number of resources of type %s  or classifications %s, %d, for resource %s", rule.ResourceTypes, rule.Classifications, len(resourcesOfType), resource.Id())}
			}
		}
		var subRuleErrors []error
		for _, subRule := range rule.Rules {
			err := e.handleOperationalRule(resource, subRule, dag, nil)
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
		if len(resourcesOfType) < rule.NumNeeded {
			return []error{fmt.Errorf("insufficient resource error: rule with enforcement any available has less than the required number of resources of type %s or classifications %s, %d for resource %s", rule.ResourceTypes, rule.Classifications, len(resourcesOfType), resource.Id())}
		}
	default:
		return []error{fmt.Errorf("unknown enforcement type %s, for resource %s", rule.Enforcement, resource.Id())}
	}
	return nil
}

func setField(dag *core.ResourceGraph, resource core.Resource, rule core.OperationalRule, res core.Resource) error {
	copyResource := cloneResource(resource)
	if rule.SetField == "" {
		return nil
	}
	if reflect.ValueOf(resource).Elem().FieldByName(rule.SetField).Kind() == reflect.Slice || reflect.ValueOf(resource).Elem().FieldByName(rule.SetField).Kind() == reflect.Array {
		reflect.ValueOf(resource).Elem().FieldByName(rule.SetField).Set(reflect.Append(reflect.ValueOf(resource).Elem().FieldByName(rule.SetField), reflect.ValueOf(res)))
	} else if reflect.TypeOf(core.ResourceId{}) == reflect.ValueOf(resource).Elem().FieldByName(rule.SetField).Type() {
		reflect.ValueOf(resource).Elem().FieldByName(rule.SetField).Set(reflect.ValueOf(res.Id()))
	} else {
		reflect.ValueOf(resource).Elem().FieldByName(rule.SetField).Set(reflect.ValueOf(res))
	}
	if copyResource.Id() != resource.Id() {
		if dag.GetResource(resource.Id()) != nil {
			return fmt.Errorf("resource %s was replaced with %s, but the original resource still exists in the graph", copyResource.Id(), resource.Id())
		}
		err := dag.ReplaceConstruct(copyResource, resource)
		if err != nil {
			return err
		}
	}
	return nil
}

func cloneResource(resource core.Resource) core.Resource {
	newRes := reflect.New(reflect.TypeOf(resource).Elem()).Interface().(core.Resource)
	for i := 0; i < reflect.ValueOf(newRes).Elem().NumField(); i++ {
		field := reflect.ValueOf(newRes).Elem().Field(i)
		field.Set(reflect.ValueOf(resource).Elem().Field(i))
	}
	return newRes
}

func nameResource(dag *core.ResourceGraph, resourceToSet core.Resource, resource core.Resource, unique bool) {
	numResources := 0
	for _, res := range dag.ListResources() {
		if res.Id().Type == resourceToSet.Id().Type {
			numResources++
		}
	}
	if unique {
		reflect.ValueOf(resourceToSet).Elem().FieldByName("Name").Set(reflect.ValueOf(fmt.Sprintf("%s-%s-%d", resourceToSet.Id().Type, resource.Id().Name, numResources)))
	} else {
		reflect.ValueOf(resourceToSet).Elem().FieldByName("Name").Set(reflect.ValueOf(fmt.Sprintf("%s-%d", resourceToSet.Id().Type, numResources)))
	}
	reflect.ValueOf(resourceToSet).Elem().FieldByName("ConstructRefs").Set(reflect.ValueOf(core.BaseConstructSetOf(resource)))
}

func addDependencyForDirection(dag *core.ResourceGraph, direction core.Direction, resource core.Resource, dependentResource core.Resource) {
	if direction == core.Upstream {
		dag.AddDependency(dependentResource, resource)
	} else {
		dag.AddDependency(resource, dependentResource)
	}
}

func removeDependencyForDirection(dag *core.ResourceGraph, direction core.Direction, resource core.Resource, dependentResource core.Resource) error {
	if direction == core.Upstream {
		return dag.RemoveDependency(dependentResource.Id(), resource.Id())
	} else {
		return dag.RemoveDependency(resource.Id(), dependentResource.Id())
	}
}

func getDependencyForDirection(dag *core.ResourceGraph, direction core.Direction, resource core.Resource, dependentResource core.Resource) *graph.Edge[core.Resource] {
	if direction == core.Upstream {
		return dag.GetDependency(dependentResource.Id(), resource.Id())
	} else {
		return dag.GetDependency(resource.Id(), dependentResource.Id())
	}
}

// handleOperationalResourceError tries to determine how to fix OperatioanlResourceErrors by adding dependencies to the resource graph where needed.
// If the error cannot be fixed, it will return an error.
func (e *Engine) handleOperationalResourceError(err *core.OperationalResourceError, dag *core.ResourceGraph) error {
	resources := e.ListResources()
	// determine the type of resource necessary to satisfy the operational resource error
	var neededResource core.Resource
	for _, res := range resources {

		if e.ClassificationDocument.ResourceContainsClassifications(res, err.Needs) {
			var paths []knowledgebase.Path
			if err.Direction == core.Downstream {
				paths = e.KnowledgeBase.FindPaths(err.Resource, res, knowledgebase.EdgeConstraint{})
			} else {
				paths = e.KnowledgeBase.FindPaths(res, err.Resource, knowledgebase.EdgeConstraint{})
			}
			if len(paths) == 0 {
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
		var resources []core.Resource
		// The direction here is flipped since we are looking at the resources relative to the parent, not relative to the resource used in the error
		if err.Direction == core.Upstream {
			resources = dag.GetAllDownstreamResources(err.Parent)
		} else {
			resources = dag.GetAllUpstreamResources(err.Parent)
		}
		for _, res := range resources {
			if res.Id().Type == neededResource.Id().Type && res.Id().Provider == neededResource.Id().Provider && dag.GetDependency(err.Resource.Id(), res.Id()) == nil {
				addDependencyForDirection(dag, err.Direction, err.Resource, res)
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
	resourceIds := []string{}
	for _, res := range availableResources {
		resourceIds = append(resourceIds, res.Id().Name)
	}
	sort.Strings(resourceIds)

	currNumSatisfied := numSatisfied
	for i := 0; i < err.Count-currNumSatisfied; i++ {
		for _, res := range availableResources {
			if len(resourceIds) > i && res.Id().Name == resourceIds[i] {
				addDependencyForDirection(dag, err.Direction, err.Resource, res)
				numSatisfied++
				break
			}
		}
	}

	// if theres no available resources from us to choose from, we must create new resources
	if len(availableResources) < err.Count-numSatisfied {
		for i := numSatisfied; i < err.Count; i++ {
			newRes := cloneResource(neededResource)
			nameResource(dag, newRes, err.Resource, err.MustCreate)

			addDependencyForDirection(dag, err.Direction, err.Resource, newRes)
			if err.Parent != nil {
				addDependencyForDirection(dag, err.Direction, newRes, err.Parent)
			}
			err := e.MakeResourceOperational(dag, newRes)
			if err != nil {
				return err
			}
			numSatisfied++
		}
	}

	return nil
}

func TemplateConfigure(resource core.Resource, template core.ResourceTemplate, dag *core.ResourceGraph) error {
	for _, config := range template.Configuration {
		field, _, err := parseFieldName(resource, config.Field, dag)
		if err != nil {
			return err
		}
		if (!field.IsValid() || !field.IsZero()) || config.ZeroValueAllowed {
			//since pointers will be non zero but could still be nil we need to check that case before proceeding
			if field.Kind() == reflect.Ptr && !field.IsNil() && !field.Elem().IsZero() {
				continue
			} else if field.Kind() != reflect.Ptr {
				continue
			}
		}
		err = ConfigureField(resource, config.Field, config.Value, config.ZeroValueAllowed, dag)
		if err != nil {
			return err
		}
	}
	return nil
}
