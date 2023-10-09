package knowledgebase2

import (
	"bytes"
	"encoding"
	"encoding/json"
	"fmt"
	"path/filepath"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"text/template"

	"github.com/dominikbraun/graph"
	construct "github.com/klothoplatform/klotho/pkg/construct2"
	"go.uber.org/zap"
)

type (
	// DynamicValueContext is used to scope the DAG into the template functions
	DynamicValueContext struct {
		DAG        construct.Graph
		KB         TemplateKB
		resultJson bool
	}

	// DynamicValueData provides the resource or edge to the templates as
	// `{{ .Self }}` for resources
	// `{{ .Source }}` and `{{ .Destination }}` for edges
	DynamicValueData struct {
		Resource construct.ResourceId
		Edge     *construct.Edge
	}
)

func (ctx DynamicValueContext) TemplateFunctions() template.FuncMap {
	return template.FuncMap{
		"hasUpstream":   ctx.HasUpstream,
		"upstream":      ctx.Upstream,
		"allUpstream":   ctx.AllUpstream,
		"hasDownstream": ctx.HasDownstream,
		"downstream":    ctx.Downstream,
		"allDownstream": ctx.AllDownstream,
		"shortestPath":  ctx.ShortestPath,
		"longestPath":   ctx.LongestPath,
		"fieldValue":    ctx.FieldValue,
		"hasField":      ctx.HasField,
		"fieldRef":      ctx.FieldRef,

		"toJson": ctx.toJson,

		"split":    strings.Split,
		"join":     strings.Join,
		"basename": filepath.Base,

		"firstId":              firstId,
		"filterIds":            filterIds,
		"filterMatch":          filterMatch,
		"mapString":            mapString,
		"zipToMap":             zipToMap,
		"keysToMapWithDefault": keysToMapWithDefault,
		"replace":              replaceRegex,

		"add": add,
		"sub": sub,
	}
}

func (ctx DynamicValueContext) Parse(tmpl string) (*template.Template, error) {
	t, err := template.New("config").Funcs(ctx.TemplateFunctions()).Parse(tmpl)
	return t, err
}

func (ctx DynamicValueContext) ExecuteDecodeAsResourceId(tmpl string, data DynamicValueData) (construct.ResourceId, error) {
	var selector construct.ResourceId
	err := ctx.ExecuteDecode(tmpl, data, &selector)
	if err != nil {
		return selector, err
	}
	if selector.IsZero() {
		// ? Should this error instead?
		// Make sure we don't just add arbitrary dependencies, since all resources match the zero value
		return selector, fmt.Errorf("selector '%s' is zero", tmpl)
	}
	return selector, nil
}

// ExecuteDecode executes the template `tmpl` using `data` and decodes the value into `value`
func (ctx DynamicValueContext) ExecuteDecode(tmpl string, data DynamicValueData, value interface{}) error {
	t, err := ctx.Parse(tmpl)
	if err != nil {
		return err
	}
	buf := new(bytes.Buffer)
	if err := t.Execute(buf, data); err != nil {
		return err
	}

	if ctx.resultJson {
		dec := json.NewDecoder(buf)
		return dec.Decode(value)
	}

	// trim the spaces so you don't have to sprinkle the templates with `{{-` and `-}}` (the `-` trims spaces)
	bstr := strings.TrimSpace(buf.String())

	switch value := value.(type) {
	case *string:
		*value = bstr
		return nil

	case *[]byte:
		*value = []byte(bstr)
		return nil

	case *bool:
		b, err := strconv.ParseBool(bstr)
		if err != nil {
			return err
		}
		*value = b
		return nil
	case *int:
		i, err := strconv.Atoi(bstr)
		if err != nil {
			return err
		}
		*value = i
		return nil
	case *float64:
		f, err := strconv.ParseFloat(bstr, 64)
		if err != nil {
			return err
		}
		*value = f
		return nil
	case *float32:
		f, err := strconv.ParseFloat(bstr, 32)
		if err != nil {
			return err
		}
		*value = float32(f)
		return nil

	case encoding.TextUnmarshaler:
		// notably, this handles `construct.ResourceId` and `construct.IaCValue`
		return value.UnmarshalText([]byte(bstr))
	}

	resultStr := reflect.ValueOf(buf.String())
	valueRefl := reflect.ValueOf(value).Elem()
	if resultStr.Type().AssignableTo(valueRefl.Type()) {
		// this covers alias types like `type MyString string`
		valueRefl.Set(resultStr)
		return nil
	}

	err = json.Unmarshal([]byte(bstr), value)
	if err == nil {
		return nil
	}

	return fmt.Errorf("cannot decode template result '%s' into %T", buf, value)
}

