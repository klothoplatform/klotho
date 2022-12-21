package golang

import (
	"github.com/klothoplatform/klotho/pkg/query"
	sitter "github.com/smacker/go-tree-sitter"
)

// doQuery is a thin wrapper around `query.Exec` to use go as the Language.
func doQuery(c *sitter.Node, q string) query.NextMatchFunc {
	return query.Exec(language, c, q)
}
