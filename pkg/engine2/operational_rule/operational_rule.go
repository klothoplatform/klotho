package operational_rule

import (
	"errors"
	"fmt"

	"github.com/klothoplatform/klotho/pkg/engine2/solution_context"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base2"
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

	if rule.If != "" {
		result := false
		dyn := solution_context.DynamicCtx(ctx.Solution)
		err := dyn.ExecuteDecode(rule.If, ctx.Data, &result)
		if err != nil {
			return err
		}
		if !result {
			zap.S().Debugf("rule did not match if condition, skipping")
			return nil
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
