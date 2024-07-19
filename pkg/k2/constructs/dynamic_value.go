package constructs

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"slices"
	"strings"
	"text/template"

	"github.com/klothoplatform/klotho/pkg/async"
	"github.com/klothoplatform/klotho/pkg/construct"
	template2 "github.com/klothoplatform/klotho/pkg/k2/constructs/template"
	"github.com/klothoplatform/klotho/pkg/k2/constructs/template/properties"
	"github.com/klothoplatform/klotho/pkg/k2/model"
	"github.com/klothoplatform/klotho/pkg/templateutils"
	"go.uber.org/zap"
)

type DynamicValueContext struct {
	constructs *async.ConcurrentMap[model.URN, *Construct]
}

type DynamicValueData struct {
	currentOwner      InfraOwner
	currentSelection  DynamicValueSelection
	propertySource    *template2.PropertySource
	resourceKeyPrefix string
}

func (ctx DynamicValueContext) TemplateFunctions() template.FuncMap {
	return templateutils.WithCommonFuncs(template.FuncMap{
		"fieldRef":           ctx.FieldRef,
		"pathAncestor":       ctx.PathAncestor,
		"pathAncestorExists": ctx.PathAncestorExists,
		"toJSON":             ctx.toJson,
	})
}

func (ctx DynamicValueContext) Parse(tmpl string) (*template.Template, error) {
	t, err := template.New("config").Funcs(ctx.TemplateFunctions()).Parse(tmpl)
	return t, err
}

func (ctx DynamicValueContext) ExecuteUnmarshal(tmpl string, data any, value any) error {
	t, err := ctx.Parse(tmpl)
	if err != nil {
		return err
	}
	return ctx.ExecuteTemplateUnmarshal(t, data, value)
}

func (ctx DynamicValueContext) Unmarshal(data *bytes.Buffer, v any) error {
	return properties.UnmarshalAny(data, v)
}

// ExecuteTemplateUnmarshal executes the template tmpl using data as input and unmarshals the value into v
func (ctx DynamicValueContext) ExecuteTemplateUnmarshal(
	t *template.Template,
	data any,
	v any,
) error {
	buf := new(bytes.Buffer)
	if err := t.Execute(buf, data); err != nil {
		return err
	}

	if err := ctx.Unmarshal(buf, v); err != nil {
		return fmt.Errorf("cannot decode template result '%s' into %T", buf, v)
	}

	return nil
}

// Self returns the owner of this dynamic value
func (data *DynamicValueData) Self() any {
	return data.currentOwner
}

// Selected returns the current selection in the dynamic value data
func (data *DynamicValueData) Selected() DynamicValueSelection {
	return data.currentSelection
}

func (data *DynamicValueData) Select(path string) bool {
	var ps *template2.PropertySource
	if data.currentSelection.Value != nil {
		ps = template2.NewPropertySource(data.currentSelection.Value)
	} else {
		ps = data.propertySource
		if ps == nil {
			ps = data.currentOwner.GetPropertySource()
		}
	}

	if v, ok := ps.GetProperty(path); ok {
		s := SelectItem(v)
		data.currentSelection = s
		return true
	}
	return false
}

// Inputs returns the inputs of the current owner
func (data *DynamicValueData) Inputs() any {
	ps := data.propertySource
	if ps == nil {
		ps = data.currentOwner.GetPropertySource()
	}
	val, _ := ps.GetProperty("inputs")
	return val
}

// Resources returns the resources of the current owner
func (data *DynamicValueData) Resources() any {
	ps := data.propertySource
	if ps == nil {
		ps = data.currentOwner.GetPropertySource()
	}
	val, _ := ps.GetProperty("resources")
	return val
}

// Edges returns the edges of the current owner
func (data *DynamicValueData) Edges() any {
	ps := data.propertySource
	if ps == nil {
		ps = data.currentOwner.GetPropertySource()
	}
	val, _ := ps.GetProperty("edges")
	return val
}

// Meta returns the metadata of the current owner
func (data *DynamicValueData) Meta() any {
	ps := data.propertySource
	if ps == nil {
		ps = data.currentOwner.GetPropertySource()
	}
	val, _ := ps.GetProperty("meta")
	return val
}

func (data *DynamicValueData) Prefix() string {
	return data.resourceKeyPrefix
}

// From returns the 'from' construct if the current owner is a binding
func (data *DynamicValueData) From() any {
	ps := data.propertySource
	if ps == nil {
		ps = data.currentOwner.GetPropertySource()
	}
	val, _ := ps.GetProperty("from")
	return val
}

