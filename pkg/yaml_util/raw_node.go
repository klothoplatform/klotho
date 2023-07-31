package yaml_util

import "gopkg.in/yaml.v3"

type RawNode struct{ *yaml.Node }

func (n *RawNode) UnmarshalYAML(value *yaml.Node) error {
	n.Node = value
	return nil
}
