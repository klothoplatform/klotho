package knowledgebase2

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"text/template"
)

type (
	SanitizeTmpl struct {
		template *template.Template
	}

	// SanitizeError is returned when a value is sanitized if the input is not valid. The Sanitized field
	// is always the same type as the Input field.
	SanitizeError struct {
		Input     any
		Sanitized any
	}
)

func NewSanitizationTmpl(name string, tmpl string) (*SanitizeTmpl, error) {
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
	return &SanitizeTmpl{
		template: t,
	}, err
}

var sanitizeBufs = sync.Pool{
	New: func() interface{} {
		return new(bytes.Buffer)
	},
}

func (t SanitizeTmpl) Execute(value string) (string, error) {
	buf := sanitizeBufs.Get().(*bytes.Buffer)
	defer sanitizeBufs.Put(buf)
	buf.Reset()

	err := t.template.Execute(buf, value)
	if err != nil {
		return value, fmt.Errorf("could not execute sanitize name template on %q: %w", value, err)
	}
	return strings.TrimSpace(buf.String()), nil
}

func (t SanitizeTmpl) Check(value string) error {
	sanitized, err := t.Execute(value)
	if err != nil {
		return err
	}
	if sanitized != value {
		return &SanitizeError{
			Input:     value,
			Sanitized: sanitized,
		}
	}
	return nil
}

func (err SanitizeError) Error() string {
	return fmt.Sprintf("invalid value %q, suggested value: %q", err.Input, err.Sanitized)
}
