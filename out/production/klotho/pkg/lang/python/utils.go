package python

import (
	"fmt"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
)

// stringLiteralContent returns the string literal content of the supplied node
// after stripping any enclosing quotes and un-escaping any quotes of the same type inside the string.
//
// Passing in a Node that references a b-string will result in an error.
func stringLiteralContent(node *sitter.Node) (string, error) {
	if node.Type() != "string" {
		return "", fmt.Errorf("node of type %s cannot be parsed as string literal content", node.Type())
	}

	nodeContent := node.Content()
	if nodeContent == "" {
		return "", nil
	}

	psLen := 0
	if strings.HasPrefix(nodeContent, `"`) || strings.HasPrefix(nodeContent, `'`) {
		psLen = 1
	}
	if strings.HasPrefix(nodeContent, `"""`) || strings.HasPrefix(nodeContent, `'''`) {
		psLen = 3
	}
	if psLen == 0 {
		return "", fmt.Errorf("unsupported Python string format: %s", nodeContent[0:1])
	}

	return nodeContent[psLen : len(nodeContent)-psLen], nil

}
