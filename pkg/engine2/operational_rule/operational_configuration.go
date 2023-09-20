package operational_rule

import (
	"fmt"

	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base2"
)

func (ctx OperationalRuleContext) HandleConfigurationRule(config knowledgebase.ConfigurationRule) error {
	res, err := ctx.ConfigCtx.ExecuteDecodeAsResourceId(config.Resource, ctx.Data)
	if err != nil {
		return err
	}
	resource := ctx.Graph.GetResource(res)
	if resource == nil {
		return fmt.Errorf("resource %s not found", res)
	}
	err = ctx.Graph.ConfigureResource(resource, config.Config, ctx.Data)
	if err != nil {
		return err
	}
	return nil
}
