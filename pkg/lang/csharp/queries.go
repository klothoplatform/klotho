package csharp

import (
	_ "embed"
	"github.com/klothoplatform/klotho/pkg/query"
	sitter "github.com/smacker/go-tree-sitter"
)

// DoQuery is a thin wrapper around `query.Exec` to use C# as the Language.
func DoQuery(c *sitter.Node, q string) query.NextMatchFunc {
	return query.Exec(Language, c, q)
}

// AllMatches completes processes all query matches immediately and returns a []query.MatchNodes.
func AllMatches(c *sitter.Node, q string) []query.MatchNodes {
	return query.Collect(DoQuery(c, q))
}

var (

	//go:embed queries/using_directives.scm
	usingDirectives string

	//go:embed queries/type_declarations.scm
	typeDeclarations string

	//go:embed queries/method_declarations.scm
	methodDeclarations string
)
