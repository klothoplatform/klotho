package python

import (
	_ "embed"

	"github.com/klothoplatform/klotho/pkg/query"
	sitter "github.com/smacker/go-tree-sitter"
)

var (
	//go:embed queries/expose/verb.scm
	exposeVerb string

	//go:embed queries/expose/fastapi_assignment.scm
	exposeFastAPIAssignment string

	//go:embed queries/persist/cache_assignment.scm
	persistKV string

	//go:embed queries/persist/aiofiles_open.scm
	aiofilesOpen string

	//go:embed queries/persist/orm.scm
	orm string

	//go:embed queries/persist/redis.scm
	redis string

	//go:embed queries/find_imports.scm
	findImports string

	//go:embed queries/find_function_calls.scm
	findFunctionCalls string

	//go:embed queries/find_qualified_attr_usage.scm
	FindQualifiedAttrUsage string
)

// DoQuery is a thin wrapper around `query.Exec` to use python as the Language.
func DoQuery(c *sitter.Node, q string) query.NextMatchFunc {
	return query.Exec(Language, c, q)
}
