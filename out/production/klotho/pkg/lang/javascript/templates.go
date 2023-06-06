package javascript

import (
	"embed"
	"io"

	"github.com/klothoplatform/klotho/pkg/templateutils"
)

//go:embed *.tmpl
var templateFiles embed.FS

var tmplRuntimeImport = templateutils.MustTemplate(templateFiles, "runtime_import.js.tmpl")
var tmplPubsubProxy = templateutils.MustTemplate(templateFiles, "proxy_emitter.js.tmpl")

type RuntimeImport struct {
	VarName  string
	FilePath string
}

func NewRuntimeImport(ctx RuntimeImport, w io.Writer) error {
	return tmplRuntimeImport.Execute(w, ctx)
}

type EmitterSubscriberProxyEntry struct {
	ImportPath string
	VarName    string
	Events     []string
}

type EmitterSubscriberProxy struct {
	RuntimeImport string
	Path          string
	Entries       []EmitterSubscriberProxyEntry
}
