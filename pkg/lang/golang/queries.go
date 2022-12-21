package golang

import (
	_ "embed"
)

//go:embed queries/expose/chirouter_assignment.scm
var findRouterAssignment string

//go:embed queries/expose/http_listen_serve.scm
var findHttpListen string

//go:embed queries/expose/verb.scm
var findExposeVerb string

//go:embed queries/expose/imports.scm
var findImports string
