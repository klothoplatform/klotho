package operational_rule

import (
	"fmt"
	"reflect"

	"github.com/klothoplatform/klotho/pkg/collectionutil"
	construct "github.com/klothoplatform/klotho/pkg/construct2"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base2"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

type (
	OperationalResourceAction struct {
		Step       *knowledgebase.OperationalStep
		CurrentIds []construct.ResourceId
	}
)

func (ctx OperationalRuleContext) HandleOperationalStep(step *knowledgebase.OperationalStep) error {
	// Default to 1 resource needed
	if step.NumNeeded == 0 {
		step.NumNeeded = 1
	}

	resourceId, err := ctx.ConfigCtx.ExecuteDecodeAsResourceId(step.Resource, ctx.Data)
	if err != nil {
		return err
	}
	resource, _ := ctx.Graph.GetResource(resourceId)
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
			err := ctx.clearProperty(step, resource, ctx.Property.Path)
			if err != nil {
				return err
			}
		}
		var err error
		ids, err = ctx.addDependenciesFromProperty(step, resource, ctx.Property.Path)
		if err != nil {
			return err
		}
	} else {
		ids, err = ctx.getResourcesForStep(step, resource)
		if err != nil {
			return err
		}
		if replace {
			for _, id := range ids {
				err := ctx.Graph.RemoveDependency(id, resource.ID)
				if err != nil {
					return err
				}
			}
		}
		ids = []construct.ResourceId{}
	}

	if len(ids) > step.NumNeeded {
		return nil
	}

	if step.FailIfMissing {
		return errors.Errorf("operational resource error for %s", resource.ID)
	}

	action := OperationalResourceAction{
		Step:       step,
		CurrentIds: ids,
	}
	return ctx.handleOperationalResourceAction(resource, action)
}

func (ctx OperationalRuleContext) handleOperationalResourceAction(resource *construct.Resource, action OperationalResourceAction) error {
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
		res, _ := ctx.Graph.GetResource(explicitResource)
		if res == nil {
			res = construct.CreateResource(explicitResource)
		}
		err := ctx.addDependencyForDirection(action.Step, resource, res)
		if err != nil {
			return err
		}
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
			newRes := construct.CreateResource(typeToCreate)
			ctx.generateResourceName(newRes, resource, action.Step.Unique)
			err := ctx.addDependencyForDirection(action.Step, resource, newRes)
			if err != nil {
				return err
			}
			numNeeded--
		}
	}

	for _, typeToCreate := range resourceTypes {

		namespacedIds, err := ctx.KB.GetAllowedNamespacedResourceIds(ctx.ConfigCtx, typeToCreate)
		if err != nil {
			return err
		}
		var namespaceResourcesForResource []*construct.Resource
		for _, namespacedId := range namespacedIds {
			if ctx.KB.HasFunctionalPath(resource.ID, namespacedId) {
				downstreams, err := ctx.Graph.DownstreamOfType(resource, 3, namespacedId.QualifiedTypeName())
				if err != nil {
					return err
				}
				namespaceResourcesForResource = append(namespaceResourcesForResource, downstreams...)
			}
		}

		var availableResources []*construct.Resource
		resources, err := ctx.Graph.ListResources()
		if err != nil {
			return err
		}
		for _, res := range resources {
			if collectionutil.Contains(action.CurrentIds, res.ID) {
				continue
			}
			if res.ID.QualifiedTypeName() == typeToCreate.QualifiedTypeName() {
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
			err := ctx.addDependencyForDirection(action.Step, resource, res)
			if err != nil {
				return err
			}
			numNeeded--
		}
	}

	for numNeeded > 0 {
		typeToCreate := resourceTypes[0]
		newRes := construct.CreateResource(typeToCreate)
		ctx.generateResourceName(newRes, resource, action.Step.Unique)
		err := ctx.addDependencyForDirection(action.Step, resource, newRes)
		if err != nil {
			return err
		}
		numNeeded--
	}

	return nil
}

func (ctx OperationalRuleContext) findResourcesWhichSatisfyStepClassifications(step *knowledgebase.OperationalStep, resource *construct.Resource) []construct.ResourceId {
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
			hasPath = ctx.KB.HasFunctionalPath(resource.ID, res.Id())
		} else {
			hasPath = ctx.KB.HasFunctionalPath(res.Id(), resource.ID)
		}
		// if a type is explicilty stated as needed, we will consider it even if there isnt a direct p
		if !hasPath {
			continue
		}
		result = append(result, res.Id())
	}
	return result
}

