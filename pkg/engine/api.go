package engine

import (
	"fmt"
	"reflect"

	"github.com/klothoplatform/klotho/pkg/construct"
)

func (e *Engine) ListResources() []construct.Resource {
	var resources []construct.Resource
	for _, res := range e.Guardrails.AllowedResources {
		resource, _ := e.getConstructFromId(res)
		resources = append(resources, resource.(construct.Resource))
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
	providers := []string{construct.AbstractConstructProvider}
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
	if provider == construct.AbstractConstructProvider {
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
	if field.Implements(reflect.TypeOf((*construct.BaseConstruct)(nil)).Elem()) {
		return false
	} else if field.Implements(reflect.TypeOf((*construct.Resource)(nil)).Elem()) {
		return false
	} else if field == reflect.TypeOf(construct.BaseConstructSet{}) {
		return false
	}
	if field.Kind() == reflect.Array || field.Kind() == reflect.Slice {
		if field.Elem().Implements(reflect.TypeOf((*construct.BaseConstruct)(nil)).Elem()) {
			return false
		}
	}
	return true
}

func getStructFieldFields(field reflect.Type) any {
	fields := map[string]any{}
	if field.Kind() == reflect.Ptr {
		field = field.Elem()
	}
	if field.Kind() == reflect.Struct {
		element := reflect.New(field).Interface()
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
		fields["key"] = getStructFieldFields(field.Key())
		fields["value"] = getStructFieldFields(field.Elem())

		return fields
	} else {
		return field.String()
	}
	return fields
}
