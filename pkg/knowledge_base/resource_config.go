package knowledgebase

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"text/template"

	"github.com/klothoplatform/klotho/pkg/construct"
)

type (
	// Configuration defines how to act on any intrinsic values of a resource to make it operational
	Configuration struct {
		// Fields defines a field that should be set on the resource
		Field string `json:"field" yaml:"field"`
		// Value defines the value that should be set on the resource
		DefaultValue any `json:"default_value" yaml:"default_value"`

		Steps []ConfigurationStep `json:"steps" yaml:"steps"`
	}

	ConfigurationStep struct {
		Object   string `json:"object" yaml:"object"`
		Property string `json:"property" yaml:"property"`
		Value    string `json:"value" yaml:"value"`
	}

	ConfigurationContext struct {
		dag      *construct.ResourceGraph
		resource construct.Resource
		field    string
		Value    any
	}
)

func (c *Configuration) Apply(dag *construct.ResourceGraph, resource construct.Resource, value any) error {
	ctx := &ConfigurationContext{
		dag:      dag,
		resource: resource,
		field:    c.Field,
		Value:    c.DefaultValue,
	}
	if value != nil {
		ctx.Value = value
	}

	for i, step := range c.Steps {
		if err := step.Apply(ctx); err != nil {
			return fmt.Errorf("error applying configuration %s step %d: %w", c.Field, i, err)
		}
	}
	return nil
}

func (c *ConfigurationStep) Apply(ctx *ConfigurationContext) (err error) {
	obj := ctx.resource
	if c.Object != "" {
		objTmpl, err := template.New("value").Funcs(ctx.Funcs()).Parse(c.Object)
		if err != nil {
			return fmt.Errorf("unable to parse object template: %w", err)
		}
		objBuf := new(bytes.Buffer)
		if err = objTmpl.Execute(objBuf, ctx); err != nil {
			return fmt.Errorf("unable to execute object template: %w", err)
		}
		var objId construct.ResourceId
		if err := objId.UnmarshalText(objBuf.Bytes()); err != nil {
			return err
		}
		obj = ctx.dag.GetResource(objId)
	}

	objValue := reflect.ValueOf(obj).Elem()

	var field reflect.Value
	if c.Property != "" {
		propPath := strings.Split(c.Property, ".")
		// TODO handle map keys like `Resource.Field[key]`
		field = objValue
		for i, prop := range propPath {
			field = field.FieldByName(prop)
			if !field.IsValid() {
				return fmt.Errorf("property '%s' not found on object '%s' (type %T)", strings.Join(propPath[:i+1], "."), obj.Id(), obj)
			}
		}
	} else {
		if obj != ctx.resource {
			return fmt.Errorf("property required when object is not the resource being configured")
		}
		field = objValue.FieldByName(ctx.field)
	}

	value := reflect.ValueOf(ctx.Value)
	if c.Value != "" {
		valueTmpl, err := template.New("value").Funcs(ctx.Funcs()).Parse(c.Value)
		if err != nil {
			return fmt.Errorf("unable to parse value template: %w", err)
		}
		valueBuf := new(bytes.Buffer)
		if err = valueTmpl.Execute(valueBuf, ctx); err != nil {
			return fmt.Errorf("unable to execute value template: %w", err)
		}

		value = reflect.New(field.Type())
		err = json.Unmarshal(valueBuf.Bytes(), value.Interface())
		if err != nil {
			if field.Kind() == reflect.String {
				// guess that the value is to be taken as a literal string
				value = reflect.ValueOf(valueBuf.String())
			} else {
				return fmt.Errorf("unable to unmarshal value from '%s': %w", valueBuf, err)
			}
		} else {
			value = value.Elem()
		}
	} else if ctx.Value == nil {
		value = reflect.New(field.Type()).Elem()
	}
	if !value.Type().AssignableTo(field.Type()) {
		return fmt.Errorf("value type %s is not assignable to field type %s", value.Type(), field.Type())
	}

	// TODO handle map setting or array appending
	field.Set(value)
	return nil
}
