package operational_rule

import (
	"errors"
	"fmt"

	"github.com/dominikbraun/graph"
	construct "github.com/klothoplatform/klotho/pkg/construct2"
	"github.com/klothoplatform/klotho/pkg/engine2/reconciler"
	"github.com/klothoplatform/klotho/pkg/engine2/solution_context"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base2"
	"github.com/klothoplatform/klotho/pkg/set"
	"go.uber.org/zap"
)

//go:generate 	mockgen -source=./operational_rule.go --destination=../operational_eval/operational_rule_mock_test.go --package=operational_eval

type (
	OperationalRuleContext struct {
		Solution solution_context.SolutionContext
		Property knowledgebase.Property
		Data     knowledgebase.DynamicValueData
	}

	OpRuleHandler interface {
		HandleOperationalRule(rule knowledgebase.OperationalRule) error
		HandlePropertyRule(rule knowledgebase.PropertyRule) error
		SetData(data knowledgebase.DynamicValueData)
	}
)

func (ctx *OperationalRuleContext) HandleOperationalRule(rule knowledgebase.OperationalRule) error {
	shouldRun, err := EvaluateIfCondition(rule.If, ctx.Solution, ctx.Data)
	if err != nil {
		return err
	}
	if !shouldRun {
		return nil
	}

	var errs error
	for i, operationalStep := range rule.Steps {
		err := ctx.HandleOperationalStep(operationalStep)
		if err != nil {
			errs = errors.Join(errs, fmt.Errorf("could not apply step %d: %w", i, err))
			continue
		}
	}

	for i, operationalConfig := range rule.ConfigurationRules {
		err := ctx.HandleConfigurationRule(operationalConfig)
		if err != nil {
			errs = errors.Join(errs, fmt.Errorf("could not apply configuration rule %d: %w", i, err))
		}
	}

	return errs
}

func (ctx *OperationalRuleContext) HandlePropertyRule(rule knowledgebase.PropertyRule) error {
	if ctx.Property == nil {
		return fmt.Errorf("property rule has no property")
	}
	if ctx.Data.Resource.IsZero() {
		return fmt.Errorf("property rule has no resource")
	}

	shouldRun, err := EvaluateIfCondition(rule.If, ctx.Solution, ctx.Data)
	if err != nil {
		return err
	}
	if !shouldRun {
		return nil
	}

	if ctx.Property != nil && len(rule.Step.Resources) > 0 {
		err := ctx.CleanProperty(rule.Step)
		if err != nil {
			return err
		}
	}

	var errs error
	if len(rule.Step.Resources) > 0 {
		err = ctx.HandleOperationalStep(rule.Step)
		if err != nil {
			errs = errors.Join(errs, fmt.Errorf("could not apply step: %w", err))
		}
	}

	if rule.Value != nil {
		dynctx := solution_context.DynamicCtx(ctx.Solution)
		val, err := ctx.Property.Parse(rule.Value, dynctx, ctx.Data)
		if err != nil {
			errs = errors.Join(errs, fmt.Errorf("could not parse value %s: %w", rule.Value, err))
		}
		resource, err := ctx.Solution.RawView().Vertex(ctx.Data.Resource)
		if err != nil {
			errs = errors.Join(errs, fmt.Errorf("could not get resource %s: %w", ctx.Data.Resource, err))
		} else {
			err = ctx.Property.SetProperty(resource, val)
			if err != nil {
				errs = errors.Join(errs, fmt.Errorf("could not set property %s: %w", ctx.Property, err))
			}
		}
	}
	return errs
}

func (ctx *OperationalRuleContext) SetData(data knowledgebase.DynamicValueData) {
	ctx.Data = data
}

