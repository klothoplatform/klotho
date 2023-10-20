package operational_rule

import (
	"fmt"
	"strings"

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
	val, err := resource.GetProperty(config.Config.Field)
	action := "set"
	if err == nil && val != nil {
		resTempalte, err := ctx.Solution.KnowledgeBase().GetResourceTemplate(resource.ID)
		if err != nil {
			return err
		}
		prop := resTempalte.GetProperty(config.Config.Field)
		if prop != nil && (strings.Contains(prop.Type, "list") || strings.Contains(prop.Type, "set") || strings.Contains(prop.Type, "map")) {
			action = "add"
		}
	}

	resolvedField := config.Config.Field
	err = dyn.ExecuteDecode(config.Config.Field, ctx.Data, &resolvedField)
	config.Config.Field = resolvedField

	err = solution_context.ConfigureResource(ctx.Solution, resource, config.Config, ctx.Data, action)
	if err != nil {
		return err
	}
	return nil
}
