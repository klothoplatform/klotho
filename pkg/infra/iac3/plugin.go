package iac3

import (
	"bufio"
	"bytes"
	"embed"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"regexp"
	"strings"
	"text/template"

	construct "github.com/klothoplatform/klotho/pkg/construct2"
	"github.com/klothoplatform/klotho/pkg/engine2/solution_context"
	kio "github.com/klothoplatform/klotho/pkg/io"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base2"
	"github.com/klothoplatform/klotho/pkg/templateutils"
)

type (
	PulumiConfig struct {
		AppName string
	}

	Plugin struct {
		Config *PulumiConfig
		KB     *knowledgebase.KnowledgeBase
	}
)

func (p Plugin) Name() string {
	return "pulumi3"
}

var (
	//go:embed Pulumi.yaml.tmpl Pulumi.dev.yaml.tmpl templates/globals.ts templates/tsconfig.json
	files embed.FS

	//go:embed templates/aws/*/factory.ts templates/aws/*/package.json templates/aws/*/*.ts.tmpl
	//go:embed templates/kubernetes/*/factory.ts templates/kubernetes/*/package.json templates/kubernetes/*/*.ts.tmpl
	standardTemplates embed.FS

	pulumiBase  = templateutils.MustTemplate(files, "Pulumi.yaml.tmpl")
	pulumiStack = templateutils.MustTemplate(files, "Pulumi.dev.yaml.tmpl")
)

func (p Plugin) Translate(ctx solution_context.SolutionContext) ([]kio.File, error) {

	err := p.sanitizeConfig()
	if err != nil {
		return nil, err
	}
	// TODO We'll eventually want to split the output into different files, but we don't know exactly what that looks
	// like yet. For now, just write to a single file, "index.ts"
	buf := getBuffer()
	defer releaseBuffer(buf)

	templatesFS, err := fs.Sub(standardTemplates, "templates")
	if err != nil {
		return nil, err
	}
	err = addPulumiKubernetesProviders(ctx.DeploymentGraph())
	if err != nil {
		return nil, fmt.Errorf("error adding pulumi kubernetes providers: %w", err)
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

	pulumiYaml, err := addTemplate("Pulumi.yaml", pulumiBase, p.Config)
	if err != nil {
		return nil, err
	}
	pulumiStack, err := addTemplate(fmt.Sprintf("Pulumi.%s.yaml", p.Config.AppName), pulumiStack, p.Config)
	if err != nil {
		return nil, err
	}
	var content []byte
	content, err = files.ReadFile("templates/tsconfig.json")
	if err != nil {
		return nil, err
	}
	tsConfig := &kio.RawFile{
		FPath:   "tsconfig.json",
		Content: content,
	}

	files := []kio.File{indexTs, pJson, pulumiYaml, pulumiStack, tsConfig}

	dockerfiles, err := RenderDockerfiles(ctx)
	if err != nil {
		return nil, err
	}

	files = append(files, dockerfiles...)

	return files, nil
}

func (p *Plugin) sanitizeConfig() error {
	reg, err := regexp.Compile("[^a-zA-Z0-9-_]+")
	if err != nil {
		return fmt.Errorf("error compiling regex: %v", err)
	}
	p.Config.AppName = reg.ReplaceAllString(p.Config.AppName, "")
	return nil
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
		text = strings.TrimPrefix(text, "export ")
		_, err := fmt.Fprintln(w, text)
		if err != nil {
			return err
		}
	}
	_, err = fmt.Fprintln(w)
	return err
}

func addPulumiKubernetesProviders(g construct.Graph) error {
	providers := make(map[construct.ResourceId]*construct.Resource)
	kubeconfigId := construct.ResourceId{Provider: "kubernetes", Type: "kube_config"}
	err := construct.WalkGraph(g, func(id construct.ResourceId, resource *construct.Resource, nerr error) error {
		if !kubeconfigId.Matches(id) {
			return nerr
		}
		provider := &construct.Resource{
			ID: construct.ResourceId{
				Provider: "kubernetes",
				Type:     "kubernetes_provider",
				Name:     id.Name,
			},
			Properties: construct.Properties{
				"KubeConfig": id,
			},
		}
		err := g.AddVertex(provider)
		if err != nil {
			return errors.Join(nerr, err)
		}
		err = g.AddEdge(provider.ID, id)
		if err != nil {
			return errors.Join(nerr, err)
		}
		providers[id] = provider

		return nerr
	})
	if err != nil {
		return err
	}

	err = construct.WalkGraph(g, func(id construct.ResourceId, resource *construct.Resource, nerr error) error {
		if id.Provider != "kubernetes" {
			return nerr
		}
		cluster, err := resource.GetProperty("Cluster")
		if err != nil {
			return errors.Join(nerr, err)
		}
		if cluster == nil {
			return nerr
		}
		clusterId, ok := cluster.(construct.ResourceId)
		if !ok {
			return errors.Join(nerr, fmt.Errorf("resource %s is a kubernetes resource but does not have an id as cluster property (is: %T)", id, cluster))
		}
		upstreams, err := construct.DirectUpstreamDependencies(g, clusterId)
		if err != nil {
			return errors.Join(nerr, err)
		}
		var kubeconfig construct.ResourceId
		for _, upstream := range upstreams {
			if kubeconfigId.Matches(upstream) {
				kubeconfig = upstream
				break
			}
		}
		provider, ok := providers[kubeconfig]
		if !ok {
			return errors.Join(nerr, fmt.Errorf("resource %s is a kubernetes resource but does not have a provider resource for cluster %s", id, clusterId))
		}
		err = resource.SetProperty("Provider", provider.ID)
		if err != nil {
			return errors.Join(nerr, err)
		}
		err = g.AddEdge(id, provider.ID)
		if err != nil {
			return errors.Join(nerr, err)
		}
		return nerr
	})
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
