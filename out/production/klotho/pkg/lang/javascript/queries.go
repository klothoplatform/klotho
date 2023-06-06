package javascript

import (
	_ "embed"

	"github.com/klothoplatform/klotho/pkg/query"
	sitter "github.com/smacker/go-tree-sitter"
)

var (
	//go:embed queries/persist/kv.scm
	persistKV string

	//go:embed queries/persist/secret.scm
	persistSecret string

	//go:embed queries/persist/orm.scm
	persistORM string

	//go:embed queries/persist/redis.scm
	persistRedis string

	//go:embed queries/expose/express/verb.scm
	exposeVerb string

	//go:embed queries/expose/listener.scm
	exposeListener string

	//go:embed queries/expose/express/routers.scm
	exposeRouters string

	//go:embed queries/expose/express/express.scm
	expressApp string

	//go:embed queries/expose/nestJs/factory.scm
	nestJsFactory string

	//go:embed queries/expose/nestJs/module.scm
	nestJsModule string

	//go:embed queries/expose/nestJs/controller.scm
	nestJsController string

	//go:embed queries/expose/nestJs/routes.scm
	nestJsRoute string

	//go:embed queries/proxy/usage.scm
	proxyUsage string

	//go:embed queries/proxy/async.scm
	proxyAsync string

	//go:embed queries/proxy/export.scm
	proxyExport string

	//go:embed queries/modules/import.scm
	modulesImport string

	//go:embed queries/modules/export.scm
	modulesExport string

	//go:embed queries/modules/default.scm
	modulesDefault string

	//go:embed queries/pubsub/publisher.scm
	pubsubPublisher string

	//go:embed queries/pubsub/subscriber.scm
	pubsubSubscriber string

	//go:embed queries/declare_and_instantiate.scm
	declareAndInstantiate string

	//go:embed queries/exported_var.scm
	exportedVar string

	//go:embed queries/method_invocation.scm
	methodInvocation string

	//go:embed queries/function_invocation.scm
	functionInvocation string
)

// DoQuery is a thin wrapper around `query.Exec` to use javascript as the Language.
func DoQuery(c *sitter.Node, q string) query.NextMatchFunc {
	return query.Exec(Language, c, q)
}
