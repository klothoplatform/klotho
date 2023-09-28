package solution_context

import (
	"fmt"
	"reflect"

	construct "github.com/klothoplatform/klotho/pkg/construct2"
	"github.com/klothoplatform/klotho/pkg/engine2/constraints"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base2"
)

func (ctx SolutionContext) LoadConstraints(input map[constraints.ConstraintScope][]constraints.Constraint) error {
	for _, constraint := range input[constraints.ApplicationConstraintScope] {
		err := ctx.ApplyApplicationConstraint(constraint.(*constraints.ApplicationConstraint))
		if err != nil {
			return err
		}
	}

	for _, constraint := range input[constraints.EdgeConstraintScope] {
		err := ctx.ApplyEdgeConstraint(constraint.(*constraints.EdgeConstraint))
		if err != nil {
			return err
		}
		edgeConstraint, ok := constraint.(*constraints.EdgeConstraint)
		if !ok {
			return fmt.Errorf("could not cast constraint to edge constraint")
		}
		ctx.EdgeConstraints = append(ctx.EdgeConstraints, *edgeConstraint)
	}

	for _, constraint := range input[constraints.ConstructConstraintScope] {
		constructConstraint, ok := constraint.(*constraints.ConstructConstraint)
		if !ok {
			return fmt.Errorf("could not cast constraint to construct constraint")
		}
		ctx.ConstructConstraints = append(ctx.ConstructConstraints, *constructConstraint)
	}

	for _, constraint := range input[constraints.ResourceConstraintScope] {
		resourceConstraint, ok := constraint.(*constraints.ResourceConstraint)
		if !ok {
			return fmt.Errorf("could not cast constraint to resource constraint")
		}
		ctx.ResourceConstraints = append(ctx.ResourceConstraints, *resourceConstraint)
	}

	return nil
}

// ApplyApplicationConstraint applies an application constraint to the either the engines working state construct graph
//
// Currently ApplicationConstraints can only be applied if the representing nodes are klotho constructs and not provider level resources
func (ctx SolutionContext) ApplyApplicationConstraint(constraint *constraints.ApplicationConstraint) error {
	ctx.With("constraint", constraint)
	switch constraint.Operator {
	case constraints.AddConstraintOperator:
		res := &construct.Resource{ID: constraint.Node}
		return ctx.addResource(res, false)
	case constraints.RemoveConstraintOperator:
		node, _ := ctx.GetResource(constraint.Node)
		if node == nil {
			return fmt.Errorf("could not find resource %s", constraint.Node)
		}
		return ctx.RemoveResource(node, true)
	case constraints.ReplaceConstraintOperator:
		node, _ := ctx.GetResource(constraint.Node)
		if node == nil {
			return fmt.Errorf("could not find resource %s", constraint.Node)
		}
		var replacementNode *construct.Resource
		if node.ID.QualifiedTypeName() == constraint.ReplacementNode.QualifiedTypeName() {
			replacementNode = cloneResource(node)
			reflect.ValueOf(replacementNode).Elem().FieldByName("Name").Set(reflect.ValueOf(constraint.ReplacementNode.Name))
			return ctx.ReplaceResourceId(constraint.Node, replacementNode)
		} else {
			replacementNode = &construct.Resource{ID: constraint.ReplacementNode}
			functionalUpstream, err := ctx.UpstreamFunctional(node)
			if err != nil {
				return err
			}
			functionalDownstream, err := ctx.DownstreamFunctional(node)
			if err != nil {
				return err
			}
			err = ctx.RemoveResource(node, true)
			if err != nil {
				return err
			}
			for _, res := range functionalUpstream {
				err = ctx.AddDependency(res, replacementNode)
				if err != nil {
					return err
				}
			}
			for _, res := range functionalDownstream {
				err = ctx.AddDependency(replacementNode, res)
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// ApplyEdgeConstraint applies an edge constraint to the either the engines working state construct graph or end state resource graph
//
// The following actions are taken for each operator
// - MustExistConstraintOperator, the edge is added to the working state construct graph
// - MustNotExistConstraintOperator, the edge is removed from the working state construct graph if the source and targets refer to klotho constructs. Otherwise the action fails
// - MustContainConstraintOperator, the constraint is applied to the edge before edge expansion, so when we use the knowledgebase to expand it ensures the node in the constraint is present in the expanded path
// - MustNotContainConstraintOperator, the constraint is applied to the edge before edge expansion, so when we use the knowledgebase to expand it ensures the node in the constraint is not present in the expanded path
func (ctx SolutionContext) ApplyEdgeConstraint(constraint *constraints.EdgeConstraint) error {
	ctx.With("constraint", constraint)
	src, _ := ctx.GetResource(constraint.Target.Source)
	if src == nil {
		src = &construct.Resource{ID: constraint.Target.Source}
	}
	dst, _ := ctx.GetResource(constraint.Target.Target)
	if dst == nil {
		dst = &construct.Resource{ID: constraint.Target.Target}
	}

	switch constraint.Operator {
	case constraints.MustExistConstraintOperator:
		err := ctx.AddDependency(src, dst)
		if err != nil {
			return err
		}
	case constraints.MustNotExistConstraintOperator:

		paths, err := ctx.AllPaths(constraint.Target.Source, constraint.Target.Target)
		if err != nil {
			return err
		}

		// first we will remove all dependencies that make up the paths from the constraints source to target
		for _, path := range paths {
			var prevRes *construct.Resource
			for _, res := range path {
				if prevRes != nil {
					err := ctx.RemoveDependency(prevRes.ID, res.ID)
					if err != nil {
						return err
					}
				}
				prevRes = res
			}
		}

		// Next we will try to delete any node in those paths in case they no longer are required for the architecture
		// We will pass the explicit field as false so that explicitly added resources do not get deleted
		for _, path := range paths {
			for _, res := range path {
				resource, _ := ctx.GetResource(res.ID)
				if resource != nil {
					err := ctx.RemoveResource(resource, false)
					if err != nil {
						return err
					}
				}
			}
		}
	}
	return nil
}

func (ctx SolutionContext) ApplyResourceConstraint(resource *construct.Resource, rc constraints.ResourceConstraint) error {
	ctx.With("constraint", rc)
	configuration := knowledgebase.Configuration{
		Field: rc.Property,
		Value: rc.Value,
	}
	switch rc.Operator {
	case constraints.AddConstraintOperator:
		return ctx.ConfigureResource(resource, configuration, knowledgebase.ConfigTemplateData{Resource: resource.ID}, "add")
	case constraints.RemoveConstraintOperator:
		return ctx.ConfigureResource(resource, configuration, knowledgebase.ConfigTemplateData{Resource: resource.ID}, "remove")
	case constraints.EqualsConstraintOperator:
		return ctx.ConfigureResource(resource, configuration, knowledgebase.ConfigTemplateData{Resource: resource.ID}, "set")
	default:
		return fmt.Errorf("invalid operator %s", rc.Operator)
	}
}

func cloneResource(resource *construct.Resource) *construct.Resource {
	newRes := reflect.New(reflect.TypeOf(resource).Elem()).Interface().(construct.Resource)
	for i := 0; i < reflect.ValueOf(newRes).Elem().NumField(); i++ {
		field := reflect.ValueOf(newRes).Elem().Field(i)
		field.Set(reflect.ValueOf(resource).Elem().Field(i))
	}
	return &newRes
}
