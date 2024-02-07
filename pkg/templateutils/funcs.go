package templateutils

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"strings"
	"text/template"
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
