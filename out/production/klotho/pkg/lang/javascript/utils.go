package javascript

import (
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
)

func StringLiteralContent(node *sitter.Node) string {
	return strings.Trim(node.Content(), `'"`)
}