func (ctx DynamicValueContext) ResolveConfig(config Configuration, data DynamicValueData) (Configuration, error) {
	if cfgVal, ok := config.Value.(string); ok {
		res, err := ctx.DAG.Vertex(data.Resource)
		if err != nil {
			return config, err
		}

		field := reflect.ValueOf(res).Elem().FieldByName(config.Field)
		if !field.IsValid() {
			return config, fmt.Errorf("field %s not found on resource %s", config.Field, data.Resource)
		}

		valueRefl := reflect.New(field.Type())
		value := valueRefl.Interface()
		err = ctx.ExecuteDecode(cfgVal, data, value)
		if err != nil {
			return config, err
		}

		config.Value = valueRefl.Elem().Interface()
	}
	return config, nil
}

func (data DynamicValueData) Self() (construct.ResourceId, error) {
	if data.Resource.IsZero() {
		return construct.ResourceId{}, fmt.Errorf("no .Self is set")
	}
	return data.Resource, nil
}

func (data DynamicValueData) Source() (construct.ResourceId, error) {
	if data.Edge.Source.IsZero() {
		return construct.ResourceId{}, fmt.Errorf("no .Source is set")
	}
	return data.Edge.Source, nil
}

func (data DynamicValueData) Destination() (construct.ResourceId, error) {
	if data.Edge.Target.IsZero() {
		return construct.ResourceId{}, fmt.Errorf("no .Destination is set")
	}
	return data.Edge.Target, nil
}

