package csharp

import (
	sitter "github.com/smacker/go-tree-sitter"
	"strings"
)

func stringLiteralContent(node *sitter.Node) string {
	if node == nil {
		return ""
	}
	return strings.Trim(strings.TrimPrefix(node.Content(), "@"), `"`)
}
