package lang

import (
	"fmt"

	sitter "github.com/smacker/go-tree-sitter"
)

func PrintNodes(n map[string]*sitter.Node) string {
	s := make(map[string]string)
	for k, v := range n {
		s[k] = fmt.Sprintf(`(%s) "%s"`, v.Type(), v.Content())
	}
	return fmt.Sprintf("%v", s)
}
