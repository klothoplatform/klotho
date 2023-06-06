package iac2

import (
	"bytes"
	"embed"
	"fmt"
	"text/template"

	"github.com/klothoplatform/klotho/pkg/config"
	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/provider/aws/resources"
	"github.com/klothoplatform/klotho/pkg/templateutils"
)

type (
	Plugin struct {
		Config *config.Application
	}
)

func (p Plugin) Name() string {
	return "pulumi2"
}

//go:embed Pulumi.yaml.tmpl Pulumi.dev.yaml.tmpl
var files embed.FS

var pulumiBase = templateutils.MustTemplate(files, "Pulumi.yaml.tmpl")
var pulumiStack = templateutils.MustTemplate(files, "Pulumi.dev.yaml.tmpl")

func (p Plugin) Translate(cloudGraph *core.ResourceGraph) ([]core.File, error) {

	// TODO We'll eventually want to split the output into different files, but we don't know exactly what that looks
	// like yet. For now, just write to a single file, "new_index.ts"
	buf := &bytes.Buffer{}
	tc := CreateTemplatesCompiler(cloudGraph)

	// index.ts
	if err := tc.RenderImports(buf); err != nil {
		return nil, err
	}
	if _, err := buf.WriteString("\n\n"); err != nil {
		return nil, err
	}

	buf.WriteString("const kloConfig: pulumi.Config = new pulumi.Config('klo')\n")
	buf.WriteString("const protect = kloConfig.getBoolean('protect') ?? false")
	buf.WriteString(`
const awsConfig = new pulumi.Config('aws')
const awsProfile = awsConfig.get('profile')` + "\n")
	if err := tc.renderResource(buf, resources.NewAccountId()); err != nil {
		return nil, err
	}
	buf.WriteString("\n")
	if err := tc.renderResource(buf, resources.NewRegion()); err != nil {
		return nil, err
	}
	buf.WriteString("\n\n")

	if err := tc.RenderBody(buf); err != nil {
		return nil, err
	}

	indexTs := &core.RawFile{
		FPath:   `index.ts`,
		Content: buf.Bytes(),
	}

	pJson, err := tc.RenderPackageJSON()
	if err != nil {
		return nil, err
	}
	pJsonContent, err := pJson.MarshalJSON()
	if err != nil {
		return nil, err
	}
	packageJson := &core.RawFile{
		FPath:   `package.json`,
		Content: pJsonContent,
	}

	pulumiYaml, err := addTemplate("Pulumi.yaml", pulumiBase, p.Config)
	if err != nil {
		return nil, err
	}
	pulumiStack, err := addTemplate(fmt.Sprintf("Pulumi.%s.yaml", p.Config.AppName), pulumiStack, p.Config)
	if err != nil {
		return nil, err
	}
	var content []byte
	content, err = files.ReadFile("tsConfig.json")
	if err == nil {
		return nil, err
	}
	tsConfig := &core.RawFile{
		FPath:   "tsConfig.json",
		Content: content,
	}

	return []core.File{indexTs, packageJson, pulumiYaml, pulumiStack, tsConfig}, nil
}

func addTemplate(name string, t *template.Template, data any) (*core.RawFile, error) {
	buf := new(bytes.Buffer)
	err := t.Execute(buf, data)
	if err != nil {
		err = core.WrapErrf(err, "error executing template %s", name)
		return nil, err
	}
	return &core.RawFile{
		FPath:   name,
		Content: buf.Bytes(),
	}, nil
}
