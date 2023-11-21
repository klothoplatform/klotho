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

	if ctx.Data.Edge != nil && ctx.Data.Edge.Source.String() == "aws:api_deployment:rest_api_0:api_deployment-0" {
		fmt.Println("here")
	}
	shouldRun, err := EvaluateIfCondition(rule, ctx.Solution, ctx.Data)
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

	checkResForMatch := func(res construct.ResourceId) (bool, error) {
		propRes, err := ctx.Solution.RawView().Vertex(res)
		if err != nil {
			return false, err
		}
		for _, step := range rule.Steps {
			for i, sel := range step.Resources {
				match, err := sel.IsMatch(solution_context.DynamicCtx(ctx.Solution), ctx.Data, propRes)
				if err != nil {
					return false, fmt.Errorf("error checking if %s matches selector %d: %w", prop, i, err)
				}
				if match {
					return true, nil
				}
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
		return reconciler.RemoveResource(ctx.Solution, prop.Resource, false)
	}

	return nil
}

func EvaluateIfCondition(
	rule knowledgebase.OperationalRule,
	sol solution_context.SolutionContext,
	data knowledgebase.DynamicValueData,
) (bool, error) {
	if rule.If == "" {
		return true, nil
	}
	result := false
	dyn := solution_context.DynamicCtx(sol)
	err := dyn.ExecuteDecode(rule.If, data, &result)
	if err != nil {
		return false, err
	}
	return result, nil
}
