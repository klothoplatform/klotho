package templateutils

import (
	"embed"
	"strings"
	"text/template"

	sprig "github.com/Masterminds/sprig/v3"
)

func MustTemplate(fs embed.FS, name string) *template.Template {
	content, err := fs.ReadFile(name)
	if err != nil {
		panic(err)
	}
	t, err := template.New(name).
		Funcs(mustTemplateFuncs).
		Funcs(sprig.HermeticTxtFuncMap()).
		Parse(string(content))
	if err != nil {
		panic(err)
	}
	return t
}

var mustTemplateFuncs = template.FuncMap{
	"joinString": strings.Join,

	"json":       ToJSON,
	"jsonPretty": ToJSONPretty,

	"fileBase":    FileBase,
	"fileTrimExt": FileTrimExtFunc,
	"fileSep":     FileSep,

	"replaceAll": ReplaceAll,
}
