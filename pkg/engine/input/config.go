package input

import (
	"github.com/klothoplatform/klotho/pkg/construct"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Node   construct.ResourceId `yaml:"node"`
	Source construct.ResourceId `yaml:"source"`
	Target construct.ResourceId `yaml:"target"`
	params *yaml.Node           `yaml:"-"`
}

func (c *Config) UnmarshalYAML(node *yaml.Node) error {
	type alias Config
	var a alias
	err := node.Decode(&a)
	if err != nil {
		return err
	}
	*c = Config(a)
	c.params = node
	err = validateNodes(c.Node, c.Source, c.Target)
	return err
}

func (c *Config) DecodeParams(v interface{}) error {
	return c.params.Decode(v)
}
