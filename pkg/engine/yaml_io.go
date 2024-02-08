package engine

import (
	"github.com/klothoplatform/klotho/pkg/construct"
	"github.com/klothoplatform/klotho/pkg/engine/constraints"
	"gopkg.in/yaml.v3"
)

// FileFormat is used for engine input/output to render or read a YAML file
// An example yaml file is:
//
//	constraints:
//	    - scope: application
//	      operator: add
//	      node: p:t:a
//	resources:
//	    p:t:a:
//	    p:t:b:
//	edges:
//	    p:t:a -> p:t:b:
type FileFormat struct {
	Constraints constraints.Constraints
	Graph       construct.Graph
}

func (ff FileFormat) MarshalYAML() (interface{}, error) {
	constraintsNode := &yaml.Node{}
	err := constraintsNode.Encode(ff.Constraints.ToList())
	if err != nil {
		return nil, err
	}
	if len(constraintsNode.Content) == 0 {
		// this makes `constraints: {}` like `constraints:`
		constraintsNode = &yaml.Node{
			Kind:  yaml.ScalarNode,
			Tag:   "!!null",
			Value: "",
		}
	}

	graphNode := &yaml.Node{}
	err = graphNode.Encode(construct.YamlGraph{Graph: ff.Graph})
	if err != nil {
		return nil, err
	}

	root := &yaml.Node{
		Kind: yaml.MappingNode,
		Content: []*yaml.Node{
			{
				Kind:  yaml.ScalarNode,
				Value: "constraints",
			},
			constraintsNode,
		},
	}
	root.Content = append(root.Content, graphNode.Content...)
	return root, nil
}

func (ff *FileFormat) UnmarshalYAML(node *yaml.Node) error {
	var constraints struct {
		Constraints constraints.ConstraintList `yaml:"constraints"`
	}
	err := node.Decode(&constraints)
	if err != nil {
		return err
	}
	ff.Constraints, err = constraints.Constraints.ToConstraints()
	if err != nil {
		return err
	}

	var graph construct.YamlGraph
	err = node.Decode(&graph)
	if err != nil {
		return err
	}
	ff.Graph = graph.Graph

	return nil
}
