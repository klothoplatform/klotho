package klotho

import (
	"fmt"
	"reflect"

	"github.com/klothoplatform/klotho/pkg/compiler/types"
	"github.com/klothoplatform/klotho/pkg/construct"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base"
)

type KlothoProvider struct {
	constructsByType map[string]construct.BaseConstruct
}

func (KlothoProvider) Name() string { return construct.AbstractConstructProvider }

func (KlothoProvider) ListResources() []construct.Resource {
	return nil
}

func (KlothoProvider) GetOperationalTemplates() map[construct.ResourceId]*construct.ResourceTemplate {
	return nil
}

func (KlothoProvider) GetEdgeTemplates() map[string]*knowledgebase.EdgeTemplate {
	return nil
}

func (p *KlothoProvider) CreateConstructFromId(id construct.ResourceId, dag *construct.ConstructGraph) (construct.BaseConstruct, error) {
	if p.constructsByType == nil {
		p.constructsByType = make(map[string]construct.BaseConstruct)
		for _, c := range types.ListAllConstructs() {
			p.constructsByType[c.Id().Type] = c
		}
	}
	c, ok := p.constructsByType[id.Type]
	if !ok {
		return nil, fmt.Errorf("no construct matching type '%s'", id.Type)
	}

	newConstruct := reflect.New(reflect.TypeOf(c).Elem()).Interface()
	c, ok = newConstruct.(construct.Construct)
	if !ok {
		return nil, fmt.Errorf("item %s of type %T is not of type construct.Construct", id, newConstruct)
	}
	reflect.ValueOf(c).Elem().FieldByName("Name").SetString(id.Name)

	return c, nil
}
