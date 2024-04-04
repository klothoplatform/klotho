package main

import (
	"errors"
	"fmt"

	"github.com/alecthomas/kong"
	"github.com/dominikbraun/graph"
	"github.com/klothoplatform/klotho/pkg/construct"
	"github.com/klothoplatform/klotho/pkg/engine"
	"github.com/klothoplatform/klotho/pkg/engine/path_selection"
	"github.com/klothoplatform/klotho/pkg/graph_addons"
	"github.com/klothoplatform/klotho/pkg/knowledgebase"
	"github.com/klothoplatform/klotho/pkg/knowledgebase/reader"
	"github.com/klothoplatform/klotho/pkg/logging"
	"github.com/klothoplatform/klotho/pkg/templates"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type Args struct {
	Verbose        bool   `short:"v" help:"Enable verbose mode"`
	Distance       int    `short:"d" help:"Distance from single type to display" default:"2"`
	Classification string `short:"c" help:"Classification to filter for (like path expansion)"`
	Source         string `arg:"" optional:""`
	Target         string `arg:"" optional:""`
}

func main() {
	var args Args
	ctx := kong.Parse(&args)

	logOpts := logging.LogOpts{
		Verbose:         args.Verbose,
		CategoryLogsDir: "",
		DefaultLevels: map[string]zapcore.Level{
			"lsp":       zap.WarnLevel,
			"lsp/pylsp": zap.WarnLevel,
		},
		Encoding: "pretty_console",
	}

	zap.ReplaceGlobals(logOpts.NewLogger())
	defer zap.L().Sync() //nolint:errcheck

	if err := args.Run(ctx); err != nil {
		panic(err)
	}
}

func (args Args) Run(ctx *kong.Context) error {
	kb, err := reader.NewKBFromFs(templates.ResourceTemplates, templates.EdgeTemplates, templates.Models)
	if err != nil {
		return err
	}

	switch {
	case args.Source == "" && args.Target == "":
		break
	case args.Target == "":
		if args.Classification != "" {
			return fmt.Errorf("classification can only be used with two types (for now)")
		}
		kb = args.filterSingleKb(kb)
	default:
		if args.Classification != "" {
			var edge construct.SimpleEdge
			if err := edge.Source.UnmarshalText([]byte(args.Source)); err != nil {
				return fmt.Errorf("could not parse source: %w", err)
			}
			edge.Source.Name = "source"
			if err := edge.Target.UnmarshalText([]byte(args.Target)); err != nil {
				return fmt.Errorf("could not parse target: %w", err)
			}
			edge.Target.Name = "target"
			g, err := path_selection.BuildPathSelectionGraph(edge, kb, args.Classification, true)
			if err != nil {
				return err
			}
			return engine.GraphToSVG(kb, g, "kb_path_selection")
		}
		kb = args.filterPathKB(kb)
	}

	return KbToSVG(kb, "knowledgebase")
}

func (args Args) filterPathKB(kb *knowledgebase.KnowledgeBase) *knowledgebase.KnowledgeBase {
	var source, target construct.ResourceId
	if err := source.UnmarshalText([]byte(args.Source)); err != nil {
		panic(fmt.Errorf("could not parse source: %w", err))
	}
	if err := target.UnmarshalText([]byte(args.Target)); err != nil {
		panic(fmt.Errorf("could not parse target: %w", err))
	}

	paths, err := kb.AllPaths(source, target)
	if err != nil {
		panic(err)
	}
	shortestPath, err := graph.ShortestPath(kb.Graph(), args.Source, args.Target)
	if err != nil {
		panic(err)
	}

	filteredKb := knowledgebase.NewKB()
	g := filteredKb.Graph()
	addV := func(t *knowledgebase.ResourceTemplate) (err error) {
		if t.QualifiedTypeName == args.Source || t.QualifiedTypeName == args.Target {
			attribs := map[string]string{
				"color":    "green",
				"penwidth": "2",
			}
			if t.QualifiedTypeName == args.Source {
				attribs["rank"] = "source"
			} else {
				attribs["rank"] = "sink"
			}
			err = g.AddVertex(t, graph.VertexAttributes(attribs))
		} else {
			err = g.AddVertex(t)
		}
		if errors.Is(err, graph.ErrVertexAlreadyExists) {
			return nil
		}
		return err
	}
	addE := func(path []*knowledgebase.ResourceTemplate, t1, t2 *knowledgebase.ResourceTemplate) error {
		edge, err := kb.Graph().Edge(t1.QualifiedTypeName, t2.QualifiedTypeName)
		if err != nil {
			return err
		}
		err = g.AddEdge(t1.QualifiedTypeName, t2.QualifiedTypeName, func(ep *graph.EdgeProperties) {
			*ep = edge.Properties
			if len(path) == len(shortestPath) {
				ep.Attributes["color"] = "green"
				ep.Attributes["penwidth"] = "2"
			}
		})
		if errors.Is(err, graph.ErrEdgeAlreadyExists) {
			return nil
		}
		return err
	}
	var errs error
	for _, path := range paths {
		if len(path) > len(shortestPath)*2 {
			continue
		}
		errs = errors.Join(errs, addV(path[0]))
		for i, t := range path[1:] {
			errs = errors.Join(
				errs,
				addV(t),
				addE(path, path[i], t),
			)
		}
	}
	return filteredKb
}

func (args Args) filterSingleKb(kb *knowledgebase.KnowledgeBase) *knowledgebase.KnowledgeBase {
	filteredKb := knowledgebase.NewKB()
	g := filteredKb.Graph()

	r, props, err := kb.Graph().VertexWithProperties(args.Source)
	if err != nil {
		panic(err)
	}
	err = g.AddVertex(r, func(vp *graph.VertexProperties) {
		*vp = props
		vp.Attributes["color"] = "green"
		vp.Attributes["penwidth"] = "2"
	})
	if err != nil {
		panic(err)
	}

	addV := func(s string) (err error) {
		t, err := kb.Graph().Vertex(s)
		if err != nil {
			return err
		}
		err = g.AddVertex(t)
		if errors.Is(err, graph.ErrVertexAlreadyExists) {
			return nil
		}
		return err
	}
	walkFunc := func(up bool) func(p graph_addons.Path[string], nerr error) error {
		edge := func(a, b string) (graph.Edge[*knowledgebase.ResourceTemplate], error) {
			if up {
				a, b = b, a
			}
			return kb.Graph().Edge(a, b)
		}

		return func(p graph_addons.Path[string], nerr error) error {
			last := p[len(p)-1]
			if err := addV(last); err != nil {
				return err
			}
			edge, err := edge(p[len(p)-2], last)
			if err != nil {
				return err
			}
			err = g.AddEdge(p[len(p)-2], last, func(ep *graph.EdgeProperties) {
				*ep = edge.Properties
			})
			if err != nil && !errors.Is(err, graph.ErrEdgeAlreadyExists) {
				return err
			}
			if len(p) >= args.Distance {
				return graph_addons.SkipPath
			}
			return nil
		}
	}

	err = errors.Join(
		graph_addons.WalkUp(kb.Graph(), args.Source, walkFunc(true)),
		graph_addons.WalkDown(kb.Graph(), args.Source, walkFunc(false)),
	)
	if err != nil {
		panic(err)
	}

	return filteredKb
}
