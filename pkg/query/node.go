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

func NodeContentRegex(node *sitter.Node, regex *regexp.Regexp) bool {
	content := node.Content()
	return regex.MatchString(content)
}

func FirstAncestorOfType(node *sitter.Node, ptype string) *sitter.Node {
	for n := node; n != nil; n = n.Parent() {
		if n.Type() == ptype {
			return n
		}
	}
	return nil
}

func NodeContentOrEmpty(node *sitter.Node) string {
	if node == nil {
		return ""
	}
	return node.Content()
}
