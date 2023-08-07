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

func (e *Engine) ListResourceFields(provider string, resourceType string) map[string]any {
	if provider == core.AbstractConstructProvider {
		for _, construct := range e.Constructs {
			if construct.Id().Type == resourceType {
				fields := map[string]any{}
				for i := 0; i < reflect.ValueOf(construct).Elem().NumField(); i++ {
					field := reflect.ValueOf(construct).Elem().Type().Field(i)
					if isFieldConfigurable(field.Type) {
						fields[field.Name] = getStructFieldFields(field.Type)
					}
				}
				return fields
			}
		}
	} else if e.Providers[provider] == nil {
		return map[string]any{}
	}
	for _, res := range e.Providers[provider].ListResources() {
		if res.Id().Type == resourceType {
			fields := map[string]any{}
			for i := 0; i < reflect.ValueOf(res).Elem().NumField(); i++ {
				field := reflect.ValueOf(res).Elem().Type().Field(i)
				if isFieldConfigurable(field.Type) {
					fields[field.Name] = getStructFieldFields(field.Type)
				}
			}
			return fields
		}
	}
	return map[string]any{}
}

func isFieldConfigurable(field reflect.Type) bool {
	if field.Implements(reflect.TypeOf((*core.BaseConstruct)(nil)).Elem()) {
		return false
	} else if field.Implements(reflect.TypeOf((*core.Resource)(nil)).Elem()) {
		return false
	} else if field == reflect.TypeOf(core.BaseConstructSet{}) {
		return false
	}
	if field.Kind() == reflect.Array || field.Kind() == reflect.Slice {
		if field.Elem().Implements(reflect.TypeOf((*core.BaseConstruct)(nil)).Elem()) {
			return false
		}
	}
	return true
}

func getStructFieldFields(field reflect.Type) any {
	fields := map[string]any{}
	if field.Kind() == reflect.Struct {
		element := reflect.New(field).Interface()
		for i := 0; i < reflect.ValueOf(element).Elem().NumField(); i++ {
			subField := reflect.ValueOf(element).Elem().Type().Field(i)
			if isFieldConfigurable(subField.Type) {
				fields[subField.Name] = getStructFieldFields(subField.Type)
			}
		}
	} else if field.Kind() == reflect.Ptr {
		element := reflect.New(field.Elem()).Interface()
		for i := 0; i < reflect.ValueOf(element).Elem().NumField(); i++ {
			subField := reflect.ValueOf(element).Elem().Type().Field(i)
			if isFieldConfigurable(subField.Type) {
				fields[subField.Name] = getStructFieldFields(subField.Type)
			}
		}
	} else if field.Kind() == reflect.Array || field.Kind() == reflect.Slice {
		arrFields := []any{}
		arrFields = append(arrFields, getStructFieldFields(field.Elem()))
		return arrFields
	} else if field.Kind() == reflect.Map {
		fields["key"] = field.Key().String()
		fields["value"] = getStructFieldFields(field.Elem())

		return fields
	} else {
		return field.String()
	}
	return fields
}
