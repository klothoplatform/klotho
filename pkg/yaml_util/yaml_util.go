package yaml_util

import (
	"bytes"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
	"strings"
)

type CheckMode bool

const (
	Lenient = CheckMode(false)
	Strict  = CheckMode(true)
)

// SetValue upserts the value the content yaml to a given value, specified by a dotted path. For example, setting
// `foo.bar.baz` to `hello, world` is equivalent to upserting the following yaml:
//
//	foo:
//	    bar:
//	        baz: hello, world
//
// This method will make a best effort to preserve comments, as per the `yaml` package's abilities. You may overwrite
// scalars, but you may not overwrite a non-scalar. You may also specify a path that doesn't exist in the source yaml,
// as long as none of the paths correspond to existing elements other than yaml mappings.
func SetValue(content []byte, optionPath string, optionValue string) ([]byte, error) {
	// General approach:
	// 1) convert the yaml into a map, using yaml
	// 2a) find the node tree's value at the specified path
	// 2b) set that node's value, assuming it's a scalar (or empty)
	// 2) write the node tree back into bytes. this will preserve comments and such

	// step 1
	var tree yaml.Node
	if err := yaml.Unmarshal(content, &tree); err != nil {
		return nil, err
	}

	// step 2a
	segments := strings.Split(optionPath, ".")
	var topNode *yaml.Node
	if len(tree.Content) == 0 {
		topNode = &yaml.Node{Kind: yaml.MappingNode}
	} else {
		topNode = tree.Content[0] // the tree's root is a DocumentNode; we assume one document
	}
	setOptionAtNode := topNode
	for _, segment := range segments[:len(segments)-1] {
		if setOptionAtNode.Kind != yaml.MappingNode {
			return nil, errors.Errorf(`can't set the path "%s"'`, optionPath)
		}
		if child := findChild(setOptionAtNode.Content, segment); child != nil {
			setOptionAtNode = child
		} else {
			newSubMap := &yaml.Node{Kind: yaml.MappingNode}
			setOptionAtNode.Content = append(setOptionAtNode.Content, &yaml.Node{
				Kind:  yaml.ScalarNode,
				Value: segment,
			})
			setOptionAtNode.Content = append(setOptionAtNode.Content, newSubMap)
			setOptionAtNode = newSubMap
		}
	}

	// step 2b
	if setOptionAtNode.Kind != yaml.MappingNode {
		return nil, errors.Errorf(`can't set the path "%s"'`, optionPath)
	}
	lastSegment := segments[len(segments)-1]
	if currValue := findChild(setOptionAtNode.Content, lastSegment); currValue != nil {
		if currValue.Kind != yaml.ScalarNode {
			return nil, errors.Errorf(`"%s" cannot be a scalar`, optionPath)
		}
		currValue.Tag = "" // if the existing type isn't a string, we want to reset it
		currValue.Value = optionValue
	} else {
		setOptionAtNode.Content = append(setOptionAtNode.Content, &yaml.Node{
			Kind:  yaml.ScalarNode,
			Value: lastSegment,
		})
		setOptionAtNode.Content = append(setOptionAtNode.Content, &yaml.Node{
			Kind:  yaml.ScalarNode,
			Value: optionValue,
		})
	}

	// step 3
	return yaml.Marshal(topNode)
}

// CheckValid validates that the given yaml actually represents the type provided, and returns a non-nil error
// describing the problem if it doesn't. You need to explicitly provide the type to be checked:
//
//	CheckValid[MyCoolType](contents)
//
// The strict flag governs whether the check will allow unknown fields.
func CheckValid[T any](content []byte, mode CheckMode) error {
	if strings.TrimSpace(string(content)) == "" {
		// the decoder will fail on this (EOF), but we want to consider it valid yaml
		return nil
	}
	var ignored T
	decoder := yaml.NewDecoder(bytes.NewReader(content))
	decoder.KnownFields(bool(mode))
	return decoder.Decode(&ignored)
}

// YamlErrors returns the yaml.TypeError errors if the given err is a TypeError; otherwise, it just returns a
// single-element array of the given error's string (disregarding any wrapped errors).
func YamlErrors(err error) []string {
	switch err := err.(type) {
	case *yaml.TypeError:
		return err.Errors
	default:
		return []string{err.Error()}
	}

}

func findChild(within []*yaml.Node, named string) *yaml.Node {
	for i := 0; i < len(within); i += 2 {
		node := within[i]
		if node.Kind == yaml.ScalarNode && node.Value == named {
			return within[i+1]
		}
	}
	return nil
}
