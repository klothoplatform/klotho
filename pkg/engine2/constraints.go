package engine2

import (
	"errors"
	"fmt"

	"github.com/dominikbraun/graph"
	construct "github.com/klothoplatform/klotho/pkg/construct2"
	"github.com/klothoplatform/klotho/pkg/engine2/constraints"
	"github.com/klothoplatform/klotho/pkg/engine2/reconciler"
	"github.com/klothoplatform/klotho/pkg/engine2/solution_context"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base2"
)

func ApplyConstraints(ctx solution_context.SolutionContext) error {
	var errs error
	for _, constraint := range ctx.Constraints().Application {
		errs = errors.Join(errs, applyApplicationConstraint(ctx, constraint))
	}
	if errs != nil {
		return errs
	}

	for _, constraint := range ctx.Constraints().Edges {
		errs = errors.Join(errs, applyEdgeConstraint(ctx, constraint))
	}
	if errs != nil {
		return errs
	}

	resourceConstraints := ctx.Constraints().Resources
	for i := range resourceConstraints {
		errs = errors.Join(errs, applySanitization(ctx, &resourceConstraints[i]))
	}

	return nil
}

// applyApplicationConstraint returns a resource to be made operational, if needed. Otherwise, it returns nil.
func applyApplicationConstraint(ctx solution_context.SolutionContext, constraint constraints.ApplicationConstraint) error {
	ctx = ctx.With("constraint", constraint)

	res, err := knowledgebase.CreateResource(ctx.KnowledgeBase(), constraint.Node)
	if err != nil {
		return err
	}

	switch constraint.Operator {
	case constraints.AddConstraintOperator:
		return ctx.OperationalView().AddVertex(res)

	case constraints.RemoveConstraintOperator:
		return reconciler.RemoveResource(ctx, res.ID, true)

	case constraints.ReplaceConstraintOperator:
		node, err := ctx.RawView().Vertex(res.ID)
		if err != nil {
			return fmt.Errorf("could not find resource for %s: %w", res.ID, err)
		}
		if node.ID.QualifiedTypeName() == constraint.ReplacementNode.QualifiedTypeName() {
			rt, err := ctx.KnowledgeBase().GetResourceTemplate(constraint.ReplacementNode)
			if err != nil {
				return err
			}
			constraint.ReplacementNode.Name, err = rt.SanitizeName(constraint.ReplacementNode.Name)
			if err != nil {
				return err
			}
			return ctx.OperationalView().UpdateResourceID(res.ID, constraint.ReplacementNode)
		} else {
			replacement, err := knowledgebase.CreateResource(ctx.KnowledgeBase(), constraint.ReplacementNode)
			if err != nil {
				return err
			}
			return construct.ReplaceResource(ctx.OperationalView(), res.ID, replacement)
		}

	default:
		return fmt.Errorf("unknown operator %s", constraint.Operator)
	}
}

// applyEdgeConstraint applies an edge constraint to the either the engines working state construct graph or end state resource graph
//
// The following actions are taken for each operator
// - MustExistConstraintOperator, the edge is added to the working state construct graph
// - MustNotExistConstraintOperator, the edge is removed from the working state construct graph if the source and targets refer to klotho constructs. Otherwise the action fails
//
// The following operators are handled during path selection, so any existing paths must be
// - MustContainConstraintOperator, the constraint is applied to the edge before edge expansion, so when we use the knowledgebase to expand it ensures the node in the constraint is present in the expanded path
// - MustNotContainConstraintOperator, the constraint is applied to the edge before edge expansion, so when we use the knowledgebase to expand it ensures the node in the constraint is not present in the expanded path
func applyEdgeConstraint(ctx solution_context.SolutionContext, constraint constraints.EdgeConstraint) error {
	ctx = ctx.With("constraint", constraint)

	for _, id := range []*construct.ResourceId{&constraint.Target.Source, &constraint.Target.Target} {
		rt, err := ctx.KnowledgeBase().GetResourceTemplate(*id)
		if err != nil {
			res := "source"
			if *id == constraint.Target.Target {
				res = "target"
			}
			return fmt.Errorf("could not get template for %s: %w", res, err)
		}
		(*id).Name, err = rt.SanitizeName((*id).Name)
		if err != nil {
			res := "source"
			if *id == constraint.Target.Target {
				res = "target"
			}
			return fmt.Errorf("could not sanitize %s name: %w", res, err)
		}
	}

	addPath := func() error {
		switch _, err := ctx.RawView().Vertex(constraint.Target.Source); {
		case errors.Is(err, graph.ErrVertexNotFound):
			res, err := knowledgebase.CreateResource(ctx.KnowledgeBase(), constraint.Target.Source)
			if err != nil {
				return fmt.Errorf("could not create source resource: %w", err)
			}
			err = ctx.OperationalView().AddVertex(res)
			if err != nil {
				return fmt.Errorf("could not add source resource %s: %w", constraint.Target.Source, err)
			}

		case err != nil:
			return fmt.Errorf("could not get source resource %s: %w", constraint.Target.Source, err)
		}

		switch _, err := ctx.RawView().Vertex(constraint.Target.Target); {
		case errors.Is(err, graph.ErrVertexNotFound):
			res, err := knowledgebase.CreateResource(ctx.KnowledgeBase(), constraint.Target.Target)
			if err != nil {
				return fmt.Errorf("could not create target resource: %w", err)
			}
			err = ctx.OperationalView().AddVertex(res)
			if err != nil {
				return fmt.Errorf("could not add target resource %s: %w", constraint.Target.Target, err)
			}

		case err != nil:
			return fmt.Errorf("could not get target resource %s: %w", constraint.Target.Target, err)
		}

		return ctx.OperationalView().AddEdge(constraint.Target.Source, constraint.Target.Target)
	}

	removePath := func() error {
		paths, err := graph.AllPathsBetween(ctx.DataflowGraph(), constraint.Target.Source, constraint.Target.Target)
		switch {
		case errors.Is(err, graph.ErrTargetNotReachable):
			return nil
		case err != nil:
			return err
		}

		var errs error

		// first we will remove all dependencies that make up the paths from the constraints source to target
		for _, path := range paths {
			for i, res := range path {
				if i == 0 {
					continue
				}
				errs = errors.Join(errs, ctx.OperationalView().RemoveEdge(path[i-1], res))
			}
		}
		if errs != nil {
			return errs
		}

		// Next we will try to delete any node in those paths in case they no longer are required for the architecture
		// We will pass the explicit field as false so that explicitly added resources do not get deleted
		for _, path := range paths {
			for _, resource := range path {
				errs = errors.Join(errs, reconciler.RemoveResource(ctx, resource, false))
			}
		}
		return errs
	}

	switch constraint.Operator {
	case constraints.MustExistConstraintOperator:
		return addPath()

	case constraints.MustNotExistConstraintOperator:
		return removePath()
	}
	return nil
}

// applySanitization applies sanitization to the resource name in ResourceConstraints. This is not needed on
// Application or Edge constraints due to them applying within the graph (to make sure that even generated resources
// are sanitized).
func applySanitization(ctx solution_context.SolutionContext, constraint *constraints.ResourceConstraint) error {
	rt, err := ctx.KnowledgeBase().GetResourceTemplate(constraint.Target)
	if err != nil {
		return err
	}
	constraint.Target.Name, err = rt.SanitizeName(constraint.Target.Name)
	return err
}