func (ctx OperationalRuleContext) shouldReplace(step *knowledgebase.OperationalStep) (bool, error) {
	if step.ReplacementCondition != "" {
		result := false
		err := ctx.ConfigCtx.ExecuteDecode(step.ReplacementCondition, ctx.Data, &result)
		if err != nil {
			return result, err
		}
		return result, nil
	}
	return false, nil
}

func (ctx OperationalRuleContext) getResourcesForStep(step *knowledgebase.OperationalStep, resource *construct.Resource) ([]construct.ResourceId, error) {
	var dependentResources []*construct.Resource
	var resourcesOfType []construct.ResourceId
	var err error
	if step.Direction == knowledgebase.Upstream {
		dependentResources, err = ctx.Graph.Upstream(resource, 3)
		if err != nil {
			return nil, err
		}
	} else {
		dependentResources, err = ctx.Graph.Downstream(resource, 3)
		if err != nil {
			return nil, err
		}
	}
	if step.Resources != nil {
		for _, res := range dependentResources {
			if collectionutil.Contains(step.Resources, res.ID.QualifiedTypeName()) {
				resourcesOfType = append(resourcesOfType, res.ID)
			}
		}
	} else if step.Classifications != nil {
		for _, res := range dependentResources {
			resTemplate, err := ctx.KB.GetResourceTemplate(res.ID)
			if err != nil {
				return nil, err
			}
			if resTemplate.ResourceContainsClassifications(step.Classifications) {
				resourcesOfType = append(resourcesOfType, res.ID)
			}
		}
	}
	return resourcesOfType, nil
}

func (ctx OperationalRuleContext) addDependenciesFromProperty(step *knowledgebase.OperationalStep, resource *construct.Resource, propertyName string) ([]construct.ResourceId, error) {

	val, err := resource.GetProperty(propertyName)
	if err != nil {
		return nil, err
	}
	field := reflect.ValueOf(val)
	if field.IsValid() {
		if field.Kind() == reflect.Slice || field.Kind() == reflect.Array {
			var ids []construct.ResourceId
			for i := 0; i < field.Len(); i++ {
				val := field.Index(i)
				err := ctx.addDependencyForDirection(step, resource, val.Interface().(*construct.Resource))
				if err != nil {
					return []construct.ResourceId{}, err
				}
				ids = append(ids, val.Interface().(*construct.Resource).ID)
			}
			return ids, nil
		} else if field.Kind() == reflect.Ptr && !field.IsNil() {
			val := field
			err := ctx.addDependencyForDirection(step, resource, val.Interface().(*construct.Resource))
			if err != nil {
				return []construct.ResourceId{}, err
			}
			return []construct.ResourceId{val.Interface().(*construct.Resource).ID}, nil
		}
	}
	return nil, nil
}

func (ctx OperationalRuleContext) clearProperty(step *knowledgebase.OperationalStep, resource *construct.Resource, propertyName string) error {
	val, err := resource.GetProperty(propertyName)
	if err != nil {
		return err
	}
	field := reflect.ValueOf(val)
	if field.IsValid() {
		if field.Kind() == reflect.Slice || field.Kind() == reflect.Array {
			for i := 0; i < field.Len(); i++ {
				val := field.Index(i)
				err := ctx.removeDependencyForDirection(step.Direction, resource, val.Interface().(*construct.Resource))
				if err != nil {
					return err
				}
			}
			err := resource.SetProperty(propertyName, reflect.MakeSlice(field.Type(), 0, 0).Interface())
			if err != nil {
				return fmt.Errorf("error clearing property %s on resource %s: %w", propertyName, resource.ID, err)
			}
		} else if field.Kind() == reflect.Ptr && !field.IsNil() {
			val := field
			err := ctx.removeDependencyForDirection(step.Direction, resource, val.Interface().(*construct.Resource))
			if err != nil {
				return err
			}
			err = resource.SetProperty(propertyName, reflect.Zero(field.Type()).Interface())
			if err != nil {
				return fmt.Errorf("error clearing property %s on resource %s: %w", propertyName, resource.ID, err)
			}
		}
	}
	return nil
}

