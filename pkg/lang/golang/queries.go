package golang

import (
	_ "embed"
)

//go:embed queries/imports.scm
var findImports string

//go:embed queries/find_args.scm
var findArgs string

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

//go:embed queries/package.scm
var packageQuery string

//go:embed queries/gocloud/file_bucket.scm
var fileBucket string
