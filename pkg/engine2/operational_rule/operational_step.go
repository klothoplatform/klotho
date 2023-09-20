package operational_rule

import (
	"fmt"
	"reflect"

	"github.com/klothoplatform/klotho/pkg/collectionutil"
	"github.com/klothoplatform/klotho/pkg/construct"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base2"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

type (
	OperationalResourceAction struct {
		Step       knowledgebase.OperationalStep
		CurrentIds []construct.ResourceId
	}
)

func (ctx OperationalRuleContext) HandleOperationalStep(step knowledgebase.OperationalStep) error {
	// Default to 1 resource needed
	if step.NumNeeded == 0 {
		step.NumNeeded = 1
	}

	resourceId, err := ctx.ConfigCtx.ExecuteDecodeAsResourceId(step.Resource, knowledgebase.ConfigTemplateData{})
	if err != nil {
		return err
	}
	resource := ctx.Graph.GetResource(resourceId)
	if resource == nil {
		return errors.Errorf("resource %s not found", resourceId)
	}

	replace, err := ctx.shouldReplace(step)
	if err != nil {
		return err
	}

	// If we are replacing we want to remove all dependencies and clear the property
	// otherwise we want to add dependencies from the property and gather the resources which satisfy the step
	var ids []construct.ResourceId
	if ctx.Property != nil {
		if replace {
			ctx.clearProperty(step, resource, ctx.Property.Name)
		}
		ids = ctx.addDependenciesFromProperty(step, resource, ctx.Property.Name)
	} else {
		ids, err = ctx.getResourcesForStep(step, resource)
		if err != nil {
			return err
		}
		if replace {
			for _, id := range ids {
				ctx.Graph.RemoveDependency(id, resource.Id())
			}
		}
		ids = []construct.ResourceId{}
	}

	if len(ids) > step.NumNeeded {
		return nil
	}

	if step.FailIfMissing {
		return errors.Errorf("operational resource error for %s", resource.Id())
	}

	action := OperationalResourceAction{
		Step:       step,
		CurrentIds: ids,
	}
	ctx.handleOperationalResourceAction(resource, action)
	return nil
}

func (ctx OperationalRuleContext) handleOperationalResourceAction(resource construct.Resource, action OperationalResourceAction) error {
	numNeeded := action.Step.NumNeeded - len(action.CurrentIds)
	if numNeeded <= 0 {
		return nil
	}

	explicitResources, resourceTypes, err := action.Step.ExtractResourcesAndTypes(ctx.ConfigCtx, ctx.Data)
	if err != nil {
		return err
	}

	// Add explicitly named resources first
	for _, explicitResource := range explicitResources {
		if numNeeded <= 0 {
			return nil
		}
		res := ctx.Graph.GetResource(explicitResource)
		if res == nil {
			res = ctx.CreateResourcefromId(explicitResource)
		}
		ctx.addDependencyForDirection(action.Step.Direction, resource, res)
		numNeeded--
	}
	if numNeeded <= 0 {
		return nil
	}

	// If the rule contains classifications, we are going to get the resource types which satisfy those and put it onto the list of applicable resource types
	if len(action.Step.Classifications) > 0 {
		resourceTypes = append(resourceTypes, ctx.findResourcesWhichSatisfyStepClassifications(action.Step, resource)...)
	}

	// If there are no resource types, we can't do anything since we dont understand what resources will satisfy the rule
	if len(resourceTypes) == 0 {
		return errors.Errorf("no resources found that can satisfy the operational resource error")
	}

	if action.Step.Unique {
		// loop over the number of resources still needed and create them if the unique flag is true
		for numNeeded > 0 {
			typeToCreate := resourceTypes[0]
			newRes := ctx.CreateResourcefromId(typeToCreate)
			ctx.generateResourceName(newRes, resource, action.Step.Unique)
			ctx.addDependencyForDirection(action.Step.Direction, resource, newRes)
			numNeeded--
		}
	}

	for _, typeToCreate := range resourceTypes {

		namespacedIds, err := ctx.KB.GetAllowedNamespacedResourceIds(ctx.ConfigCtx, typeToCreate)
		if err != nil {
			return err
		}
		var namespaceResourcesForResource []construct.Resource
		for _, namespacedId := range namespacedIds {
			if ctx.KB.HasFunctionalPath(resource.Id(), namespacedId) {
				namespaceResourcesForResource = append(namespaceResourcesForResource,
					ctx.Graph.GetFunctionalDownstreamResourcesOfType(resource, namespacedId)...)
			}
		}

		var availableResources []construct.Resource
		for _, res := range ctx.Graph.ListResources() {
			if collectionutil.Contains(action.CurrentIds, res.Id()) {
				continue
			}
			if res.Id().QualifiedTypeName() == typeToCreate.QualifiedTypeName() {
				namespaceResource := ctx.KB.GetResourcesNamespaceResource(res)
				// needed resource is not namespaced or resource doesnt have any namespace types downstream or the namespaced resource is using the right namespace
				if len(namespacedIds) == 0 || len(namespaceResourcesForResource) == 0 || collectionutil.Contains(namespaceResourcesForResource, namespaceResource) {
					availableResources = append(availableResources, res)
				}
			}
		}

		// TODO: Here we should evaluate resources based on the operator, so spread, etc so that we can order the selection of resources
		for _, res := range availableResources {
			if numNeeded <= 0 {
				return nil
			}
			ctx.addDependencyForDirection(action.Step.Direction, resource, res)
			numNeeded--
		}
	}

	for numNeeded > 0 {
		typeToCreate := resourceTypes[0]
		newRes := ctx.CreateResourcefromId(typeToCreate)
		ctx.generateResourceName(newRes, resource, action.Step.Unique)
		ctx.addDependencyForDirection(action.Step.Direction, resource, newRes)
		numNeeded--
	}

	return nil
}

func (ctx OperationalRuleContext) findResourcesWhichSatisfyStepClassifications(step knowledgebase.OperationalStep, resource construct.Resource) []construct.ResourceId {
	// determine the type of resource necessary to satisfy the operational resource error
	var result []construct.ResourceId
	for _, res := range ctx.KB.ListResources() {
		resTempalte, err := ctx.KB.GetResourceTemplate(res.Id())
		if err != nil {
			continue
		}
		if !resTempalte.ResourceContainsClassifications(step.Classifications) {
			continue
		}
		var hasPath bool
		if step.Direction == knowledgebase.Downstream {
			hasPath = ctx.KB.HasFunctionalPath(resource.Id(), res.Id())
		} else {
			hasPath = ctx.KB.HasFunctionalPath(res.Id(), resource.Id())
		}
		// if a type is explicilty stated as needed, we will consider it even if there isnt a direct p
		if !hasPath {
			continue
		}
		result = append(result, res.Id())
	}
	return result
}

func (ctx OperationalRuleContext) shouldReplace(step knowledgebase.OperationalStep) (bool, error) {
	if step.ReplacementCondition != "" {
		result := false
		data := knowledgebase.ConfigTemplateData{}
		err := ctx.ConfigCtx.ExecuteDecode(step.ReplacementCondition, data, &result)
		if err != nil {
			return result, err
		}
		return result, nil
	}
	return false, nil
}

func (ctx OperationalRuleContext) getResourcesForStep(step knowledgebase.OperationalStep, resource construct.Resource) ([]construct.ResourceId, error) {
	var dependentResources []construct.Resource
	var resourcesOfType []construct.ResourceId
	if step.Direction == knowledgebase.Upstream {
		dependentResources = ctx.Graph.GetFunctionalUpstreamResources(resource)
	} else {
		dependentResources = ctx.Graph.GetFunctionalDownstreamResources(resource)
	}
	if step.Resources != nil {
		for _, res := range dependentResources {
			if collectionutil.Contains(step.Resources, res.Id().QualifiedTypeName()) {
				resourcesOfType = append(resourcesOfType, res.Id())
			}
		}
	} else if step.Classifications != nil {
		for _, res := range dependentResources {
			resTemplate, err := ctx.KB.GetResourceTemplate(res.Id())
			if err != nil {
				return nil, err
			}
			if resTemplate.ResourceContainsClassifications(step.Classifications) {
				resourcesOfType = append(resourcesOfType, res.Id())
			}
		}
	}
	return resourcesOfType, nil
}

func (ctx OperationalRuleContext) addDependenciesFromProperty(step knowledgebase.OperationalStep, resource construct.Resource, propertyName string) []construct.ResourceId {
	field := reflect.ValueOf(resource).Elem().FieldByName(propertyName)
	if field.IsValid() {
		if field.Kind() == reflect.Slice || field.Kind() == reflect.Array {
			var ids []construct.ResourceId
			for i := 0; i < field.Len(); i++ {
				val := field.Index(i)
				ctx.addDependencyForDirection(step.Direction, resource, val.Interface().(construct.Resource))
				ids = append(ids, val.Interface().(construct.Resource).Id())
			}
			return ids
		} else if field.Kind() == reflect.Ptr && !field.IsNil() {
			val := field
			ctx.addDependencyForDirection(step.Direction, resource, val.Interface().(construct.Resource))
			return []construct.ResourceId{val.Interface().(construct.Resource).Id()}
		}
	}
	return nil
}

func (ctx OperationalRuleContext) clearProperty(step knowledgebase.OperationalStep, resource construct.Resource, propertyName string) {
	field := reflect.ValueOf(resource).Elem().FieldByName(propertyName)
	if field.IsValid() {
		if field.Kind() == reflect.Slice || field.Kind() == reflect.Array {
			for i := 0; i < field.Len(); i++ {
				val := field.Index(i)
				ctx.removeDependencyForDirection(step.Direction, resource, val.Interface().(construct.Resource))
			}
			field.Set(reflect.MakeSlice(field.Type(), 0, 0))
		} else if field.Kind() == reflect.Ptr && !field.IsNil() {
			val := field
			ctx.removeDependencyForDirection(step.Direction, resource, val.Interface().(construct.Resource))
			field.Set(reflect.Zero(field.Type()))
		}
	}
}

func (ctx OperationalRuleContext) addDependencyForDirection(direction knowledgebase.Direction, resource construct.Resource, dependentResource construct.Resource) {
	if direction == knowledgebase.Upstream {
		ctx.Graph.AddDependency(dependentResource, resource)
	} else {
		ctx.Graph.AddDependency(resource, dependentResource)
	}
}

func (ctx OperationalRuleContext) removeDependencyForDirection(direction knowledgebase.Direction, resource construct.Resource, dependentResource construct.Resource) error {
	if direction == knowledgebase.Upstream {
		return ctx.Graph.RemoveDependency(dependentResource.Id(), resource.Id())
	} else {
		return ctx.Graph.RemoveDependency(resource.Id(), dependentResource.Id())
	}
}

func (ctx OperationalRuleContext) generateResourceName(resourceToSet construct.Resource, resource construct.Resource, unique bool) {
	numResources := 0
	for _, res := range ctx.Graph.ListResources() {
		if res.Id().Type == resourceToSet.Id().Type {
			numResources++
		}
	}
	if unique {
		reflect.ValueOf(resourceToSet).Elem().FieldByName("Name").Set(reflect.ValueOf(fmt.Sprintf("%s-%s-%d", resourceToSet.Id().Type, resource.Id().Name, numResources)))
	} else {
		reflect.ValueOf(resourceToSet).Elem().FieldByName("Name").Set(reflect.ValueOf(fmt.Sprintf("%s-%d", resourceToSet.Id().Type, numResources)))
	}
}

func (ctx OperationalRuleContext) setField(resource construct.Resource, fieldResource construct.Resource, step knowledgebase.OperationalStep) error {
	if ctx.Property == nil {
		return nil
	}
	// snapshot the ID from before any field changes
	oldId := resource.Id()

	resVal := reflect.ValueOf(resource)
	fieldValue := reflect.ValueOf(fieldResource)

	field := resVal.Elem().FieldByName(ctx.Property.Name)

	if field.Kind() == reflect.Slice || field.Kind() == reflect.Array {
		field.Set(reflect.Append(field, fieldValue))
	} else {
		if field.Kind() == reflect.Ptr && !field.IsNil() {
			oldFieldValue := field.Interface()
			if oldRes, ok := oldFieldValue.(construct.Resource); ok && fieldResource.Id() != oldRes.Id() {
				err := ctx.removeDependencyForDirection(step.Direction, resource, oldRes)
				if err != nil {
					return err
				}
				zap.S().Infof("Removing old field value for '%s' (%s) for %s", ctx.Property.Name, oldRes.Id(), fieldResource.Id())
				// Remove the old field value if it's unused
				err = ctx.Graph.RemoveResource(oldRes, false)
				if err != nil {
					return err
				}
			}
		}

		if reflect.TypeOf(construct.ResourceId{}).AssignableTo(field.Type()) {
			field.Set(reflect.ValueOf(fieldResource.Id()))
		} else {
			field.Set(fieldValue)
		}
	}
	zap.S().Infof("set field %s#%s to %s", resource.Id(), ctx.Property.Name, fieldResource.Id())
	// If this sets the field driving the namespace, for example,
	// then the Id could change, so replace the resource in the graph
	// to update all the edges to the new Id.
	if oldId != resource.Id() {
		err := ctx.Graph.ReplaceResourceId(oldId, resource)
		if err != nil {
			return err
		}
	}
	return nil
}