func (ctx OperationalRuleContext) addDependencyForDirection(step *knowledgebase.OperationalStep, resource, dependentResource *construct.Resource) error {
	if step.Direction == knowledgebase.Upstream {
		err := ctx.Graph.AddDependency(dependentResource, resource)
		if err != nil {
			return err
		}
		return ctx.setField(resource, dependentResource, step)
	} else {
		err := ctx.Graph.AddDependency(resource, dependentResource)
		if err != nil {
			return err
		}
		return ctx.setField(resource, dependentResource, step)
	}
}

func (ctx OperationalRuleContext) removeDependencyForDirection(direction knowledgebase.Direction, resource, dependentResource *construct.Resource) error {
	if direction == knowledgebase.Upstream {
		return ctx.Graph.RemoveDependency(dependentResource.ID, resource.ID)
	} else {
		return ctx.Graph.RemoveDependency(resource.ID, dependentResource.ID)
	}
}

func (ctx OperationalRuleContext) generateResourceName(resourceToSet, resource *construct.Resource, unique bool) {
	numResources := 0
	resources, err := ctx.Graph.ListResources()
	if err != nil {
		return
	}
	for _, res := range resources {
		if res.ID.Type == resourceToSet.ID.Type {
			numResources++
		}
	}
	if unique {
		resourceToSet.ID.Name = fmt.Sprintf("%s-%s-%d", resourceToSet.ID.Type, resource.ID.Name, numResources)
	} else {
		resourceToSet.ID.Name = fmt.Sprintf("%s-%d", resourceToSet.ID.Type, numResources)
	}
}

func (ctx OperationalRuleContext) setField(resource, fieldResource *construct.Resource, step *knowledgebase.OperationalStep) error {
	if ctx.Property == nil {
		return nil
	}
	// snapshot the ID from before any field changes
	oldId := resource.ID

	if ctx.Property.IsPropertyTypeScalar() {
		res, err := resource.GetProperty(ctx.Property.Path)
		if err != nil {
			zap.S().Debugf("property %s not found on resource %s", ctx.Property.Path, resource.ID)
		}
		// If the current field is a resource id we will compare it against the one passed in to see if we need to remove the current resource
		currResId, ok := res.(construct.ResourceId)
		if ok {
			if res != nil && res != fieldResource.ID {
				oldPropertyResource, err := ctx.Graph.GetResource(currResId)
				if err != nil {
					return err
				}
				err = ctx.removeDependencyForDirection(step.Direction, resource, oldPropertyResource)
				if err != nil {
					return err
				}
				zap.S().Infof("Removing old field value for '%s' (%s) for %s", ctx.Property.Path, res, fieldResource.ID)
				// Remove the old field value if it's unused
				err = ctx.Graph.RemoveResource(oldPropertyResource, false)
				if err != nil {
					return err
				}
			}
		}
		// Right now we only enforce the top level properties if they have rules, so we can assume the path is equal to the name of the property
		err = resource.SetProperty(ctx.Property.Path, fieldResource.ID)
		if err != nil {
			return fmt.Errorf("error setting field %s#%s with %s: %w", resource.ID, ctx.Property.Path, fieldResource.ID, err)
		}
		zap.S().Infof("set field %s#%s to %s", resource.ID, ctx.Property.Path, fieldResource.ID)
		// See if we need to namespace the resource due to setting the property
		if ctx.Property.Namespace {
			resource.ID.Namespace = fieldResource.ID.Name
		}
	} else {
		// Right now we only enforce the top level properties if they have rules, so we can assume the path is equal to the name of the property
		err := resource.AppendProperty(ctx.Property.Path, []construct.ResourceId{fieldResource.ID})
		if err != nil {
			return fmt.Errorf("error appending field %s#%s with %s: %w", resource.ID, ctx.Property.Path, fieldResource.ID, err)
		}
		zap.S().Infof("appended field %s#%s with %s", resource.ID, ctx.Property.Path, fieldResource.ID)
	}

	// If this sets the field driving the namespace, for example,
	// then the Id could change, so replace the resource in the graph
	// to update all the edges to the new Id.
	if oldId != resource.ID {
		err := ctx.Graph.ReplaceResourceId(oldId, resource.ID)
		if err != nil {
			return err
		}
	}
	return nil
}
