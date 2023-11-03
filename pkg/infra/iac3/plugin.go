package iac3

import (
	"bufio"
	"bytes"
	"embed"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"strings"
	"text/template"

	"github.com/klothoplatform/klotho/pkg/config"
	construct "github.com/klothoplatform/klotho/pkg/construct2"
	"github.com/klothoplatform/klotho/pkg/engine2/solution_context"
	kio "github.com/klothoplatform/klotho/pkg/io"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base2"
	"github.com/klothoplatform/klotho/pkg/lang/javascript"
	"github.com/klothoplatform/klotho/pkg/templateutils"
)

type Plugin struct {
	Config *config.Application
	KB     *knowledgebase.KnowledgeBase
}

func (p Plugin) Name() string {
	return "pulumi3"
}

var (
	//go:embed Pulumi.yaml.tmpl Pulumi.dev.yaml.tmpl templates/globals.ts
	files embed.FS

	//go:embed templates/*/*/factory.ts templates/*/*/package.json templates/*/*/*.ts.tmpl
	standardTemplates embed.FS

	pulumiBase  = templateutils.MustTemplate(files, "Pulumi.yaml.tmpl")
	pulumiStack = templateutils.MustTemplate(files, "Pulumi.dev.yaml.tmpl")
)

func (p Plugin) Translate(ctx solution_context.SolutionContext) ([]kio.File, error) {

	// TODO We'll eventually want to split the output into different files, but we don't know exactly what that looks
	// like yet. For now, just write to a single file, "index.ts"
	buf := getBuffer()
	defer releaseBuffer(buf)

	templatesFS, err := fs.Sub(standardTemplates, "templates")
	if err != nil {
		return nil, err
	}

	tc := &TemplatesCompiler{
		graph:     ctx.DeploymentGraph(),
		templates: &templateStore{fs: templatesFS},
	}
	tc.vars, err = VariablesFromGraph(tc.graph)
	if err != nil {
		return nil, err
	}

	if err := tc.RenderImports(buf); err != nil {
		return nil, err
	}
	buf.WriteString("\n\n")

	if err := renderGlobals(buf); err != nil {
		return nil, err
	}

	resources, err := construct.ReverseTopologicalSort(tc.graph)
	if err != nil {
		return nil, err
	}

	var errs error
	for _, r := range resources {
		errs = errors.Join(errs, tc.RenderResource(buf, r))
		buf.WriteString("\n")
	}
	if errs != nil {
		return nil, errs
	}

	indexTs := &kio.RawFile{
		FPath:   `index.ts`,
		Content: buf.Bytes(),
	}

	pJson, err := tc.PackageJSON()
	if err != nil {
		return nil, err
	}
	packageJson := &javascript.PackageFile{
		FPath:   "package.json",
		Content: pJson,
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
	content, err = files.ReadFile("tsconfig.json")
	if err == nil {
		return nil, err
	}
	tsConfig := &kio.RawFile{
		FPath:   "tsconfig.json",
		Content: content,
	}

	files := []kio.File{indexTs, packageJson, pulumiYaml, pulumiStack, tsConfig}

	dockerfiles, err := RenderDockerfiles(ctx)
	if err != nil {
		return nil, err
	}

	files = append(files, dockerfiles...)

	return files, nil
}

func renderGlobals(w io.Writer) error {
	globalsFile, err := files.Open("templates/globals.ts")
	if err != nil {
		return err
	}
	defer globalsFile.Close()

	scan := bufio.NewScanner(globalsFile)
	for scan.Scan() {
		text := strings.TrimSpace(scan.Text())
		if text == "" {
			continue
		}
		if strings.HasPrefix(text, "import") {
			continue
		}
		_, err := fmt.Fprintln(w, text)
		if err != nil {
			return err
		}
	}
	_, err = fmt.Fprintln(w)
	return err
}

func addTemplate(name string, t *template.Template, data any) (*kio.RawFile, error) {
	buf := new(bytes.Buffer) // Don't use the buffer pool since RawFile uses the byte array

	err := t.Execute(buf, data)
	if err != nil {
		return nil, fmt.Errorf("error executing template %s: %w", name, err)
	}
	return &kio.RawFile{
		FPath:   name,
		Content: buf.Bytes(),
	}, nil
}
