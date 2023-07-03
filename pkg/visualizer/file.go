package visualizer

import (
	"fmt"
	"io"
	"reflect"
	"strings"

	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/ioutil"
	"gopkg.in/yaml.v3"
)

const indent = "    "

type (
	File struct {
		FilenamePrefix string
		AppName        string
		Provider       string
		DAG            *core.ResourceGraph
	}

	// FetchPropertiesFunc is a function that takes a resource of some type, and returns some properties for it.
	// This function also takes in the DAG, since some resources need that context to figure out their properties (for
	// example, a subnet is private or public depending on how it's used).
	FetchPropertiesFunc[K core.Resource] func(res K, dag *core.ResourceGraph) map[string]any

	// propertiesFetcher takes a [core.Resource] and returns some properties for it. This is similar to
	// FetchPropertiesFunc, except that the argument is always a core.Resource (as opposed to a specific subtype
	// of core.Resource, as with FetchPropertiesFunc).
	propertiesFetcher interface {
		apply(res core.Resource, dag *core.ResourceGraph) map[string]any
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

func (f *File) Clone() core.File {
	return f
}

func (f *File) WriteTo(w io.Writer) (n int64, err error) {
	propFetcher := defaultPropertiesFetchers()
	wh := ioutil.NewWriteToHelper(w, &n, &err)

	wh.Writef("%s:\n", f.AppName)
	wh.Writef("  provider: %s\n", f.Provider)
	wh.Write("  resources:\n")

	resources, err := f.DAG.ReverseTopologicalSort()
	if err != nil {
		return
	}
	for _, resource := range resources {
		if resource.Id().Provider == core.InternalProvider {
			// Don't show internal resources such as imported in the topology
			// TODO maybe make some way of indicating imported resources in the visualizer
			continue
		}
		key := f.KeyFor(resource)
		if key == "" {
			continue
		}
		wh.Writef(indent+"%s:\n", key)
		properties := propFetcher.apply(resource, f.DAG)

		// Add any edge properties as metadata
		_, props := f.DAG.GetResourceWithProperties(resource.Id())
		if properties == nil {
			properties = make(map[string]any)
		}
		for key, val := range props {
			if _, ok := properties[key]; !ok {
				valRes := f.DAG.GetResourceFromString(val)
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

		edges := f.DAG.GetDownstreamDependencies(resource)
		for _, edge := range edges {
			src := f.KeyFor(edge.Source)
			dst := f.KeyFor(edge.Destination)
			if src != "" && dst != "" {
				wh.Writef(indent+"%s -> %s:\n", src, dst)
			}
		}
		wh.Write("\n")
	}

	return
}

func (f *File) KeyFor(res core.Resource) string {
	resId := res.Id()
	var providerInfo string
	if resId.Provider != f.Provider {
		providerInfo = resId.Provider + `:`
	}
	return strings.ToLower(fmt.Sprintf("%s%s/%s", providerInfo, TypeFor(res, f.DAG), resId.Name))
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

func defaultPropertiesFetchers() byTypePropertiesFetcher {
	var all []typedPropertiesFetcher
	// BEGIN Add your property fetchers here
	all = append(all, asApplier(subnetProperties))
	// END

	result := make(map[reflect.Type]propertiesFetcher, len(all))
	for _, pf := range all {
		result[pf.reflectType()] = pf
	}

	return result
}

func asApplier[K core.Resource](f FetchPropertiesFunc[K]) typedPropertiesFetcher {
	return f
}

func (f FetchPropertiesFunc[K]) apply(res core.Resource, dag *core.ResourceGraph) map[string]any {
	if res, ok := res.(K); ok {
		return f(res, dag)
	}
	return nil
}

func (pf byTypePropertiesFetcher) apply(res core.Resource, dag *core.ResourceGraph) map[string]any {
	resType := reflect.TypeOf(res)
	if f := pf[resType]; f != nil {
		return f.apply(res, dag)
	}
	return nil
}

func (c FetchPropertiesFunc[K]) reflectType() reflect.Type {
	return reflect.TypeOf((*K)(nil)).Elem()
}
