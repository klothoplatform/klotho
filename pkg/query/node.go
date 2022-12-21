package query

import (
	"regexp"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
)

func NodeContentStartWith(node *sitter.Node, src []byte, s string) bool {
	content := node.Content(src)

	if s != "" && strings.HasPrefix(content, s) {
		return true
	}
	return false
}

func NodeContentEquals(node *sitter.Node, src []byte, s string) bool {
	content := node.Content(src)
	if s != "" && content == s {
		return true
	}
	return false
}

func NodeContentRegex(node *sitter.Node, src []byte, regex *regexp.Regexp) bool {
	content := node.Content(src)
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
