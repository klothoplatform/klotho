package cli

import (
	"fmt"
	"github.com/klothoplatform/klotho/pkg/compiler"
	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/engine/constraints"
	"github.com/klothoplatform/klotho/pkg/ioutil"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base"
	"go.uber.org/zap"
	"io"
	"path/filepath"
	"sort"
	"strings"
)

func newTestCase(startingResource core.Resource, targetResource core.Resource, plugins *PluginSetBuilder) *testCase {
	defer recoverPrintFailedPath(startingResource, targetResource)
	graph := core.NewConstructGraph()
	graph.AddConstruct(startingResource)
	graph.AddConstruct(targetResource)
	graph.AddDependency(startingResource.Id(), targetResource.Id())
	document := &compiler.CompilationDocument{
		FileDependencies: &core.FileDependencies{},
		Constructs:       graph,
	}
	klothoCompiler := compiler.Compiler{
		AnalysisAndTransformationPlugins: []compiler.AnalysisAndTransformationPlugin{},
		IaCPlugins:                       []compiler.IaCPlugin{},
		Engine:                           plugins.Engine,
		Document:                         document,
	}
	klothoCompiler.Engine.Context.InitialState = document.Constructs
	c := map[constraints.ConstraintScope][]constraints.Constraint{}
	klothoCompiler.Engine.LoadContext(document.Constructs, c, "infracopilot")
	dag, err := klothoCompiler.Engine.Run()
	if err != nil {
		zap.L().Sugar().Errorf("Failed to run engine: %s", err)
		return nil
	}

	paths := klothoCompiler.Engine.KnowledgeBase.FindPaths(startingResource, targetResource, knowledgebase.EdgeConstraint{})
	sort.Slice(paths, func(i, j int) bool {
		return len(paths[i]) < len(paths[j])
	})

	if len(paths) == 0 {
		zap.L().Sugar().Debugf("%s:%s -> %s:%s has no paths", startingResource.Id().Provider, startingResource.Id().Type, targetResource.Id().Provider, targetResource.Id().Type)
		return nil
	}
	if len(paths[0]) > getPathsConfig.maxPathLength {
		zap.L().Sugar().Debugf("%s:%s -> %s:%s is too long", startingResource.Id().Provider, startingResource.Id().Type, targetResource.Id().Provider, targetResource.Id().Type)
		return nil
	}

	zap.S().Debugf("Finished running engine")
	files, err := klothoCompiler.Engine.VisualizeViews()
	if err != nil {
		zap.L().Sugar().Errorf("Failed to visualize views: %s", err)
		return nil
	}
	document.OutputFiles = append(document.OutputFiles, files...)
	document.Resources = dag

	name := fmt.Sprintf("%s:%s-%s:%s",
		startingResource.Id().Provider,
		startingResource.Id().Type,
		targetResource.Id().Provider,
		targetResource.Id().Type,
	)

	err = klothoCompiler.Document.OutputGraph(filepath.Join(getPathsConfig.outDir, name))
	if err != nil {
		zap.L().Sugar().Errorf("Failed to output graph: %s", err)
		return nil
	}

	doc := []string{fmt.Sprintf("create,%s,RESOURCE,%s,%s_%s",
		startingResource.Id().Type,
		startingResource.Id().Name,
		strings.ToUpper(startingResource.Id().Provider),
		strings.ToUpper(startingResource.Id().Type)),

		fmt.Sprintf("create,%s,RESOURCE,%s,%s_%s",
			targetResource.Id().Type,
			targetResource.Id().Name,
			strings.ToUpper(targetResource.Id().Provider),
			strings.ToUpper(targetResource.Id().Type)),
		fmt.Sprintf("connect,%s_%s/%s,%s_%s/%s",
			strings.ToUpper(startingResource.Id().Provider),
			strings.ToUpper(startingResource.Id().Type),
			startingResource.Id().Name,
			strings.ToUpper(targetResource.Id().Provider),
			strings.ToUpper(targetResource.Id().Type),
			targetResource.Id().Name,
		),
	}
	f := &TextFile{path: fmt.Sprintf("%s.csv", name), Body: strings.Join(doc, "\n")}
	document.OutputFiles = append(document.OutputFiles, f)

	return &testCase{
		Name:     name,
		Source:   startingResource,
		Target:   targetResource,
		Document: document,
		Length:   len(paths[0]),
	}
}

type TextFile struct {
	path string
	Body string
}

func (f *TextFile) Path() string {
	return f.path
}

func (f *TextFile) Clone() core.File {
	return f
}

func (f *TextFile) WriteTo(w io.Writer) (n int64, err error) {
	wh := ioutil.NewWriteToHelper(w, &n, &err)
	wh.Writef(f.Body)
	return n, err
}

func recoverPrintFailedPath(source core.Resource, target core.Resource) {
	if r := recover(); r != nil {
		zap.L().Sugar().Errorf("Failed to find path from %s to %s due to panic:\n%s", source.Id(), target.Id(), r)
	}
}