// CleanProperty clears the property associated with the rule if it no longer matches the rule.
// For array properties, each element must match at least one step selector and non-matching
// elements will be removed.
func (ctx OperationalRuleContext) CleanProperty(step knowledgebase.OperationalStep) error {
	log := zap.L().With(
		zap.String("op", "op_rule"),
		zap.String("property", ctx.Property.Details().Path),
		zap.String("resource", ctx.Data.Resource.String()),
	).Sugar()
	resource, err := ctx.Solution.RawView().Vertex(ctx.Data.Resource)
	if err != nil {
		return err
	}
	path, err := resource.PropertyPath(ctx.Property.Details().Path)
	if err != nil {
		return err
	}
	prop := path.Get()
	if prop == nil {
		return nil
	}

	checkResForMatch := func(res construct.ResourceId) (bool, error) {
		propRes, err := ctx.Solution.RawView().Vertex(res)
		if err != nil {
			return false, err
		}
		for i, sel := range step.Resources {
			match, err := sel.IsMatch(solution_context.DynamicCtx(ctx.Solution), ctx.Data, propRes)
			if err != nil {
				return false, fmt.Errorf("error checking if %s matches selector %d: %w", prop, i, err)
			}
			if match {
				return true, nil
			}
		}

		return false, nil
	}

	switch prop := prop.(type) {
	case construct.ResourceId:
		isMatch, err := checkResForMatch(prop)
		if err != nil {
			return err
		}
		if isMatch {
			return nil
		}
		log.Infof("removing %s, does not match selectors", prop)
		err = path.Remove(nil)
		if err != nil {
			return err
		}
		err = ForceRemoveDependency(ctx.Data.Resource, prop, ctx.Solution)
		if err != nil {
			return err
		}
		return reconciler.RemoveResource(ctx.Solution, prop, false)

	case []construct.ResourceId:
		matching := make([]construct.ResourceId, 0, len(prop))
		toRemove := make(set.Set[construct.ResourceId])
		for _, id := range prop {
			isMatch, err := checkResForMatch(id)
			if err != nil {
				return err
			}
			if !isMatch {
				toRemove.Add(id)
			} else {
				matching = append(matching, id)
			}
		}

		if len(matching) == len(prop) {
			return nil
		}
		err := path.Set(matching)
		if err != nil {
			return err
		}
		var errs error
		for rem := range toRemove {
			log.Infof("removing %s, does not match selectors", prop)
			errs = errors.Join(errs, ForceRemoveDependency(ctx.Data.Resource, rem, ctx.Solution))
			errs = errors.Join(errs, reconciler.RemoveResource(ctx.Solution, rem, false))
		}

	case []any:
		matching := make([]any, 0, len(prop))
		toRemove := make(set.Set[construct.ResourceId])
		for _, propV := range prop {
			id, ok := propV.(construct.ResourceId)
			if !ok {
				propRef, ok := propV.(construct.PropertyRef)
				if !ok {
					matching = append(matching, propV)
					continue
				}
				id = propRef.Resource
			}
			isMatch, err := checkResForMatch(id)
			if err != nil {
				return err
			}
			if !isMatch {
				toRemove.Add(id)
			} else {
				matching = append(matching, id)
			}
		}

		if len(matching) == len(prop) {
			return nil
		}
		err := path.Set(matching)
		if err != nil {
			return err
		}
		var errs error
		for rem := range toRemove {
			log.Infof("removing %s, does not match selectors", prop)
			errs = errors.Join(errs, ForceRemoveDependency(ctx.Data.Resource, rem, ctx.Solution))
			errs = errors.Join(errs, reconciler.RemoveResource(ctx.Solution, rem, false))
		}

	case construct.PropertyRef:
		isMatch, err := checkResForMatch(prop.Resource)
		if err != nil {
			return err
		}
		if isMatch {
			return nil
		}
		log.Infof("removing %s, does not match selectors", prop)
		err = path.Remove(nil)
		if err != nil {
			return err
		}
		err = ForceRemoveDependency(ctx.Data.Resource, prop.Resource, ctx.Solution)
		if err != nil {
			return err
		}
		return reconciler.RemoveResource(ctx.Solution, prop.Resource, false)
	}

	return nil
}

func EvaluateIfCondition(
	tmplString string,
	sol solution_context.SolutionContext,
	data knowledgebase.DynamicValueData,
) (bool, error) {
	if tmplString == "" {
		return true, nil
	}
	result := false
	dyn := solution_context.DynamicCtx(sol)
	err := dyn.ExecuteDecode(tmplString, data, &result)
	if err != nil {
		return false, err
	}
	return result, nil
}

func ForceRemoveDependency(
	res1, res2 construct.ResourceId,
	sol solution_context.SolutionContext,
) error {

	err := sol.RawView().RemoveEdge(res1, res2)
	if err != nil && !errors.Is(err, graph.ErrEdgeNotFound) {
		return err
	}
	err = sol.RawView().RemoveEdge(res2, res1)
	if err != nil && !errors.Is(err, graph.ErrEdgeNotFound) {
		return err
	}
	return nil
}
