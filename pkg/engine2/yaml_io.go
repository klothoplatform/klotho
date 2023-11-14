package engine2

import (
	"github.com/klothoplatform/klotho/pkg/construct2"
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
	Graph       construct2.Graph
}

func (ff FileFormat) MarshalYAML() (interface{}, error) {
	constraintsNode := &yaml.Node{}
	err := constraintsNode.Encode(ff.Constraints)
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
	err = graphNode.Encode(construct2.YamlGraph{Graph: ff.Graph})
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
		Constraints constraints.Constraints `yaml:"constraints"`
	}
	err := node.Decode(&constraints)
	if err != nil {
		return err
	}
	ff.Constraints = constraints.Constraints

	var graph construct2.YamlGraph
	err = node.Decode(&graph)
	if err != nil {
		return err
	}
	ff.Graph = graph.Graph

	return nil
}
