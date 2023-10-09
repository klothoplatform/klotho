package operational_rule

import (
	"fmt"
	"reflect"

	"github.com/klothoplatform/klotho/pkg/engine2/solution_context"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base2"
)

func (ctx OperationalRuleContext) HandleConfigurationRule(config knowledgebase.ConfigurationRule) error {
	res, err := ctx.ConfigCtx.ExecuteDecodeAsResourceId(config.Resource, ctx.Data)
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
		if reflect.ValueOf(val).Kind() == reflect.Slice || reflect.ValueOf(val).Kind() == reflect.Array || reflect.ValueOf(val).Kind() == reflect.Map {
			action = "add"
		}
	}

	err = solution_context.ConfigureResource(ctx.Solution, resource, config.Config, ctx.Data, action)
	if err != nil {
		return err
	}
	return nil
}