// Log is primarily used for debugging templates and shouldn't actually appear in any.
// Allows for outputting any intermediate values (such as `$integration := downstream "aws:api_integration" .Self`)
func (data DynamicValueData) Log(level string, message string, args ...interface{}) string {
	l := zap.L()
	if !data.Resource.IsZero() {
		l = l.With(zap.String("resource", data.Resource.String()))
	}
	if !data.Edge.Source.IsZero() {
		l = l.With(zap.String("edge", data.Edge.Source.String()+" -> "+data.Edge.Target.String()))
	}
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

func (ctx DynamicValueContext) upstream(selector any, resource construct.ResourceId) (construct.ResourceId, error) {
	selId, err := TemplateArgToRID(selector)
	if err != nil {
		return construct.ResourceId{}, err
	}

	upstream, err := Upstream(ctx.DAG, ctx.KB, resource, FirstFunctionalLayer)
	if err != nil {
		return construct.ResourceId{}, err
	}
	for _, up := range upstream {
		if selId.Matches(up) {
			return up, nil
		}
	}
	return construct.ResourceId{}, nil
}

// Upstream returns the first resource that matches `selector` which is upstream of `resource`
func (ctx DynamicValueContext) HasUpstream(selector any, resource construct.ResourceId) (bool, error) {
	up, err := ctx.upstream(selector, resource)
	if err != nil {
		return false, err
	}
	return !up.IsZero(), nil
}

// Upstream returns the first resource that matches `selector` which is upstream of `resource`
func (ctx DynamicValueContext) Upstream(selector any, resource construct.ResourceId) (construct.ResourceId, error) {
	up, err := ctx.upstream(selector, resource)
	if err != nil {
		return construct.ResourceId{}, err
	}
	if up.IsZero() {
		return up, fmt.Errorf("no upstream resource of '%s' found matching selector '%s'", resource, selector)
	}
	return up, nil
}

// AllUpstream is like Upstream but returns all transitive upstream resources.
// nolint: lll
func (ctx DynamicValueContext) AllUpstream(selector any, resource construct.ResourceId) (construct.ResourceList, error) {
	selId, err := TemplateArgToRID(selector)
	if err != nil {
		return nil, err
	}
	upstreams, err := Upstream(ctx.DAG, ctx.KB, resource, AllDepsLayer)
	if err != nil {
		return []construct.ResourceId{}, err
	}
	matches := make([]construct.ResourceId, 0, len(upstreams))
	for _, up := range upstreams {
		if selId.Matches(up) {
			matches = append(matches, up)
		}
	}
	return matches, nil
}

func (ctx DynamicValueContext) downstream(selector any, resource construct.ResourceId) (construct.ResourceId, error) {
	selId, err := TemplateArgToRID(selector)
	if err != nil {
		return construct.ResourceId{}, err
	}

	downstream, err := Downstream(ctx.DAG, ctx.KB, resource, FirstFunctionalLayer)
	if err != nil {
		return construct.ResourceId{}, err
	}
	for _, down := range downstream {
		if selId.Matches(down) {
			return down, nil
		}
	}
	return construct.ResourceId{}, nil
}

// Downstream returns the first resource that matches `selector` which is downstream of `resource`
// nolint: lll
func (ctx DynamicValueContext) HasDownstream(selector any, resource construct.ResourceId) (bool, error) {
	down, err := ctx.downstream(selector, resource)
	if err != nil {
		return false, err
	}
	return !down.IsZero(), nil
}

// Downstream returns the first resource that matches `selector` which is downstream of `resource`
// nolint: lll
func (ctx DynamicValueContext) Downstream(selector any, resource construct.ResourceId) (construct.ResourceId, error) {
	down, err := ctx.downstream(selector, resource)
	if err != nil {
		return construct.ResourceId{}, err
	}
	if down.IsZero() {
		return down, fmt.Errorf("no downstream resource of '%s' found matching selector '%s'", resource, selector)
	}
	return down, nil
}

// AllDownstream is like Downstream but returns all transitive downstream resources.
// nolint: lll
func (ctx DynamicValueContext) AllDownstream(selector any, resource construct.ResourceId) (construct.ResourceList, error) {
	selId, err := TemplateArgToRID(selector)
	if err != nil {
		return nil, err
	}
	downstreams, err := Downstream(ctx.DAG, ctx.KB, resource, AllDepsLayer)
	if err != nil {
		return []construct.ResourceId{}, err
	}
	matches := make([]construct.ResourceId, 0, len(downstreams))
	for _, down := range downstreams {
		if selId.Matches(down) {
			matches = append(matches, down)
		}
	}
	return matches, nil
}

// ShortestPath returns all the resource IDs on the shortest path from source to destination
func (ctx DynamicValueContext) ShortestPath(source, destination any) (construct.ResourceList, error) {
	srcId, err := TemplateArgToRID(source)
	if err != nil {
		return nil, err
	}
	dstId, err := TemplateArgToRID(destination)
	if err != nil {
		return nil, err
	}
	return graph.ShortestPath(ctx.DAG, srcId, dstId)
}

// LongestPath returns all the resource IDs on the longest path from source to destination
func (ctx DynamicValueContext) LongestPath(source, destination any) ([]construct.ResourceId, error) {
	srcId, err := TemplateArgToRID(source)
	if err != nil {
		return nil, err
	}
	dstId, err := TemplateArgToRID(destination)
	if err != nil {
		return nil, err
	}
	paths, err := graph.AllPathsBetween(ctx.DAG, srcId, dstId)
	if err != nil {
		return nil, err
	}
	var longest []construct.ResourceId
	for _, path := range paths {
		if len(path) > len(longest) {
			longest = path
		}
	}
	return longest, nil
}

// FieldValue returns the value of `field` on `resource` in json
func (ctx DynamicValueContext) FieldValue(field string, resource any) (any, error) {
	resId, err := TemplateArgToRID(resource)
	if err != nil {
		return "", err
	}

	r, err := ctx.DAG.Vertex(resId)
	if r == nil || err != nil {
		return nil, fmt.Errorf("resource '%s' not found", resId)
	}
	val, err := r.GetProperty(field)
	if err != nil || val == nil {
		return nil, fmt.Errorf("field '%s' not found on resource '%s'", field, resId)
	}
	return val, nil
}

func (ctx DynamicValueContext) HasField(field string, resource any) (bool, error) {
	resId, err := TemplateArgToRID(resource)
	if err != nil {
		return false, err
	}

	r, err := ctx.DAG.Vertex(resId)
	if r == nil || err != nil {
		return false, fmt.Errorf("resource '%s' not found", resId)
	}
	property, err := r.GetProperty(field)
	if err != nil || property == nil {
		return false, nil
	}
	return true, nil
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

// toJson is used to return complex values that do not have TextUnmarshaler implemented
func (ctx DynamicValueContext) toJson(value any) (string, error) {
	j, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	ctx.resultJson = true
	return string(j), nil
}

// filterMatch returns a json array by filtering the values array with the regex pattern
func filterMatch(pattern string, values []string) ([]string, error) {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, err
	}

	matches := make([]string, 0, len(values))
	for _, v := range values {
		if ok := re.MatchString(v); ok {
			matches = append(matches, v)
		}
	}
	return matches, nil
}

func filterIds(selector any, ids []construct.ResourceId) ([]construct.ResourceId, error) {
	selId, err := TemplateArgToRID(selector)
	if err != nil {
		return nil, err
	}
	matches := make([]construct.ResourceId, 0, len(ids))
	for _, r := range ids {
		if selId.Matches(r) {
			matches = append(matches, r)
		}
	}
	return matches, nil
}

func firstId(selector any, ids []construct.ResourceId) (construct.ResourceId, error) {
	selId, err := TemplateArgToRID(selector)
	if err != nil {
		return construct.ResourceId{}, err
	}
	if len(ids) == 0 {
		return construct.ResourceId{}, fmt.Errorf("no ids")
	}
	for _, r := range ids {
		if selId.Matches(r) {
			return r, nil
		}
	}
	return construct.ResourceId{}, fmt.Errorf("no ids match selector")
}

// mapstring takes in a regex pattern and replacement as well as a json array of strings
// roughly `unmarshal value | sed s/pattern/replace/g | marshal`
func mapString(pattern, replace string, values []string) ([]string, error) {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, err
	}

	nv := make([]string, len(values))
	for i, v := range values {
		nv[i] = re.ReplaceAllString(v, replace)
	}
	return nv, nil
}

