package template

import (
	"github.com/klothoplatform/klotho/pkg/construct"
	"github.com/klothoplatform/klotho/pkg/k2/constructs/template/inputs"
	"github.com/klothoplatform/klotho/pkg/k2/constructs/template/property"
	"gopkg.in/yaml.v3"
)

type Properties struct {
	propertyMap property.PropertyMap
}

func NewProperties(properties property.PropertyMap) *Properties {
	if properties == nil {
		properties = make(property.PropertyMap)
	}

	return &Properties{
		propertyMap: properties,
	}
}

func (p *Properties) Clone() property.Properties {
	newProps := Properties{
		propertyMap: p.propertyMap.Clone(),
	}
	return &newProps
}

func (p *Properties) ForEach(c construct.Properties, f func(p property.Property) error) error {
	return p.propertyMap.ForEach(c, f)
}

func (p *Properties) Get(key string) (property.Property, bool) {
	return p.propertyMap.Get(key)
}

func (p *Properties) Set(key string, value property.Property) {
	p.propertyMap.Set(key, value)
}

func (p *Properties) Remove(key string) {
	p.propertyMap.Remove(key)
}

func (p *Properties) AsMap() map[string]property.Property {
	return p.propertyMap
}

func (p *Properties) UnmarshalYAML(node *yaml.Node) error {
	if p.propertyMap == nil {
		p.propertyMap = make(property.PropertyMap)
	}

	ip := make(inputs.InputTemplateMap)
	if err := node.Decode(&ip); err != nil {
		return err
	}
	converted, err := ip.Convert()
	if err != nil {
		return err
	}
	for k, v := range converted {
		p.propertyMap[k] = v
	}
	return nil
}
