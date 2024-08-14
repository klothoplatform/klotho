package engine

import (
	"errors"
	"fmt"

	"github.com/dominikbraun/graph"
	construct "github.com/klothoplatform/klotho/pkg/construct"
	"github.com/klothoplatform/klotho/pkg/engine/constraints"
	"github.com/klothoplatform/klotho/pkg/engine/reconciler"
	"github.com/klothoplatform/klotho/pkg/engine/solution"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledgebase"
	"github.com/klothoplatform/klotho/pkg/tui"
)

func ApplyConstraints(sol solution.Solution) error {
	prog := tui.GetProgress(sol.Context())

	cs := sol.Constraints()
	current, total := 0, len(cs.Application)+len(cs.Edges)+len(cs.Resources)

	var errs []error
	for _, constraint := range cs.Application {
		err := applyApplicationConstraint(sol, constraint)
		if err != nil {
			errs = append(errs, fmt.Errorf("failed to apply constraint %#v: %w", constraint, err))
		}
		current++
		prog.Update("Loading constraints", current, total)
	}
	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	for _, constraint := range cs.Edges {
		err := applyEdgeConstraint(sol, constraint)
		if err != nil {
			errs = append(errs, fmt.Errorf("failed to apply constraint %#v: %w", constraint, err))
		}
		current++
		prog.Update("Loading constraints", current, total)
	}
	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	resourceConstraints := cs.Resources
	for i := range resourceConstraints {
		err := applySanitization(sol, &resourceConstraints[i])
		if err != nil {
			errs = append(errs, fmt.Errorf("failed to apply constraint %#v: %w", resourceConstraints[i], err))
		}
		current++
		prog.Update("Loading constraints", current, total)
	}
	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	return nil
}

// applyApplicationConstraint returns a resource to be made operational, if needed. Otherwise, it returns nil.
func applyApplicationConstraint(ctx solution.Solution, constraint constraints.ApplicationConstraint) error {
	res, err := knowledgebase.CreateResource(ctx.KnowledgeBase(), constraint.Node)
	if err != nil {
		return err
	}

	switch constraint.Operator {
	case constraints.AddConstraintOperator:
		return ctx.OperationalView().AddVertex(res)

	case constraints.MustExistConstraintOperator:
		err := ctx.OperationalView().AddVertex(res)
		if errors.Is(err, graph.ErrVertexAlreadyExists) {
			return nil
		}
		return err

	case constraints.ImportConstraintOperator:
		res.Imported = true
		return ctx.OperationalView().AddVertex(res)

	case constraints.RemoveConstraintOperator:
		return reconciler.RemoveResource(ctx, res.ID, true)

	case constraints.MustNotExistConstraintOperator:
		err := reconciler.RemoveResource(ctx, res.ID, false)
		if errors.Is(err, graph.ErrVertexNotFound) {
			return nil
		}
		return err

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
func applyEdgeConstraint(ctx solution.Solution, constraint constraints.EdgeConstraint) error {
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

	if constraint.Target.Source.Name == "" || constraint.Target.Target.Name == "" {
		if constraint.Target.Source.Name == "" && constraint.Target.Target.Name == "" {
			return fmt.Errorf("source and target names are empty")
		}
		// This is considered a global constraint for the type which does not have a name and
		// will be applied anytime a new resource is added to the graph
		return nil
	}

	switch constraint.Operator {
	case constraints.AddConstraintOperator:
		return ctx.OperationalView().AddEdge(constraint.Target.Source, constraint.Target.Target)

	case constraints.MustExistConstraintOperator:
		err := ctx.OperationalView().AddEdge(constraint.Target.Source, constraint.Target.Target)
		if errors.Is(err, graph.ErrEdgeAlreadyExists) {
			return nil
		}
		return err

	case constraints.RemoveConstraintOperator:
		return reconciler.RemovePath(constraint.Target.Source, constraint.Target.Target, ctx)

	case constraints.MustNotExistConstraintOperator:
		err := reconciler.RemovePath(constraint.Target.Source, constraint.Target.Target, ctx)
		if errors.Is(err, graph.ErrTargetNotReachable) {
			return nil
		}
	}
	return nil
}

// applySanitization applies sanitization to the resource name in ResourceConstraints. This is not needed on
// Application or Edge constraints due to them applying within the graph (to make sure that even generated resources
// are sanitized).
func applySanitization(ctx solution.Solution, constraint *constraints.ResourceConstraint) error {
	rt, err := ctx.KnowledgeBase().GetResourceTemplate(constraint.Target)
	if err != nil {
		return err
	}
	constraint.Target.Name, err = rt.SanitizeName(constraint.Target.Name)
	return err
}
