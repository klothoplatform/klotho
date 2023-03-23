package golang

import (
	_ "embed"
)

//go:embed queries/imports.scm
var findImports string

//go:embed queries/find_function_call.scm
var findFunctionCall string

//go:embed queries/expose/chirouter_assignment.scm
var findRouterAssignment string

//go:embed queries/expose/http_listen_serve.scm
var findHttpListen string

//go:embed queries/expose/verb.scm
var findExposeVerb string

//go:embed queries/expose/router_mounts.scm
var findRouterMounts string

//go:embed queries/expose/function.scm
var findFunction string

//go:embed queries/expose/router_middleware.scm
var routerMiddleware string

//go:embed queries/package.scm
var packageQuery string

//go:embed queries/gocloud/file_bucket.scm
var fileBucket string

//go:embed queries/gocloud/open_variable.scm
var openVariable string
