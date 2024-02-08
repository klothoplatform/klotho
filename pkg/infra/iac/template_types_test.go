package iac

import (
	"bytes"
	"fmt"
	"io"
	"testing"
	"text/template"

	"github.com/stretchr/testify/assert"
)

func Test_appliedOutputs_Render(t *testing.T) {
	assert := assert.New(t)

	outputs := appliedOutputs{
		{Ref: "a.prop1", Name: "a"},
		{Ref: "a.prop1", Name: "a"}, // dupe
		{Ref: "a.aaa", Name: "aa"},  // non-alphabetic
		{Ref: "b.prop1", Name: "b"},
	}
	if !assert.NoError(outputs.dedupe()) {
		t.FailNow()
	}

	// duplicate removed
	assert.Len(outputs, 3)

	buf := new(bytes.Buffer)
	err := outputs.Render(buf, func(out io.Writer) error {
		_, err := fmt.Fprint(out, "TEST")
		return err
	})
	if !assert.NoError(err) {
		t.FailNow()
	}

	expected := `pulumi.all([a.aaa, a.prop1, b.prop1])
.apply(([aa, a, b]) => {
    return TEST
})`

	assert.Equal(expected, buf.String())
}

func Test_appliedOutputs_dedupe(t *testing.T) {
	assert := assert.New(t)

	outputs := appliedOutputs{
		{Ref: "a.prop1", Name: "a"},
		{Ref: "a.aaa", Name: "a"},
	}

	assert.Error(outputs.dedupe())
}

func Test_templatetypes(t *testing.T) {
	data := map[string]any{
		"int":     1,
		"str":     "foo",
		"jsonStr": jsonValue{Raw: "bar"},
		"jsonInt": jsonValue{Raw: 2},
		"jsonMap": jsonValue{Raw: map[string]any{
			"dog": "good",
		}},
		"emptyStr":         "",
		"emptyJsonStr":     jsonValue{Raw: ""},
		"emptryTmplString": templateString(""),
	}
	tests := []struct {
		name     string
		template string
		want     string
		wantErr  bool
	}{
		{
			name:     "raw values",
			template: `{{ .str }}: {{ .int }}`,
			want:     "foo: 1",
		},
		{
			name:     "json values",
			template: `{{ .jsonStr }}: {{ .jsonInt }}`,
			want:     `"bar": 2`,
		},
		{
			name:     "json map",
			template: `{{ .jsonMap }}`,
			want:     `{"dog":"good"}`,
		},
		{
			name:     "conditional empty raw",
			template: `{{ if .emptyStr }}value: {{.emptyStr}}{{ else }}empty: {{.emptyStr}}{{ end }}`,
			want:     `empty: `,
		},
		{
			name:     "conditional empty json",
			template: `{{ if .emptyJsonStr }}value: {{.emptyJsonStr}}{{ else }}empty: {{.emptyJsonStr}}{{ end }}`,
			want:     `value: ""`,
		},
		{
			name:     "conditional empty tmpl string",
			template: `{{ if .emptryTmplString }}value: {{.emptryTmplString}}{{ else }}empty: {{.emptryTmplString}}{{ end }}`,
			want:     `empty: ""`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert := assert.New(t)

			tmpl, err := template.New(tt.name).Parse(tt.template)
			if !assert.NoError(err) {
				return
			}
			buf := new(bytes.Buffer)
			err = tmpl.Execute(buf, data)
			if tt.wantErr {
				assert.Error(err)
				return
			}
			if !assert.NoError(err) {
				return
			}
			assert.Equal(tt.want, buf.String())
		})
	}
}