// zipToMap returns a json map by zipping the keys and values arrays
// Example: zipToMap(['a', 'b'], [1, 2]) => {"a": 1, "b": 2}
func zipToMap(keys []string, valuesArg any) (map[string]any, error) {
	// Have to use reflection here because technically, []string is not assignable to []any
	// thanks Go.
	valuesRefl := reflect.ValueOf(valuesArg)
	if valuesRefl.Kind() != reflect.Slice && valuesRefl.Kind() != reflect.Array {
		return nil, fmt.Errorf("values is not a slice or array")
	}
	if len(keys) != valuesRefl.Len() {
		return nil, fmt.Errorf("key length (%d) != value length (%d)", len(keys), valuesRefl.Len())
	}

	m := make(map[string]any)
	for i, k := range keys {
		m[k] = valuesRefl.Index(i).Interface()
	}
	return m, nil
}

// keysToMapWithDefault returns a json map by mapping the keys array to the static defaultValue
// Example keysToMapWithDefault(0, ['a', 'b']) => {"a": 0, "b": 0}
func keysToMapWithDefault(defaultValue any, keys []string) (map[string]any, error) {
	m := make(map[string]any)
	for _, k := range keys {
		m[k] = defaultValue
	}
	return m, nil
}

func add(args ...int) int {
	total := 0
	for _, a := range args {
		total += a
	}
	return total
}

func sub(args ...int) int {
	if len(args) == 0 {
		return 0
	}
	total := args[0]
	for _, a := range args[1:] {
		total -= a
	}
	return total
}

func replaceRegex(pattern, replace, value string) (string, error) {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return "", err
	}
	s := re.ReplaceAllString(value, replace)
	return s, nil
}
