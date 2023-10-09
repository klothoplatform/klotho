package operational_rule

import (
	"errors"
	"fmt"

	construct "github.com/klothoplatform/klotho/pkg/construct2"
	"github.com/klothoplatform/klotho/pkg/engine2/solution_context"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base2"
	"go.uber.org/zap"
)

type (
	OperationalRuleContext struct {
		Solution  solution_context.SolutionContext
		Property  *knowledgebase.Property
		ConfigCtx knowledgebase.DynamicValueContext
		Data      knowledgebase.DynamicValueData
	}
)

func (ctx OperationalRuleContext) HandleOperationalRule(rule knowledgebase.OperationalRule) ([]*construct.Resource, error) {
	if rule.If != "" {
		result := false
		err := ctx.ConfigCtx.ExecuteDecode(rule.If, ctx.Data, &result)
		if err != nil {
			return nil, err
		}
		if !result {
			zap.S().Debugf("rule did not match if condition, skipping")
			return nil, nil
		}
	}

	var createdResources []*construct.Resource
	var errs error
	for i, operationalStep := range rule.Steps {
		resources, err := ctx.HandleOperationalStep(operationalStep)
		if err != nil {
			errs = errors.Join(errs, fmt.Errorf("could not apply step %d: %w", i, err))
			continue
		}
		createdResources = append(createdResources, resources...)
	}

	for i, operationalConfig := range rule.ConfigurationRules {
		err := ctx.HandleConfigurationRule(operationalConfig)
		if err != nil {
			errs = errors.Join(errs, fmt.Errorf("could not apply configuration rule %d: %w", i, err))
		}
	}

	return createdResources, errs
}
