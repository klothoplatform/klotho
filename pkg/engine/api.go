package engine

import (
	"fmt"
	"reflect"

	"github.com/klothoplatform/klotho/pkg/core"
)

func (e *Engine) ListResources() []core.Resource {
	var resources []core.Resource
	for _, provider := range e.Providers {
		resources = append(resources, provider.ListResources()...)
	}
	return resources
}

func (e *Engine) ListResourcesByType() []string {
	var resources []string
	for _, construct := range e.Constructs {
		id := construct.Id()
		resources = append(resources, fmt.Sprintf("%s:%s", id.Provider, id.Type))
	}
	for _, res := range e.ListResources() {
		id := res.Id()
		resources = append(resources, fmt.Sprintf("%s:%s", id.Provider, id.Type))
	}
	return resources
}

func (e *Engine) ListProviders() []string {
	providers := []string{core.AbstractConstructProvider}
	for _, provider := range e.Providers {
		providers = append(providers, provider.Name())
	}
	return providers
}

func (e *Engine) ListAttributes() []string {
	attributesMap := map[string]bool{}
	for _, classification := range e.ClassificationDocument.Classifications {
		for _, gives := range classification.Gives {
			attributesMap[gives.Attribute] = true
		}
		for _, is := range classification.Is {
			attributesMap[is] = true
		}
	}
	var attributes []string
	for attribute := range attributesMap {
		attributes = append(attributes, attribute)
	}
	return attributes
}

func (e *Engine) ListResourceFields(provider string, resourceType string) map[string]string {
	if provider == core.AbstractConstructProvider {
		for _, construct := range e.Constructs {
			if construct.Id().Type == resourceType {
				fields := map[string]string{}
				for i := 0; i < reflect.ValueOf(construct).Elem().NumField(); i++ {
					if isFieldConfigurable(construct, i) {
						fields[reflect.ValueOf(construct).Elem().Type().Field(i).Name] = reflect.ValueOf(construct).Elem().Type().Field(i).Type.String()
					}
				}
				return fields
			}
		}
	} else if e.Providers[provider] == nil {
		return map[string]string{}
	}
	for _, res := range e.Providers[provider].ListResources() {
		if res.Id().Type == resourceType {
			fields := map[string]string{}
			for i := 0; i < reflect.ValueOf(res).Elem().NumField(); i++ {
				if isFieldConfigurable(res, i) {
					fields[reflect.ValueOf(res).Elem().Type().Field(i).Name] = reflect.ValueOf(res).Elem().Type().Field(i).Type.String()
				}
			}
			return fields
		}
	}
	return map[string]string{}
}

func isFieldConfigurable(construct core.BaseConstruct, i int) bool {
	field := reflect.ValueOf(construct).Elem().Type().Field(i)
	if field.Type.Implements(reflect.TypeOf((*core.BaseConstruct)(nil)).Elem()) {
		return false
	} else if field.Type.Implements(reflect.TypeOf((*core.Resource)(nil)).Elem()) {
		return false
	} else if field.Type == reflect.TypeOf(core.BaseConstructSet{}) {
		return false
	}
	if field.Type.Kind() == reflect.Array || field.Type.Kind() == reflect.Slice {
		if field.Type.Elem().Implements(reflect.TypeOf((*core.BaseConstruct)(nil)).Elem()) {
			return false
		}
	}
	return true
}
