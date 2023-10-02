package operational_rule

import (
	construct "github.com/klothoplatform/klotho/pkg/construct2"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base2"
	"go.uber.org/zap"
)

type (
	Graph interface {
		ListResources() ([]*construct.Resource, error)
		RemoveResource(resource *construct.Resource, explicit bool) error
		AddDependency(from, to *construct.Resource) error
		RemoveDependency(from, to construct.ResourceId) error
		GetResource(id construct.ResourceId) (*construct.Resource, error)
		DownstreamOfType(resource *construct.Resource, layer int, qualifiedType string) ([]*construct.Resource, error)
		Downstream(resource *construct.Resource, layer int) ([]*construct.Resource, error)
		Upstream(resource *construct.Resource, layer int) ([]*construct.Resource, error)
		ReplaceResourceId(oldId construct.ResourceId, resource construct.ResourceId) error
		ConfigureResource(resource *construct.Resource, configuration knowledgebase.Configuration, data knowledgebase.ConfigTemplateData, action string) error
	}

	OperationalRuleContext struct {
		Property  *knowledgebase.Property
		ConfigCtx knowledgebase.ConfigTemplateContext
		Data      knowledgebase.ConfigTemplateData
		Graph     Graph
		KB        knowledgebase.TemplateKB
	}
)

func (ctx OperationalRuleContext) HandleOperationalRule(rule knowledgebase.OperationalRule) error {
	if rule.If != "" {
		result := false
		err := ctx.ConfigCtx.ExecuteDecode(rule.If, ctx.Data, &result)
		if err != nil {
			return err
		}
		if !result {
			zap.S().Debugf("rule did not match if condition, skipping")
			return nil
		}
	}

	for _, operationalStep := range rule.Steps {
		err := ctx.HandleOperationalStep(operationalStep)
		if err != nil {
			return err
		}
	}

	for _, operationalConfig := range rule.ConfigurationRules {
		err := ctx.HandleConfigurationRule(operationalConfig)
		if err != nil {
			return err
		}
	}

	return nil
}
