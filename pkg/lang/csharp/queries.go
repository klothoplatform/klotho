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

	//go:embed queries/structural/using_directives.scm
	usingDirectives string

	//go:embed queries/structural/type_declarations.scm
	typeDeclarations string

	//go:embed queries/structural/method_declarations.scm
	methodDeclarations string

	//go:embed queries/structural/field_declarations.scm
	fieldDeclarations string

	//go:embed queries/expose/configured_app.scm
	configuredApp string

	//go:embed queries/expose/use_endpoints_format.scm
	useEndpointsFormat string

	//go:embed queries/expose/http_method_attribute.scm
	httpMethodAttribute string

	//go:embed queries/expose/route_attribute.scm
	exposeRouteAttribute string

	//go:embed queries/expose/area_attribute.scm
	exposeAreaAttribute string

	//go:embed queries/expose/map_route.scm
	exposeMapRoute string
)
