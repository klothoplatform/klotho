package query

import (
	"regexp"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
)

func NodeContentStartWith(node *sitter.Node, s string) bool {
	content := node.Content()

	if s != "" && strings.HasPrefix(content, s) {
		return true
	}
	return false
}

func NodeContentEquals(node *sitter.Node, s string) bool {
	content := node.Content()
	if s != "" && content == s {
		return true
	}
	return false
}

func NodeContentIn[K any](node *sitter.Node, m map[string]K) bool {
	_, keyExists := m[node.Content()]
	return keyExists
}

func NodeContentRegex(node *sitter.Node, regex *regexp.Regexp) bool {
	content := node.Content()
	return regex.MatchString(content)
}

func NodeContentOrEmpty(node *sitter.Node) string {
	if node == nil {
		return ""
	}
	return node.Content()
}

func FirstChildOfType(node *sitter.Node, ctype string) *sitter.Node {
	for i := 0; i < int(node.ChildCount()); i++ {
		n := node.Child(i)
		if n.Type() == ctype {
			return n
		}
	}
	return nil
}

func FirstAncestorOfType(node *sitter.Node, ptype string) *sitter.Node {
	for n := node; n != nil; n = n.Parent() {
		if n.Type() == ptype {
			return n
		}
	}
	return nil
}

func AncestorsOfType(node *sitter.Node, aType string) []*sitter.Node {
	var ancestors []*sitter.Node
	for n := node; n != nil; n = n.Parent() {
		if n.Type() == aType {
			ancestors = append(ancestors, n)
		}
	}
	return ancestors
}
