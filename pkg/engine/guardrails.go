package engine

import (
	"fmt"
	"reflect"

	"github.com/klothoplatform/klotho/pkg/collectionutil"
	"github.com/klothoplatform/klotho/pkg/core"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base"
	"gopkg.in/yaml.v3"
)

type (
	Guardrails struct {
		AllowedResources    []core.ResourceId `yaml:"allowed_resources"`
		DisallowedResources []core.ResourceId `yaml:"disallowed_resources"`
	}
)

func (e *Engine) LoadGuardrails(bytes []byte) error {
	inputGuardrails := &Guardrails{}
	guardrails := &Guardrails{}
	err := yaml.Unmarshal(bytes, inputGuardrails)
	if err != nil {
		return err
	}
	if inputGuardrails.AllowedResources != nil && inputGuardrails.DisallowedResources != nil {
		return fmt.Errorf("both allowed and disallowed resources specified")
	}
	for _, provider := range e.Providers {
		for _, res := range provider.ListResources() {
			if inputGuardrails.AllowedResources == nil && inputGuardrails.DisallowedResources == nil {
				guardrails.AllowedResources = append(guardrails.AllowedResources, res.Id())
			} else if inputGuardrails.AllowedResources != nil && !collectionutil.Contains(inputGuardrails.AllowedResources, res.Id()) {
				guardrails.DisallowedResources = append(guardrails.DisallowedResources, res.Id())
			} else if inputGuardrails.DisallowedResources != nil && !collectionutil.Contains(inputGuardrails.DisallowedResources, res.Id()) {
				guardrails.AllowedResources = append(guardrails.AllowedResources, res.Id())
			} else if collectionutil.Contains(inputGuardrails.DisallowedResources, res.Id()) {
				guardrails.DisallowedResources = append(guardrails.DisallowedResources, res.Id())
			} else {
				guardrails.AllowedResources = append(guardrails.AllowedResources, res.Id())
			}
		}
	}
	e.Guardrails = guardrails
	return e.ApplyGuardrails()
}

func (e *Engine) ApplyGuardrails() error {
	for res := range e.ClassificationDocument.Classifications {
		id := &core.ResourceId{}
		err := id.UnmarshalText([]byte(res))
		if err != nil {
			return err
		}
		if collectionutil.Contains(e.Guardrails.DisallowedResources, *id) {
			delete(e.ClassificationDocument.Classifications, res)
		}
	}
	for edge := range e.KnowledgeBase.EdgeMap {
		src := reflect.New(edge.Source.Elem()).Interface().(core.Resource)
		dst := reflect.New(edge.Destination.Elem()).Interface().(core.Resource)
		if collectionutil.Contains(e.Guardrails.DisallowedResources, src.Id()) || collectionutil.Contains(e.Guardrails.DisallowedResources, dst.Id()) {
			delete(e.KnowledgeBase.EdgeMap, edge)
			srcByType := e.KnowledgeBase.EdgesByType[edge.Source]
			if srcByType != nil {
				newOutgoing := []knowledgebase.Edge{}
				for _, item := range srcByType.Outgoing {
					if item != edge {
						newOutgoing = append(newOutgoing, item)
					}
				}
				srcByType.Outgoing = newOutgoing
			}
			dstByType := e.KnowledgeBase.EdgesByType[edge.Destination]
			if dstByType != nil {
				newIncoming := []knowledgebase.Edge{}
				for _, item := range dstByType.Incoming {
					if item != edge {
						newIncoming = append(newIncoming, item)
					}
				}
				dstByType.Incoming = newIncoming
			}
		}
	}

	for res := range e.ResourceTemplates {
		if collectionutil.Contains(e.Guardrails.DisallowedResources, res) {
			delete(e.ResourceTemplates, res)
		}
	}
	return nil
}

func (g *Guardrails) IsResourceAllowed(res core.ResourceId) bool {
	if g.AllowedResources == nil {
		return true
	}
	baseId := core.ResourceId{Provider: res.Provider, Type: res.Type}
	return collectionutil.Contains(g.AllowedResources, baseId)
}
