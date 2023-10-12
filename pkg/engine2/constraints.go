package engine2

import (
	"errors"
	"fmt"

	"github.com/dominikbraun/graph"
	construct "github.com/klothoplatform/klotho/pkg/construct2"
	"github.com/klothoplatform/klotho/pkg/engine2/constraints"
	"github.com/klothoplatform/klotho/pkg/engine2/reconciler"
	"github.com/klothoplatform/klotho/pkg/engine2/solution_context"
)

func ApplyConstraints(ctx solution_context.SolutionContext) error {
	op := ctx.OperationalView()

	var resources []*construct.Resource
	var errs error
	for _, constraint := range ctx.Constraints().Application {
		res, err := applyApplicationConstraint(ctx, constraint)
		errs = errors.Join(errs, err)
		if res != nil {
			resources = append(resources, res)
		}
	}
	if errs != nil {
		return errs
	}
	idChangeResult, err := op.MakeResourcesOperational(resources)
	if err != nil {
		return err
	}

	for _, constraint := range ctx.Constraints().Edges {
		if src, found := idChangeResult[constraint.Target.Source]; found {
			constraint.Target.Source = src
		}
		if tgt, found := idChangeResult[constraint.Target.Target]; found {
			constraint.Target.Target = tgt
		}
		errs = errors.Join(errs, applyEdgeConstraint(ctx, constraint))
	}
	if errs != nil {
		return errs
	}

	resources = resources[:0]
	for _, constraint := range ctx.Constraints().Resources {
		res, err := ctx.RawView().Vertex(constraint.Target)
		if err != nil {
			errs = errors.Join(errs, fmt.Errorf("could not find resource %s: %w", constraint.Target, err))
			continue
		}
		resources = append(resources, res)
	}

	_, err = op.MakeResourcesOperational(resources)
	return err
}

// applyApplicationConstraint returns a resource to be made operational, if needed. Otherwise, it returns nil.
func applyApplicationConstraint(ctx solution_context.SolutionContext, constraint constraints.ApplicationConstraint) (*construct.Resource, error) {
	ctx = ctx.With("constraint", constraint)

	switch constraint.Operator {
	case constraints.AddConstraintOperator:
		res := construct.CreateResource(constraint.Node)
		err := ctx.RawView().AddVertex(res)
		return res, err

	case constraints.RemoveConstraintOperator:
		return nil, construct.RemoveResource(ctx.RawView(), constraint.Node)

	case constraints.ReplaceConstraintOperator:
		node, err := ctx.RawView().Vertex(constraint.Node)
		if err != nil {
			return nil, fmt.Errorf("could not find resource for %s: %w", constraint.Node, err)
		}
		if node.ID.QualifiedTypeName() == constraint.ReplacementNode.QualifiedTypeName() {
			node.ID = constraint.ReplacementNode
			return nil, construct.PropagateUpdatedId(ctx.RawView(), constraint.Node)
		} else {
			replacement := construct.CreateResource(constraint.ReplacementNode)
			return replacement, construct.ReplaceResource(ctx.RawView(), constraint.Node, replacement)
		}

	default:
		return nil, fmt.Errorf("unknown operator %s", constraint.Operator)
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

	addPath := func() error {
		var resources []*construct.Resource
		switch _, err := ctx.RawView().Vertex(constraint.Target.Source); {
		case errors.Is(err, graph.ErrVertexNotFound):
			resources = append(resources, construct.CreateResource(constraint.Target.Source))

		case err != nil:
			return fmt.Errorf("could not get source resource %s: %w", constraint.Target.Source, err)
		}

		switch _, err := ctx.RawView().Vertex(constraint.Target.Target); {
		case errors.Is(err, graph.ErrVertexNotFound):
			resources = append(resources, construct.CreateResource(constraint.Target.Target))

		case err != nil:
			return fmt.Errorf("could not get target resource %s: %w", constraint.Target.Target, err)
		}
		idChangeResult, err := ctx.OperationalView().MakeResourcesOperational(resources)
		if err != nil {
			return err
		}
		if src, found := idChangeResult[constraint.Target.Source]; found {
			constraint.Target.Source = src
		}
		if tgt, found := idChangeResult[constraint.Target.Target]; found {
			constraint.Target.Target = tgt
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
				errs = errors.Join(errs, ctx.RawView().RemoveEdge(path[i-1], res))
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

	case constraints.MustContainConstraintOperator, constraints.MustNotContainConstraintOperator:
		// recompute the path with the constraint applied
		if err := removePath(); err != nil {
			return err
		}
		return addPath()
	}
	return nil
}
