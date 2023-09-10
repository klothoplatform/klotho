package knowledgebase

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"text/template"

	"github.com/klothoplatform/klotho/pkg/construct"
)

func (ctx *ConfigurationContext) Funcs() template.FuncMap {
	return template.FuncMap{
		// DAG operations
		"self":       ctx.Self,
		"upstream":   ctx.Upstream,
		"downstream": ctx.Downstream,

		// Value manipulations
		"split":                ctx.Split,
		"filterMatch":          ctx.FilterMatch,
		"mapString":            ctx.MapString,
		"zipToMap":             ctx.ZipToMap,
		"keysToMapWithDefault": ctx.KeysToMapWithDefault,
	}
}

func (ctx *ConfigurationContext) Self() string {
	return ctx.resource.Id().String()
}

// Upstream returns the first resource that matches `selector` which is upstream of the resource that is being configured.
func (ctx *ConfigurationContext) Upstream(selector string) (string, error) {
	var selId construct.ResourceId
	if err := selId.UnmarshalText([]byte(selector)); err != nil {
		return "", err
	}
	if selId.Matches(ctx.resource.Id()) {
		return ctx.resource.Id().String(), nil
	}

	upstream := ctx.dag.GetAllUpstreamResources(ctx.resource)
	for _, up := range upstream {
		if selId.Matches(up.Id()) {
			return up.Id().String(), nil
		}
	}
	return "", fmt.Errorf("no upstream resource found matching selector '%s'", selector)
}

// Downstream returns the first resource that matches `selector` which is downstream of the resource that is being configured.
func (ctx *ConfigurationContext) Downstream(selector string) (string, error) {
	var selId construct.ResourceId
	if err := selId.UnmarshalText([]byte(selector)); err != nil {
		return "", err
	}
	if selId.Matches(ctx.resource.Id()) {
		return ctx.resource.Id().String(), nil
	}

	downstream := ctx.dag.GetAllDownstreamResources(ctx.resource)
	for _, down := range downstream {
		if selId.Matches(down.Id()) {
			return down.Id().String(), nil
		}
	}
	return "", fmt.Errorf("no downstream resource found matching selector '%s'", selector)
}

func toJson(value any) (string, error) {
	j, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	return string(j), nil
}

func (ctx *ConfigurationContext) Split(delim, value string) (string, error) {
	if value == "" {
		return "[]", nil
	}
	parts := strings.Split(value, delim)
	return toJson(parts)
}

func (ctx *ConfigurationContext) FilterMatch(pattern, value string) (string, error) {
	if value == "" {
		return "[]", nil
	}

	var values []string
	if err := json.Unmarshal([]byte(value), &values); err != nil {
		return "", err
	}

	re, err := regexp.Compile(pattern)
	if err != nil {
		return "", err
	}

	var matches []string
	for _, v := range values {
		if ok := re.MatchString(v); ok {
			matches = append(matches, v)
		}
	}
	return toJson(matches)
}

func (ctx *ConfigurationContext) MapString(pattern, replace, value string) (string, error) {
	if value == "" {
		return "[]", nil
	}

	var values []string
	if err := json.Unmarshal([]byte(value), &values); err != nil {
		return "", err
	}

	re, err := regexp.Compile(pattern)
	if err != nil {
		return "", err
	}

	for i, v := range values {
		values[i] = re.ReplaceAllString(v, replace)
	}
	return toJson(values)
}

func (ctx *ConfigurationContext) ZipToMap(keys, values string) (string, error) {
	if keys == "" && values == "" {
		return "{}", nil
	}

	var keyValues []string
	if err := json.Unmarshal([]byte(keys), &keyValues); err != nil {
		return "", err
	}
	var valueValues []any
	if err := json.Unmarshal([]byte(values), &valueValues); err != nil {
		return "", err
	}

	if len(keyValues) != len(valueValues) {
		return "", fmt.Errorf("key length (%d) != value length (%d)", len(keyValues), len(valueValues))
	}

	m := make(map[string]any)
	for i, k := range keyValues {
		m[k] = valueValues[i]
	}
	return toJson(m)
}

func (ctx *ConfigurationContext) KeysToMapWithDefault(defaultValueJson string, keysJson string) (string, error) {
	if keysJson == "" {
		return "{}", nil
	}

	var keys []string
	if err := json.Unmarshal([]byte(keysJson), &keys); err != nil {
		return "", err
	}

	var defaultValue any
	if err := json.Unmarshal([]byte(defaultValueJson), &defaultValue); err != nil {
		return "", err
	}

	m := make(map[string]any)
	for _, k := range keys {
		m[k] = defaultValue
	}
	return toJson(m)
}
