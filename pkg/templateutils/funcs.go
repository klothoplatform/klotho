package templateutils

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/klothoplatform/klotho/pkg/construct"
)

var Funcs = template.FuncMap{
	"joinString": strings.Join,

	"json": func(v any) (string, error) {
		buf := new(bytes.Buffer)
		enc := json.NewEncoder(buf)
		if err := enc.Encode(v); err != nil {
			return "", err
		} else {
			return strings.TrimSpace(buf.String()), nil
		}
	},

	"jsonPretty": func(v any) (string, error) {
		buf := new(bytes.Buffer)
		enc := json.NewEncoder(buf)
		enc.SetIndent("", "    ")
		if err := enc.Encode(v); err != nil {
			return "", err
		} else {
			return strings.TrimSpace(buf.String()), nil
		}
	},

	"keyNames": func(keys []construct.Construct) (ns []string) {
		for _, k := range keys {
			ns = append(ns, k.Id().Name)
		}
		return
	},

	"fileBase": func(path string) string {
		return filepath.Base(path)
	},

	"fileTrimExt": func(path string) string {
		return strings.TrimSuffix(path, filepath.Ext(path))
	},

	"fileSep": func() string {
		return string(filepath.Separator)
	},

	"replaceAll": func(s string, old string, new string) string {
		return strings.ReplaceAll(s, old, new)
	},
}
