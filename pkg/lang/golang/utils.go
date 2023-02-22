package golang

import (
	"fmt"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
)

// stringLiteralContent returns the node content with outer quotes removed
func stringLiteralContent(node *sitter.Node) string {
	if node.Type() != "interpreted_string_literal" {
		panic(fmt.Errorf("node of type %s cannot be parsed as interpreted string literal content", node.Type()))
	}

	nodeContent := node.Content()
	if nodeContent == "" {
		return ""
	}

	psLen := 0
	if strings.HasPrefix(nodeContent, `"`) || strings.HasPrefix(nodeContent, `'`) {
		psLen = 1
	}
	if psLen == 0 {
		panic(fmt.Errorf("unsupported Go string format: %s", nodeContent[0:1]))
	}

	return nodeContent[psLen : len(nodeContent)-psLen]

}