// To returns the 'to' construct if the current owner is a binding
func (data *DynamicValueData) To() any {
	ps := data.propertySource
	if ps == nil {
		ps = data.currentOwner.GetPropertySource()
	}
	val, _ := ps.GetProperty("to")
	return val
}

// Log is primarily used for debugging templates and only be used in development to log messages to the console
func (data *DynamicValueData) Log(level string, message string, args ...interface{}) string {
	l := zap.L()

	ownerType := reflect.TypeOf(data.currentOwner).Kind().String()
	ownerString := "unknown"

	l = l.With(zap.String(ownerType, ownerString))

	switch strings.ToLower(level) {
	case "debug":
		l.Sugar().Debugf(message, args...)
	case "info":
		l.Sugar().Infof(message, args...)
	case "warn":
		l.Sugar().Warnf(message, args...)
	case "error":
		l.Sugar().Errorf(message, args...)
	default:
		l.Sugar().Warnf(message, args...)
	}
	return ""
}

// toJson is used to return complex values that do not have TextUnmarshaler implemented
func (ctx DynamicValueContext) toJson(value any) (string, error) {
	j, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	return string(j), nil
}

func (ctx DynamicValueContext) PathAncestor(path construct.PropertyPath, depth int) (string, error) {
	if depth < 0 {
		return "", fmt.Errorf("depth must be >= 0")
	}
	if depth == 0 {
		return path.String(), nil
	}
	if len(path) <= depth {
		return "", fmt.Errorf("depth %d is greater than path length %d", depth, len(path))
	}
	return path[:len(path)-depth].String(), nil
}

func (ctx DynamicValueContext) PathAncestorExists(path construct.PropertyPath, depth int) bool {
	return len(path) > depth
}

// FieldRef returns a reference to `field` on `resource` (as a PropertyRef)
func (ctx DynamicValueContext) FieldRef(field string, resource any) (construct.PropertyRef, error) {
	resId, err := TemplateArgToRID(resource)
	if err != nil {
		return construct.PropertyRef{}, err
	}

	return construct.PropertyRef{
		Resource: resId,
		Property: field,
	}, nil
}

func TemplateArgToRID(arg any) (construct.ResourceId, error) {
	switch arg := arg.(type) {
	case construct.ResourceId:
		return arg, nil

	case construct.Resource:
		return arg.ID, nil

	case string:
		var resId construct.ResourceId
		err := resId.UnmarshalText([]byte(arg))
		return resId, err
	}

	return construct.ResourceId{}, fmt.Errorf("invalid argument type %T", arg)
}

type DynamicValueSelection struct {
	Source  any
	mapKeys []reflect.Value
	next    int
	Key     string
	Value   any
	Index   int
}

func SelectItem(src any) DynamicValueSelection {
	srcValue := reflect.ValueOf(src)
	switch srcValue.Kind() {
	case reflect.Map:
		if !srcValue.IsValid() || srcValue.Len() == 0 {
			return DynamicValueSelection{
				Source: src,
			}
		}
		keys := srcValue.MapKeys()
		slices.SortStableFunc(keys, func(i, j reflect.Value) int {
			return strings.Compare(stringValue(i.Interface()), stringValue(j.Interface()))
		})
		if len(keys) == 0 {
			return DynamicValueSelection{
				Source: src,
			}
		}
		return DynamicValueSelection{
			Source:  src,
			mapKeys: keys,
		}
	default:
		return DynamicValueSelection{
			Source: src,
		}
	}
}

// Next returns the next value in the selection and whether there are more values
// If the selection is a map, the key is also returned.
// If the selection is a slice, the index is returned instead.
// If there are no more values, the second return value is false.
//
// This function is intended to be used by an orchestration layer across multiple go templates
// and is unavailable inside the templates themselves
func (s *DynamicValueSelection) Next() (any, bool) {
	srcValue := reflect.ValueOf(s.Source)

	if !srcValue.IsValid() {
		return nil, false
	}

	if len(s.mapKeys) > 0 {
		if s.next >= len(s.mapKeys) {
			return nil, false
		}
		key := s.mapKeys[s.next]
		value := srcValue.MapIndex(key).Interface()
		s.Value = value
		s.next++
		return value, true
	}
	if s.Index >= srcValue.Len() {
		return nil, false
	}
	value := srcValue.Index(s.Index).Interface()
	s.Value = value
	s.Index++
	return value, true
}

func stringValue(v any) string {
	return fmt.Sprintf("%v", v)
}
