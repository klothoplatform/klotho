package operational_rule

import (
	"errors"
	"fmt"

	construct "github.com/klothoplatform/klotho/pkg/construct2"
	"github.com/klothoplatform/klotho/pkg/engine2/reconciler"
	"github.com/klothoplatform/klotho/pkg/engine2/solution_context"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base2"
	"github.com/klothoplatform/klotho/pkg/set"
	"go.uber.org/zap"
)

type (
	OperationalRuleContext struct {
		Solution solution_context.SolutionContext
		Property *knowledgebase.Property
		Data     knowledgebase.DynamicValueData
	}
)

func (ctx OperationalRuleContext) HandleOperationalRule(rule knowledgebase.OperationalRule) error {

	shouldRun, err := ctx.EvaluateIfCondition(rule)
	if err != nil {
		return err
	}
	if !shouldRun {
		return nil
	}

	if ctx.Property != nil && len(rule.Steps) > 0 {
		err := ctx.CleanProperty(rule)
		if err != nil {
			return err
		}
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

// CleanProperty clears the property associated with the rule if it no longer matches the rule.
// For array properties, each element must match at least one step selector and non-matching
// elements will be removed.
func (ctx OperationalRuleContext) CleanProperty(rule knowledgebase.OperationalRule) error {
	log := zap.L().With(
		zap.String("op", "op_rule"),
		zap.String("property", ctx.Property.Path),
		zap.String("resource", ctx.Data.Resource.String()),
	).Sugar()
	resource, err := ctx.Solution.RawView().Vertex(ctx.Data.Resource)
	if err != nil {
		return err
	}
	path, err := resource.PropertyPath(ctx.Property.Path)
	if err != nil {
		return err
	}
	prop := path.Get()
	if prop == nil {
		return nil
	}

	switch prop := prop.(type) {
	case construct.ResourceId:
		propRes, err := ctx.Solution.RawView().Vertex(prop)
		if err != nil {
			return err
		}
		for _, step := range rule.Steps {
			for i, sel := range step.Resources {
				match, err := sel.IsMatch(solution_context.DynamicCtx(ctx.Solution), ctx.Data, propRes)
				if err != nil {
					return fmt.Errorf("error checking if %s matches selector %d: %w", prop, i, err)
				}
				if match {
					return nil
				}
			}
		}
		log.Infof("removing %s, does not match selectors", prop)
		err = path.Remove(nil)
		if err != nil {
			return err
		}
		return reconciler.RemoveResource(ctx.Solution, prop, false)

	case []construct.ResourceId:
		matching := make([]construct.ResourceId, 0, len(prop))
		toRemove := make(set.Set[construct.ResourceId])
	ridElemLoop:
		for _, id := range prop {
			propRes, err := ctx.Solution.RawView().Vertex(id)
			if err != nil {
				return err
			}
			for _, step := range rule.Steps {
				for i, sel := range step.Resources {
					match, err := sel.IsMatch(solution_context.DynamicCtx(ctx.Solution), ctx.Data, propRes)
					if err != nil {
						return fmt.Errorf("error checking if %s matches selector %d: %w", id, i, err)
					}
					if match {
						matching = append(matching, id)
						continue ridElemLoop
					}
				}
			}
			toRemove.Add(id)
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
			errs = errors.Join(errs, reconciler.RemoveResource(ctx.Solution, rem, false))
		}

	case []any:
		matching := make([]any, 0, len(prop))
		toRemove := make(set.Set[construct.ResourceId])
	anyElemLoop:
		for _, propV := range prop {
			id, ok := propV.(construct.ResourceId)
			if !ok {
				matching = append(matching, propV)
				continue
			}
			propRes, err := ctx.Solution.RawView().Vertex(id)
			if err != nil {
				return err
			}
			for _, step := range rule.Steps {
				for i, sel := range step.Resources {
					match, err := sel.IsMatch(solution_context.DynamicCtx(ctx.Solution), ctx.Data, propRes)
					if err != nil {
						return fmt.Errorf("error checking if %s matches selector %d: %w", id, i, err)
					}
					if match {
						matching = append(matching, id)
						continue anyElemLoop
					}
				}
			}
			toRemove.Add(id)
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
			errs = errors.Join(errs, reconciler.RemoveResource(ctx.Solution, rem, false))
		}
	}

	return nil
}

func (ctx OperationalRuleContext) EvaluateIfCondition(rule knowledgebase.OperationalRule) (bool, error) {
	if rule.If == "" {
		return true, nil
	}
	result := false
	dyn := solution_context.DynamicCtx(ctx.Solution)
	err := dyn.ExecuteDecode(rule.If, ctx.Data, &result)
	if err != nil {
		return false, err
	}
	return result, nil
}
