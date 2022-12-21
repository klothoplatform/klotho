package javascript

import (
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
)

func StringLiteralContent(node *sitter.Node, program []byte) string {
	return strings.Trim(node.Content(program), `'"`)
}
