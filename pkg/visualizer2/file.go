package visualizer

import (
	"fmt"
	"io"
	"strings"

	construct "github.com/klothoplatform/klotho/pkg/construct2"
	klotho_io "github.com/klothoplatform/klotho/pkg/io"
	"github.com/klothoplatform/klotho/pkg/ioutil"
	"gopkg.in/yaml.v3"
)

const indent = "  "

type (
	File struct {
		FilenamePrefix string
		AppName        string
		Provider       string
		DAG            construct.Graph
	}
)

func (f *File) Path() string {
	return fmt.Sprintf("%stopology.yaml", f.FilenamePrefix)
}

func (f *File) Clone() klotho_io.File {
	return f
}

func (f *File) WriteTo(w io.Writer) (n int64, err error) {
	wh := ioutil.NewWriteToHelper(w, &n, &err)

	wh.Writef("provider: %s\n", f.Provider)
	wh.Write("resources:\n")

	resourceIds, err := construct.ReverseTopologicalSort(f.DAG)
	if err != nil {
		return
	}
	var resources []*construct.Resource
	for _, id := range resourceIds {
		res, err := f.DAG.Vertex(id)
		if err != nil {
			return n, err
		}
		resources = append(resources, res)
	}
	for _, resource := range resources {
		key := f.KeyFor(resource)
		if key == "" {
			continue
		}
		wh.Writef(indent+"%s:\n", key)

		// Add any edge properties as metadata
		_, props, err := f.DAG.VertexWithProperties(resource.ID)
		if err != nil {
			return n, err
		}
		properties := props.Attributes

		for key, val := range properties {
			if _, ok := properties[key]; ok {
				id := construct.ResourceId{}
				err := id.UnmarshalText([]byte(val))
				if err != nil {
					properties[key] = val
					continue
				}
				valRes, err := f.DAG.Vertex(id)
				if err != nil {
					return n, err
				}
				if valRes != nil {
					properties[key] = f.KeyFor(valRes)
				} else {
					properties[key] = val
				}
			}
		}

		if len(properties) > 0 {
			writeYaml(properties, 2, wh)
		}
		// Need to write edge properties here tomorrow
		deps, err := construct.DirectDownstreamDependencies(f.DAG, resource.ID)
		if err != nil {
			return n, err
		}
		downstreamResources := make([]*construct.Resource, len(deps))
		for i, dep := range deps {
			downstreamResources[i], err = f.DAG.Vertex(dep)
			if err != nil {
				return n, err
			}
		}
		for _, res := range downstreamResources {
			src := f.KeyFor(resource)
			dst := f.KeyFor(res)
			if src != "" && dst != "" {
				wh.Writef(indent+"%s -> %s:\n", src, dst)
			}
			dep, err := f.DAG.Edge(resource.ID, res.ID)
			if err != nil {
				return n, err
			}
			if dep.Properties.Data != nil {
				writeYaml(dep.Properties.Data, 2, wh)
			}
		}
		wh.Write("\n")
	}

	return
}

func (f *File) KeyFor(res *construct.Resource) string {
	resId := res.ID
	var providerInfo string
	var namespaceInfo string
	if resId.Provider != f.Provider {
		providerInfo = resId.Provider + `:`
	}
	if resId.Namespace != "" {
		namespaceInfo = ":" + resId.Namespace
	}
	return strings.ToLower(fmt.Sprintf("%s%s%s/%s", providerInfo, res.ID.Type, namespaceInfo, resId.Name))
}

func writeYaml(e any, indentCount int, out ioutil.WriteToHelper) {
	bs, err := yaml.Marshal(e)
	if err != nil {
		out.AddErr(err)
		return
	}
	for _, line := range strings.Split(string(bs), "\n") {
		if strings.TrimSpace(line) != "" {
			for i := 0; i < indentCount; i++ {
				out.Write(indent)
			}
		}
		out.Write(line)
		out.Write("\n")
	}
}
