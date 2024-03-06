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
	"github.com/klothoplatform/klotho/pkg/knowledgebase/properties"
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

func dotEdgeAttributes(
	kb knowledgebase.TemplateKB,
	e *knowledgebase.EdgeTemplate,
	props graph.EdgeProperties,
) map[string]string {
	a := make(map[string]string)
	for k, v := range props.Attributes {
		a[k] = v
	}
	if e.DeploymentOrderReversed {
		a["style"] = "dashed"
	}
	a["edgetooltip"] = fmt.Sprintf("%s -> %s", e.Source, e.Target)

	if source, err := kb.GetResourceTemplate(e.Source); err == nil {
		var isTarget func(ps knowledgebase.Properties) knowledgebase.Property
		isTarget = func(ps knowledgebase.Properties) knowledgebase.Property {
			for _, p := range ps {
				name := p.Details().Name
				if name == "" {
					fmt.Print()
				}
				switch inst := p.(type) {
				case *properties.ResourceProperty:
					if inst.AllowedTypes.MatchesAny(e.Target) {
						return p
					}

				case knowledgebase.CollectionProperty:
					if ip := inst.Item(); ip != nil {
						ret := isTarget(knowledgebase.Properties{"item": ip})
						if ret != nil {
							return ret
						}
					}

				case knowledgebase.MapProperty:
					mapProps := make(knowledgebase.Properties)
					if kp := inst.Key(); kp != nil {
						mapProps["key"] = kp
					}
					if vp := inst.Value(); vp != nil {
						mapProps["value"] = vp
					}
					ret := isTarget(mapProps)
					if ret != nil {
						return ret
					}
				}
				return isTarget(p.SubProperties())
			}
			return nil
		}
		prop := isTarget(source.Properties)
		if prop != nil {
			if label, ok := a["label"]; ok {
				a["label"] = label + "\n" + prop.Details().Path
			} else {
				a["label"] = prop.Details().Path
			}
		}
	}
	return a
}

func KbToDot(kb knowledgebase.TemplateKB, out io.Writer) error {
	hasGraph, ok := kb.(interface {
		Graph() graph.Graph[string, *knowledgebase.ResourceTemplate]
	})
	if !ok {
		return fmt.Errorf("knowledgebase does not have a graph")
	}
	g := hasGraph.Graph()

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
		printf("  %q -> %q%s\n", e.Source, e.Target, dot.AttributesToString(dotEdgeAttributes(kb, et, e.Properties)))
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

	dotContent := new(bytes.Buffer)
	err = KbToDot(kb, io.MultiWriter(f, dotContent))
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
