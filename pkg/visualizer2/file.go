package visualizer

import (
	"fmt"
	"io"
	"reflect"
	"strings"

	construct "github.com/klothoplatform/klotho/pkg/construct2"
	klotho_io "github.com/klothoplatform/klotho/pkg/io"
	"github.com/klothoplatform/klotho/pkg/ioutil"
	"gopkg.in/yaml.v3"
)

const indent = "    "

type (
	File struct {
		FilenamePrefix string
		AppName        string
		Provider       string
		DAG            construct.Graph
	}

	// FetchPropertiesFunc is a function that takes a resource of some type, and returns some properties for it.
	// This function also takes in the DAG, since some resources need that context to figure out their properties (for
	// example, a subnet is private or public depending on how it's used).
	FetchPropertiesFunc[K construct.Resource] func(res K, dag construct.Graph) map[string]any

	// propertiesFetcher takes a [construct.Resource] and returns some properties for it. This is similar to
	// FetchPropertiesFunc, except that the argument is always a construct.Resource (as opposed to a specific subtype
	// of construct.Resource, as with FetchPropertiesFunc).
	propertiesFetcher interface {
		apply(res *construct.Resource, dag construct.Graph) map[string]any
	}

	// typedPropertiesFetcher is a propertiesFetcher that can also tell you which type it'll accept. The (unenforced)
	// expectation is that if you pass in an element "e" to apply(...) such that reflect.TypeOf(e) != tpf.reflectType(),
	// then the resulting map will always be nil.
	//
	// We use this as a convenience bridge: in the absence of wildcard generics in Go (e.g., "FetchPropertiesFunc[*]"),
	// we can treat a "FetchPropertiesFunc[K]" as a typedPropertiesFetcher, and build a list of heterogeneous fetchers.
	// Then, we can iterate through that list to build a [byTypePropertiesFetcher] that maps each FetchPropertiesFunc
	// by its type.
	typedPropertiesFetcher interface {
		propertiesFetcher
		reflectType() reflect.Type
	}

	byTypePropertiesFetcher map[reflect.Type]propertiesFetcher
)

func (f *File) Path() string {
	return fmt.Sprintf("%stopology.yaml", f.FilenamePrefix)
}

func (f *File) Clone() klotho_io.File {
	return f
}

func (f *File) WriteTo(w io.Writer) (n int64, err error) {
	wh := ioutil.NewWriteToHelper(w, &n, &err)

	wh.Writef("%s:\n", f.AppName)
	wh.Writef("  provider: %s\n", f.Provider)
	wh.Write("  resources:\n")

	resourceIds, err := construct.ReverseTopologicalSort(f.DAG)
	if err != nil {
		return
	}
	resources := make([]*construct.Resource, len(resourceIds))
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
			if _, ok := properties[key]; !ok {
				id := construct.ResourceId{}
				err := id.UnmarshalText([]byte(val))
				if err != nil {
					return n, err
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
	if resId.Provider != f.Provider {
		providerInfo = resId.Provider + `:`
	}
	return strings.ToLower(fmt.Sprintf("%s%s/%s", providerInfo, res.ID.Type, resId.Name))
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
