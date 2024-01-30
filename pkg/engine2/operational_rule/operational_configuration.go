package operational_rule

import (
	"fmt"

	"github.com/klothoplatform/klotho/pkg/engine2/solution_context"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base2"
)

func (ctx OperationalRuleContext) HandleConfigurationRule(config knowledgebase.ConfigurationRule) error {
	dyn := solution_context.DynamicCtx(ctx.Solution)
	res, err := knowledgebase.ExecuteDecodeAsResourceId(dyn, config.Resource, ctx.Data)
	if err != nil {
		return err
	}
	resource, err := ctx.Solution.DataflowGraph().Vertex(res)
	if err != nil {
		return fmt.Errorf("resource %s not found: %w", res, err)
	}

	resolvedField := config.Config.Field
	err = dyn.ExecuteDecode(config.Config.Field, ctx.Data, &resolvedField)
	if err != nil {
		return err
	}
	config.Config.Field = resolvedField
	configurer := &solution_context.Configurer{Ctx: ctx.Solution}

	err = configurer.ConfigureResource(resource, config.Config, ctx.Data, "add", false)
	if err != nil {
		return err
	}
	return nil
}
