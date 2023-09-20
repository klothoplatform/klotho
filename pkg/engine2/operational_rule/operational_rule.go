package operational_rule

import (
	"github.com/klothoplatform/klotho/pkg/construct"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base2"
	"go.uber.org/zap"
)

type (
	Graph interface {
		ListResources() []construct.Resource
		AddResource(resource construct.Resource)
		RemoveResource(resource construct.Resource, explicit bool) error
		AddDependency(from construct.Resource, to construct.Resource) error
		RemoveDependency(from construct.ResourceId, to construct.ResourceId) error
		GetResource(id construct.ResourceId) construct.Resource
		GetFunctionalDownstreamResourcesOfType(resource construct.Resource, qualifiedType construct.ResourceId) []construct.Resource
		GetFunctionalDownstreamResources(resource construct.Resource) []construct.Resource
		GetFunctionalUpstreamResourcesOfType(resource construct.Resource, qualifiedType construct.ResourceId) []construct.Resource
		GetFunctionalUpstreamResources(resource construct.Resource) []construct.Resource
		ReplaceResourceId(oldId construct.ResourceId, resource construct.Resource) error
		ConfigureResource(resource construct.Resource, configuration knowledgebase.Configuration, data knowledgebase.ConfigTemplateData) error
	}

	OperationalRuleContext struct {
		Property             *knowledgebase.Property
		ConfigCtx            knowledgebase.ConfigTemplateContext
		Data                 knowledgebase.ConfigTemplateData
		Graph                Graph
		KB                   *knowledgebase.KnowledgeBase
		CreateResourcefromId func(id construct.ResourceId) construct.Resource
	}
)

func (ctx OperationalRuleContext) HandleOperationalRule(rule knowledgebase.OperationalRule) error {
	if rule.If != "" {
		data := knowledgebase.ConfigTemplateData{}
		result := false
		err := ctx.ConfigCtx.ExecuteDecode(rule.If, data, &result)
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
		ctx.HandleConfigurationRule(operationalConfig)
	}

	return nil
}
