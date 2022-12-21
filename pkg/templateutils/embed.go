package templateutils

import (
	"embed"
	"text/template"

	sprig "github.com/Masterminds/sprig/v3"
)

func MustTemplate(fs embed.FS, name string) *template.Template {
	content, err := fs.ReadFile(name)
	if err != nil {
		panic(err)
	}
	t, err := template.New(name).
		Funcs(Funcs).
		Funcs(sprig.HermeticTxtFuncMap()).
		Parse(string(content))
	if err != nil {
		panic(err)
	}
	return t
}
