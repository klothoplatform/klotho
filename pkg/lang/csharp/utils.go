package csharp

import (
	sitter "github.com/smacker/go-tree-sitter"
	"strings"
)

func stringLiteralContent(node *sitter.Node) string {
	return strings.Trim(strings.TrimPrefix(node.Content(), "@"), `"`)
}
