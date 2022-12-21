package golang

import (
	"fmt"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
)

// stringLiteralContent returns the node content with outter quotes removed
func stringLiteralContent(node *sitter.Node, program []byte) string {
	if node.Type() != "interpreted_string_literal" {
		panic(fmt.Errorf("node of type %s cannot be parsed as interpreted string literal content", node.Type()))
	}

	nodeContent := node.Content(program)
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
