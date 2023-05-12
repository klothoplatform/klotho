package testutil

import (
	sitter "github.com/smacker/go-tree-sitter"
)

// FindNodeByContent returns a node whose content is the given content string. If there are multiple such nodes, this
// will return one of them, but it's unspecified which.
func FindNodeByContent(tree *sitter.Tree, content string) *sitter.Node {
	var findNode0 func(node *sitter.Node) *sitter.Node
	findNode0 = func(node *sitter.Node) *sitter.Node {
		if node.Content() == content {
			return node
		}
		for childIdx := 0; childIdx < int(node.ChildCount()); childIdx += 1 {
			child := node.Child(childIdx)
			if result := findNode0(child); result != nil {
				return result
			}
		}
		return nil
	}
	return findNode0(tree.RootNode())
}
