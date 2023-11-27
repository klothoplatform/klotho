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
		if prop != nil && (strings.HasPrefix(prop.Type, "list") || strings.HasPrefix(prop.Type, "set") || strings.HasPrefix(prop.Type, "map")) {
			action = "add"
		}
	}

	resolvedField := config.Config.Field
	err = dyn.ExecuteDecode(config.Config.Field, ctx.Data, &resolvedField)
	if err != nil {
		return err
	}
	config.Config.Field = resolvedField

	//inspect the value and add dependencies if its in there
	// rval := reflect.ValueOf(config.Config.Value)

	err = solution_context.ConfigureResource(ctx.Solution, resource, config.Config, ctx.Data, action)
	if err != nil {
		return err
	}
	return nil
}

// func (ctx OperationalRuleContext) addDeploymentDependencyFromConfiguration(rval reflect.Value) {
// 	switch rval.Kind() {
// 	case reflect.Slice, reflect.Array:
// 		for i := 0; i < rval.Len(); i++ {
// 		}

// 	case reflect.Map:
// 	case reflect.Struct, reflect.Pointer:
// 	}
// }
