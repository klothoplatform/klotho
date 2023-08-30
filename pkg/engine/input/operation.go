package input

import (
	"github.com/klothoplatform/klotho/pkg/construct"
	"gopkg.in/yaml.v3"
)

type Operation struct {
	Node     construct.ResourceId `yaml:"node"`
	Source   construct.ResourceId `yaml:"source"`
	Target   construct.ResourceId `yaml:"target"`
	Operator string               `yaml:"operator"`
	params   *yaml.Node           `yaml:"-"`
}

func (o *Operation) UnmarshalYAML(node *yaml.Node) error {
	type alias Operation
	var a alias
	err := node.Decode(&a)
	if err != nil {
		return err
	}
	*o = Operation(a)
	o.params = node
	err = validateNodes(o.Node, o.Source, o.Target)
	return err
}

func (o *Operation) DecodeParams(v interface{}) error {
	return o.params.Decode(v)
}
