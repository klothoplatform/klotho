package visualizer

import (
	"fmt"
	"io"
	"sort"
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
		Graph          VisGraph
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

	resourceIds, err := construct.ReverseTopologicalSort(f.Graph)
	if err != nil {
		return
	}
	adj, err := f.Graph.AdjacencyMap()
	if err != nil {
		return
	}
	for _, id := range resourceIds {
		res, err := f.Graph.Vertex(id)
		if err != nil {
			return n, err
		}
		src := f.KeyFor(id)
		if src == "" {
			continue
		}
		wh.Writef(indent+"%s:\n", src)

		props := map[string]any{
			"tag": res.Tag,
		}
		if !res.Parent.IsZero() {
			props["parent"] = f.KeyFor(res.Parent)
		}
		if len(res.Children) > 0 {
			childrenIds := res.Children.ToSlice()
			sort.Sort(construct.SortedIds(childrenIds))
			children := make([]string, len(childrenIds))
			for i, child := range childrenIds {
				children[i] = child.String()
			}
			props["children"] = children
		}

		if len(props) > 0 {
			writeYaml(props, 2, wh)
		} else {
			wh.Write("\n")
		}

		deps := adj[id]
		downstream := make([]construct.ResourceId, 0, len(deps))
		for dep := range deps {
			downstream = append(downstream, dep)
		}
		sort.Sort(construct.SortedIds(downstream))
		for _, dep := range downstream {
			dst := f.KeyFor(dep)
			if src != "" && dst != "" {
				wh.Writef(indent+"%s -> %s:\n", src, dst)
			}
			dep, err := f.Graph.Edge(id, dep)
			if err != nil {
				return n, err
			}
			if dep.Properties.Data != nil {
				writeYaml(dep.Properties.Data, 2, wh)
			}
		}
	}

	return
}

func (f *File) KeyFor(res construct.ResourceId) string {
	resId := res
	var providerInfo string
	var namespaceInfo string
	if resId.Provider != f.Provider || resId.Namespace != "" {
		providerInfo = resId.Provider + `:`
	}
	if resId.Namespace != "" {
		namespaceInfo = ":" + resId.Namespace
	}
	return strings.ToLower(fmt.Sprintf("%s%s%s/%s", providerInfo, res.Type, namespaceInfo, resId.Name))
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
