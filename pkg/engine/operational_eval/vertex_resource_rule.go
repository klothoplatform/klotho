package operational_eval

import (
	"errors"
	"fmt"

	construct "github.com/klothoplatform/klotho/pkg/construct2"
	"github.com/klothoplatform/klotho/pkg/engine/operational_rule"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base2"
)

type (
	resourceRuleVertex struct {
		Resource construct.ResourceId
		Rule     knowledgebase.AdditionalRule
		hash     string
	}
)

func (v resourceRuleVertex) Key() Key {
	return Key{Ref: construct.PropertyRef{Resource: v.Resource}, RuleHash: v.hash}
}

func (v *resourceRuleVertex) Dependencies(eval *Evaluator, propCtx dependencyCapturer) error {
	resData := knowledgebase.DynamicValueData{Resource: v.Resource}
	var errs error
	errs = errors.Join(errs, propCtx.ExecuteOpRule(resData, knowledgebase.OperationalRule{
		If:    v.Rule.If,
		Steps: v.Rule.Steps,
	}))
	if errs != nil {
		return fmt.Errorf("could not execute %s: %w", v.Key(), errs)
	}
	return nil
}

func (v *resourceRuleVertex) UpdateFrom(other Vertex) {
	if v == other {
		return
	}
	otherRule, ok := other.(*resourceRuleVertex)
	if !ok {
		panic(fmt.Sprintf("cannot merge edge with non-edge vertex: %T", other))
	}
	if v.Resource != otherRule.Resource {
		panic(fmt.Sprintf("cannot merge resource rule with different refs: %s != %s", v.Resource, otherRule.Resource))
	}
	v.Rule = otherRule.Rule
}

func (v *resourceRuleVertex) Evaluate(eval *Evaluator) error {
	sol := eval.Solution.With("resource", v.Resource)
	opCtx := operational_rule.OperationalRuleContext{
		Solution: sol,
		Data:     knowledgebase.DynamicValueData{Resource: v.Resource},
	}
	if err := v.evaluateResourceRule(&opCtx, eval); err != nil {
		return err
	}

	res, err := sol.RawView().Vertex(v.Resource)
	if err != nil {
		return fmt.Errorf("could not get resource to evaluate %s: %w", v.Resource, err)
	}
	if err := eval.UpdateId(v.Resource, res.ID); err != nil {
		return err
	}
	return nil
}

func (v *resourceRuleVertex) evaluateResourceRule(
	opCtx operational_rule.OpRuleHandler,
	eval *Evaluator,
) error {
	err := opCtx.HandleOperationalRule(knowledgebase.OperationalRule{
		If:    v.Rule.If,
		Steps: v.Rule.Steps,
	})
	if err != nil {
		return fmt.Errorf(
			"could not apply resource %s operational rule: %w",
			v.Resource, err,
		)
	}

	return nil
}
