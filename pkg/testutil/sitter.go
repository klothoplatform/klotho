package testutil

import (
	sitter "github.com/smacker/go-tree-sitter"
)

// FindNodeByContent returns a node whose content is the given content string. This could be a descendent of the given
// node. If there are multiple such nodes, this will return one of them, but it's unspecified which.
func FindNodeByContent(node *sitter.Node, content string) *sitter.Node {
	if node.Content() == content {
		return node
	}
	for childIdx := 0; childIdx < int(node.ChildCount()); childIdx += 1 {
		child := node.Child(childIdx)
		if result := FindNodeByContent(child, content); result != nil {
			return result
		}
	}
	return nil
}
