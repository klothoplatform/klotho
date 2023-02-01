package csharp

import (
	"github.com/klothoplatform/klotho/pkg/query"
	sitter "github.com/smacker/go-tree-sitter"
)

// DoQuery is a thin wrapper around `query.Exec` to use C# as the Language.
func DoQuery(c *sitter.Node, q string) query.NextMatchFunc {
	return query.Exec(Language, c, q)
}
