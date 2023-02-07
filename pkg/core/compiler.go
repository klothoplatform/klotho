package core

import (
	"fmt"
	"reflect"

	"go.uber.org/zap"
)

type (
	CompilationResult ConcurrentMap[ResourceKey, CloudResource]

	ResourceKey struct {
		// Kind is the kind of resource it is such as `exec_unit`, `persist_kv`, etc
		Kind string
		// Name must be unique to the compilation for the `Kind` of resource.
		Name string
	}

	CloudResource interface {
		// Key returns the unique identifier for the resource.
		Key() ResourceKey
	}

	HasLocalOutput interface {
		OutputTo(dest string) error
	}

	Plugin interface {
		Name() string

		// Transform is expected to mutate the result and any dependencies
		Transform(*CompilationResult, *Dependencies) error
	}

	Compiler struct {
		Plugins []Plugin
	}

	// ResourcesOrErr provided as commonly used in async operations for the result channel.
	ResourcesOrErr struct {
		Resources []CloudResource
		Err       error
	}
)

func (k ResourceKey) String() string {
	return fmt.Sprintf("%s:%s", k.Kind, k.Name)
}

func (result *CompilationResult) Get(key ResourceKey) CloudResource {
	m := (*ConcurrentMap[ResourceKey, CloudResource])(result)
	r, _ := m.Get(key)
	return r
}

func (result *CompilationResult) GetResourcesOfType(rType string) (filtered []CloudResource) {
	m := (*ConcurrentMap[ResourceKey, CloudResource])(result)
	m.Each(func(k ResourceKey, v CloudResource) (stop bool) {
		if k.Kind == rType {
			filtered = append(filtered, v)
		}
		return false
	})
	return
}

func GetResourcesOfType[T CloudResource](result *CompilationResult) (filtered []T) {
	m := (*ConcurrentMap[ResourceKey, CloudResource])(result)
	m.Each(func(k ResourceKey, v CloudResource) (stop bool) {
		if vT, ok := v.(T); ok {
			filtered = append(filtered, vT)
		}
		return false
	})
	return
}

// GetFirstResource returns the first resource found of type `rType` or nil if not found.
func (result *CompilationResult) GetFirstResource(rType string) (res CloudResource) {
	m := (*ConcurrentMap[ResourceKey, CloudResource])(result)
	m.Each(func(k ResourceKey, v CloudResource) (stop bool) {
		if k.Kind == rType {
			res = v
			return true
		}
		return false
	})
	return
}

func GetFirstResource[T CloudResource](result *CompilationResult) (res T) {
	m := (*ConcurrentMap[ResourceKey, CloudResource])(result)
	m.Each(func(k ResourceKey, v CloudResource) (stop bool) {
		if vT, ok := v.(T); ok {
			res = vT
			return true
		}
		return false
	})
	return
}

func (result *CompilationResult) Add(resource CloudResource) {
	m := (*ConcurrentMap[ResourceKey, CloudResource])(result)
	m.Set(resource.Key(), resource)
	zap.S().Infof("Adding resource %s", resource.Key())
}

func (result *CompilationResult) Keys() []ResourceKey {
	m := (*ConcurrentMap[ResourceKey, CloudResource])(result)
	return m.Keys()
}

func (result *CompilationResult) Resources() []CloudResource {
	m := (*ConcurrentMap[ResourceKey, CloudResource])(result)
	return m.Values()
}

func (result *CompilationResult) AddAll(ress []CloudResource) {
	for _, res := range ress {
		result.Add(res)
	}
}

func (result *CompilationResult) Len() int {
	m := (*ConcurrentMap[ResourceKey, CloudResource])(result)
	return m.Len()
}

func (result *CompilationResult) GetExecUnitForPath(fp string) (*ExecutionUnit, File) {
	var best *ExecutionUnit
	var bestFile File
	for _, eu := range GetResourcesOfType[*ExecutionUnit](result) {
		f := eu.Get(fp)
		if f != nil {
			astF, ok := f.(*SourceFile)
			if ok && (best == nil || FileExecUnitName(astF) == eu.Name) {
				best = eu
				bestFile = f
			}
		}
	}
	return best, bestFile
}

func (c *Compiler) Compile(main *InputFiles) (*CompilationResult, error) {
	result := &CompilationResult{}
	// Do the initial add without using Add so it doesn't log anything
	(*ConcurrentMap[ResourceKey, CloudResource])(result).Set(main.Key(), main)

	deps := &Dependencies{}

	for _, p := range c.Plugins {
		if isPluginNil(p) {
			continue
		}

		log := zap.L().With(zap.String("plugin", p.Name()))

		log.Debug("starting")
		err := p.Transform(result, deps)
		if err != nil {
			return result, NewPluginError(p.Name(), err)
		}
		log.Debug("completed")
	}

	return result, nil
}

func isPluginNil(i Plugin) bool {
	if i == nil {
		return true
	}
	switch reflect.TypeOf(i).Kind() {
	case reflect.Pointer:
		return reflect.ValueOf(i).IsNil()
	}
	return false
}
