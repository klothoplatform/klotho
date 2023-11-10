package path_selection

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/dominikbraun/graph"
	construct "github.com/klothoplatform/klotho/pkg/construct2"
	"github.com/klothoplatform/klotho/pkg/dot"
)

const (
	notChosenColour = "#e87b7b"
	// yellow = "#e3cf9d"
	choenColour = "#3f822b"
)

func attributes(result construct.Graph, id construct.ResourceId, props graph.VertexProperties) map[string]string {
	a := make(map[string]string)

	if strings.HasPrefix(id.Name, PHANTOM_PREFIX) {
		a["style"] = "dashed"
		a["label"] = id.QualifiedTypeName()
		a["shape"] = "ellipse"
	} else {
		a["label"] = id.String()
		a["shape"] = "box"
	}
	if name := props.Attributes["new_name"]; name != "" {
		// vertex is a renamed phantom
		a["style"] = "dashed"
		a["label"] = fmt.Sprintf(`%s\n:%s`, id.QualifiedTypeName(), name)
	}
	if _, err := result.Vertex(id); err == nil {
		a["color"] = choenColour
	} else {
		a["color"] = notChosenColour
	}
	return a
}

func graphToDOTCluster(class string, working, result construct.Graph, out io.Writer) error {
	var errs error
	printf := func(s string, args ...any) {
		_, err := fmt.Fprintf(out, s, args...)
		errs = errors.Join(errs, err)
	}
	label := class
	if class == "" {
		class = "default"
		label = "<default>"
	}

	printf(`  subgraph cluster_%s {
    label = %q
`, class, label)
	adj, err := working.AdjacencyMap()
	if err != nil {
		return err
	}

	fixId := func(id construct.ResourceId) construct.ResourceId {
		_, tProps, _ := working.VertexWithProperties(id)
		if tProps.Attributes != nil {
			if name := tProps.Attributes["new_name"]; name != "" {
				id.Name = name
			}
		}
		return id
	}

	for src, a := range adj {
		_, props, _ := working.VertexWithProperties(src)
		src = fixId(src)
		attribs := attributes(result, src, props)
		prefixedSrc := fmt.Sprintf("%s/%s", class, src)
		printf("    %q%s\n", prefixedSrc, dot.AttributesToString(attribs))

		for tgt, e := range a {
			tgt = fixId(tgt)
			prefixedTgt := fmt.Sprintf("%s/%s", class, tgt)

			edgeAttribs := make(map[string]string)
			if _, err := result.Edge(src, tgt); err == nil {
				edgeAttribs["color"] = choenColour
				edgeAttribs["weight"] = "1000"
				edgeAttribs["penwidth"] = "2"
			} else {
				edgeAttribs["style"] = "dashed"
			}
			edgeAttribs["label"] = fmt.Sprintf("%d", e.Properties.Weight)

			printf("    %q -> %q%s\n", prefixedSrc, prefixedTgt, dot.AttributesToString(edgeAttribs))
		}
	}
	printf("  }\n")
	return errs
}
