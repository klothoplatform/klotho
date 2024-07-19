package template

import (
	"fmt"
	"github.com/klothoplatform/klotho/pkg/k2/model"
	"github.com/klothoplatform/klotho/pkg/reflectutil"
	"reflect"
	"text/template"
)

type (
	ResourceRef struct {
		ConstructURN model.URN
		ResourceKey  string
		Property     string
		Type         ResourceRefType
	}

	ResourceRefType        string
	InterpolationSourceKey string

	InterpolationSource interface {
		GetPropertySource() *PropertySource
	}

	PropertySource struct {
		source reflect.Value
	}

	TemplateFuncSupplier interface {
		GetTemplateFuncs() template.FuncMap
	}
)

const (
	// ResourceRefTypeTemplate is a reference to a resource template and will be fully resolved prior to constraint generation
	// e.g., ${resources:resourceName.property} or ${resources:resourceName}
	ResourceRefTypeTemplate ResourceRefType = "template"
	// ResourceRefTypeIaC is a reference to an infrastructure as code resource that will be resolved by the engine
	// e.g., ${resources:resourceName#property}
	ResourceRefTypeIaC ResourceRefType = "iac"
	// ResourceRefTypeInterpolated is an initial interpolation reference to a resource.
	// An interpolated value will be evaluated during initial processing and will be converted to one of the other types.
	ResourceRefTypeInterpolated ResourceRefType = "interpolated"
)

func (r *ResourceRef) String() string {
	if r.Type == ResourceRefTypeIaC {
		return fmt.Sprintf("%s#%s", r.ResourceKey, r.Property)
	}
	return r.ResourceKey
}

func NewPropertySource(source any) *PropertySource {
	var v reflect.Value
	if sv, ok := source.(reflect.Value); ok {
		v = sv
	} else {
		v = reflect.ValueOf(source)
	}
	return &PropertySource{
		source: v,
	}
}

func (p *PropertySource) GetProperty(key string) (value any, ok bool) {
	v, err := reflectutil.GetField(p.source, key)
	if err != nil || !v.IsValid() {
		return nil, false
	}
	return v.Interface(), true
}

func GetTypedProperty[T any](source *PropertySource, key string) (T, bool) {
	var typedField T
	v, ok := source.GetProperty(key)

	if !ok {
		return typedField, false
	}

	return reflectutil.GetTypedValue[T](v)
}
