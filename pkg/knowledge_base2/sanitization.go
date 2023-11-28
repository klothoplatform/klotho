package knowledgebase2

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"regexp"
	"strings"
	"text/template"
)

type (
	SanitizeTmpl struct {
		template *template.Template
	}
)

func NewSanitizationTmpl(name string, tmpl string) (SanitizeTmpl, error) {
	t, err := template.New(name + "/sanitize").
		Funcs(template.FuncMap{
			"replace": func(pattern, replace, name string) (string, error) {
				re, err := regexp.Compile(pattern)
				if err != nil {
					return name, err
				}
				return re.ReplaceAllString(name, replace), nil
			},

			"length": func(min, max int, name string) string {
				if len(name) < min {
					return name + strings.Repeat("0", min-len(name))
				}
				if len(name) > max {
					base := name[:max-8]
					h := sha256.New()
					fmt.Fprint(h, name)
					x := fmt.Sprintf("%x", h.Sum(nil))
					return base + x[:8]
				}
				return name
			},

			"lower": strings.ToLower,
			"upper": strings.ToUpper,
		}).
		Parse(tmpl)
	return SanitizeTmpl{
		template: t,
	}, err
}

func (t SanitizeTmpl) Execute(name string) (string, error) {
	buf := new(bytes.Buffer)
	err := t.template.Execute(buf, name)
	if err != nil {
		return name, fmt.Errorf("could not execute sanitize name template on %q: %w", name, err)
	}
	return strings.TrimSpace(buf.String()), nil
}
