package operational_rule

import (
	"fmt"
	"reflect"

	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base2"
)

func (ctx OperationalRuleContext) HandleConfigurationRule(config knowledgebase.ConfigurationRule) error {
	res, err := ctx.ConfigCtx.ExecuteDecodeAsResourceId(config.Resource, ctx.Data)
	if err != nil {
		return err
	}
	resource, _ := ctx.Graph.GetResource(res)
	if resource == nil {
		return fmt.Errorf("resource %s not found", res)
	}
	val, err := resource.GetProperty(config.Config.Field)
	action := "set"
	if err == nil && val != nil {
		if reflect.ValueOf(val).Kind() == reflect.Slice || reflect.ValueOf(val).Kind() == reflect.Array || reflect.ValueOf(val).Kind() == reflect.Map {
			action = "add"
		}
	}

	err = ctx.Graph.ConfigureResource(resource, config.Config, ctx.Data, action)
	if err != nil {
		return err
	}
	return nil
}
