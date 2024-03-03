package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"

	"github.com/dominikbraun/graph"
	"github.com/klothoplatform/klotho/pkg/dot"
	"github.com/klothoplatform/klotho/pkg/graph_addons"
	"github.com/klothoplatform/klotho/pkg/knowledgebase"
)

func dotAttributes(tmpl *knowledgebase.ResourceTemplate, props graph.VertexProperties) map[string]string {
	a := make(map[string]string)
	for k, v := range props.Attributes {
		if k != "rank" {
			a[k] = v
		}
	}
	a["label"] = tmpl.QualifiedTypeName
	a["shape"] = "box"
	return a
}

func dotEdgeAttributes(e *knowledgebase.EdgeTemplate, props graph.EdgeProperties) map[string]string {
	a := make(map[string]string)
	for k, v := range props.Attributes {
		a[k] = v
	}
	if e.DeploymentOrderReversed {
		a["style"] = "dashed"
	}
	a["edgetooltip"] = fmt.Sprintf("%s -> %s", e.Source, e.Target)
	return a
}

func KbToDot(g graph.Graph[string, *knowledgebase.ResourceTemplate], out io.Writer) error {
	ids, err := graph_addons.TopologicalSort(g, func(a, b string) bool {
		return a < b
	})
	if err != nil {
		return err
	}
	var errs error
	printf := func(s string, args ...any) {
		_, err := fmt.Fprintf(out, s, args...)
		errs = errors.Join(errs, err)
	}
	printf(`digraph {
  rankdir = TB
`)
	for _, id := range ids {
		t, props, err := g.VertexWithProperties(id)
		if err != nil {
			errs = errors.Join(errs, err)
			continue
		}
		if rank, ok := props.Attributes["rank"]; ok {
			printf("  { rank = %s; %q%s; }\n", rank, id, dot.AttributesToString(dotAttributes(t, props)))
		} else {
			printf("  %q%s;\n", t.QualifiedTypeName, dot.AttributesToString(dotAttributes(t, props)))
		}
	}

	topoIndex := func(id string) int {
		for i, id2 := range ids {
			if id2 == id {
				return i
			}
		}
		return -1
	}
	edges, err := g.Edges()
	if err != nil {
		return err
	}
	sort.Slice(edges, func(i, j int) bool {
		ti, tj := topoIndex(edges[i].Source), topoIndex(edges[j].Source)
		if ti != tj {
			return ti < tj
		}
		ti, tj = topoIndex(edges[i].Target), topoIndex(edges[j].Target)
		return ti < tj
	})
	for _, e := range edges {
		et, ok := e.Properties.Data.(*knowledgebase.EdgeTemplate)
		if !ok {
			errs = errors.Join(errs, fmt.Errorf("edge %q -> %q has no EdgeTemplate", e.Source, e.Target))
			continue
		}
		printf("  %q -> %q%s\n", e.Source, e.Target, dot.AttributesToString(dotEdgeAttributes(et, e.Properties)))
	}
	printf("}\n")
	return errs
}

func KbToSVG(kb knowledgebase.TemplateKB, prefix string) error {
	if debugDir := os.Getenv("KLOTHO_DEBUG_DIR"); debugDir != "" {
		prefix = filepath.Join(debugDir, prefix)
	}
	f, err := os.Create(prefix + ".gv")
	if err != nil {
		return err
	}
	defer f.Close()

	hasGraph, ok := kb.(interface {
		Graph() graph.Graph[string, *knowledgebase.ResourceTemplate]
	})
	if !ok {
		return fmt.Errorf("knowledgebase does not have a graph")
	}
	g := hasGraph.Graph()

	dotContent := new(bytes.Buffer)
	err = KbToDot(g, io.MultiWriter(f, dotContent))
	if err != nil {
		return fmt.Errorf("could not render graph to file %s: %v", prefix+".gv", err)
	}

	svgContent, err := dot.ExecPan(bytes.NewReader(dotContent.Bytes()))
	if err != nil {
		return fmt.Errorf("could not run 'dot' for %s: %v", prefix+".gv", err)
	}

	svgFile, err := os.Create(prefix + ".gv.svg")
	if err != nil {
		return fmt.Errorf("could not create file %s: %v", prefix+".gv.svg", err)
	}
	defer svgFile.Close()
	_, err = fmt.Fprint(svgFile, svgContent)
	return err
}
