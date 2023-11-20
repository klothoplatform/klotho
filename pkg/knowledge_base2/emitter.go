package knowledgebase2

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"text/template"

	construct "github.com/klothoplatform/klotho/pkg/construct2"
)

type (
	Consumption struct {
		Emitted  []ConsumptionObject `json:"emitted" yaml:"emitted"`
		Consumed []ConsumptionObject `json:"consumed" yaml:"consumed"`
	}

	ConsumptionObject struct {
		Model        string `json:"model" yaml:"model"`
		Value        any    `json:"value" yaml:"value"`
		Resource     string `json:"resource" yaml:"resource"`
		PropertyPath string `json:"property_path" yaml:"property_path"`
		Converter    string `json:"converter" yaml:"converter"`
	}

	DelayedConsumption struct {
		Value        any
		Resource     construct.ResourceId
		PropertyPath string
	}
)

func ConsumeFromResource(consumer, emitter *construct.Resource, ctx DynamicValueContext) ([]DelayedConsumption, error) {
	consumerTemplate, err := ctx.KnowledgeBase.GetResourceTemplate(consumer.ID)
	if err != nil {
		return nil, err
	}
	emitterTemplate, err := ctx.KnowledgeBase.GetResourceTemplate(emitter.ID)
	if err != nil {
		return nil, err
	}
	var errs error
	delays := []DelayedConsumption{}
	for _, consume := range consumerTemplate.Consumption.Consumed {
		for _, emit := range emitterTemplate.Consumption.Emitted {
			if consume.Model == emit.Model {
				val, err := emit.Emit(ctx, emitter.ID)
				if err != nil {
					errs = errors.Join(errs, err)
					continue
				}

				pval, _ := consumer.GetProperty(consume.PropertyPath)
				if pval == nil {
					id := consumer.ID
					if consume.Resource != "" {
						data := DynamicValueData{Resource: consumer.ID}
						err = ctx.ExecuteDecode(consume.Resource, data, &id)
						if err != nil {
							errs = errors.Join(errs, err)
							continue
						}
					}
					if consume.Converter != "" {
						val, err = consume.Convert(val, id, ctx)
						if err != nil {
							errs = errors.Join(errs, err)
							continue
						}
					}
					delays = append(delays, DelayedConsumption{
						Value:        val,
						Resource:     id,
						PropertyPath: consume.PropertyPath,
					})
					continue
				}

				err = consume.Consume(val, ctx, consumer)
				if err != nil {
					errs = errors.Join(errs, err)
					continue
				}
			}
		}
	}
	return delays, errs
}

func (c *ConsumptionObject) Convert(value any, res construct.ResourceId, ctx DynamicValueContext) (any, error) {
	if c.Converter == "" {
		return value, fmt.Errorf("no converter specified")
	}
	if c.PropertyPath == "" {
		return value, fmt.Errorf("no property path specified")
	}
	t, err := template.New("config").Funcs(template.FuncMap{
		"sub": func(a int, b int) int {
			return a - b
		},
	},
	).Parse(c.Converter)
	if err != nil {
		return value, err
	}
	buf := new(bytes.Buffer)
	if err := t.Execute(buf, value); err != nil {
		return value, err
	}
	bstr := strings.TrimSpace(buf.String())
	val, err := TransformToPropertyValue(res, c.PropertyPath, bstr, ctx, DynamicValueData{Resource: res})
	if err == nil {
		return val, nil
	}
	return val, nil
}

func (c *ConsumptionObject) Emit(ctx DynamicValueContext, resource construct.ResourceId) (any, error) {
	if c.Value == "" {
		return nil, fmt.Errorf("no value specified")
	}
	if c.Model == "" {
		return nil, fmt.Errorf("no property path specified")
	}
	if c.Resource != "" {
		data := DynamicValueData{Resource: resource}
		err := ctx.ExecuteDecode(c.Resource, data, resource)
		if err != nil {
			return nil, err
		}
	}
	model := ctx.KnowledgeBase.GetModel(c.Model)
	data := DynamicValueData{Resource: resource}
	val, err := model.GetObjectValue(c.Value, ctx, data)
	if err != nil {
		return nil, err
	}
	if err != nil {
		return val, err
	}
	return val, nil
}

func (c *ConsumptionObject) Consume(val any, ctx DynamicValueContext, resource *construct.Resource) error {
	var err error
	if c.Resource != "" {
		newId := construct.ResourceId{}
		data := DynamicValueData{Resource: resource.ID}
		err = ctx.ExecuteDecode(c.Resource, data, &newId)
		if err != nil {
			return err
		}
		resource, err = ctx.Graph.Vertex(newId)
		if err != nil {
			return err
		}
	}
	if c.Converter != "" {
		val, err = c.Convert(val, resource.ID, ctx)
		if err != nil {
			return err
		}
	}
	err = resource.SetProperty(c.PropertyPath, val)
	if err != nil {
		return err
	}
	return nil
}
