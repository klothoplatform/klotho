package property_eval

import (
	"errors"
	"fmt"
	"io"
	"text/template"

	construct "github.com/klothoplatform/klotho/pkg/construct2"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base2"
	"github.com/klothoplatform/klotho/pkg/set"
	"go.uber.org/zap"
)

// fauxConfigContext acts like a [knowledgebase.DynamicValueContext] but replaces the [FieldValue] function
// with one that just returns the zero value of the property type and records the property reference.
type fauxConfigContext struct {
	propRef    construct.PropertyRef
	inner      knowledgebase.DynamicValueContext
	refs       set.Set[construct.PropertyRef]
	graphState graphStates
}

func (ctx *fauxConfigContext) DAG() construct.Graph {
	return ctx.inner.DAG()
}

func (ctx *fauxConfigContext) KB() knowledgebase.TemplateKB {
	return ctx.inner.KB()
}

func (ctx *fauxConfigContext) ExecuteDecode(tmpl string, data knowledgebase.DynamicValueData, value interface{}) error {
	t, err := template.New("config").Funcs(ctx.TemplateFunctions()).Parse(tmpl)
	if err != nil {
		return fmt.Errorf("could not parse template: %w", err)
	}
	return ctx.inner.ExecuteTemplateDecode(t, data, value)
}

func (ctx *fauxConfigContext) ExecuteValue(v any, data knowledgebase.DynamicValueData) error {
	_, err := knowledgebase.TransformToPropertyValue(ctx.propRef.Resource, ctx.propRef.Property, v, ctx, data)
	if err != nil {
		zap.S().Debugf("ignoring error from TransformToPropertyValue during deps calculation on %s: %s", ctx.propRef, err)
	}

	return nil
}

func (ctx *fauxConfigContext) Execute(v any, data knowledgebase.DynamicValueData) error {
	if ctx.refs == nil {
		ctx.refs = make(set.Set[construct.PropertyRef])
	}
	if ref, ok := v.(construct.PropertyRef); ok {
		ctx.refs.Add(ref)
		return nil
	}
	vStr, ok := v.(string)
	if !ok {
		return nil
	}
	tmpl, err := template.New(ctx.propRef.String()).Funcs(ctx.TemplateFunctions()).Parse(vStr)
	if err != nil {
		return fmt.Errorf("could not parse template %w", err)
	}

	// Ignore execution errors for when the zero value is invalid due to other assumptions
	// if there is an error with the template, this will be caught later when actually processing it.
	err = tmpl.Execute(
		io.Discard, // we don't care about the results, just the side effect of appending to propCtx.refs
		data,
	)
	if err != nil {
		zap.S().Debugf("ignoring error from template execution during deps calculation: %s", err)
	}
	return nil
}

func (ctx *fauxConfigContext) ExecuteOpRule(
	data knowledgebase.DynamicValueData,
	opRule knowledgebase.OperationalRule,
) error {
	var errs error
	errs = errors.Join(errs, ctx.Execute(opRule.If, data))
	for _, rule := range opRule.ConfigurationRules {
		errs = errors.Join(errs, ctx.Execute(rule.Config.Value, data))
	}
	for _, step := range opRule.Steps {
		for _, stepRes := range step.Resources {
			errs = errors.Join(errs, ctx.Execute(stepRes.Selector, data))
			for _, propValue := range stepRes.Properties {
				errs = errors.Join(errs, ctx.Execute(propValue, data))
			}
		}
	}
	return errs
}

func (ctx *fauxConfigContext) TemplateFunctions() template.FuncMap {
	funcs := ctx.inner.TemplateFunctions()
	funcs["hasField"] = ctx.HasField
	funcs["fieldValue"] = ctx.FieldValue
	funcs["hasUpstream"] = ctx.HasUpstream
	funcs["upstream"] = ctx.Upstream
	funcs["allUpstream"] = ctx.AllUpstream
	funcs["hasDownstream"] = ctx.HasDownstream
	funcs["downstream"] = ctx.Downstream
	funcs["closestDownstream"] = ctx.ClosestDownstream
	funcs["allDownstream"] = ctx.AllDownstream
	return funcs
}

func (ctx *fauxConfigContext) HasField(field string, resource any) (bool, error) {
	resId, err := knowledgebase.TemplateArgToRID(resource)
	if err != nil {
		return false, err
	}
	if resId.IsZero() {
		return false, nil
	}
	if ctx.refs == nil {
		ctx.refs = make(set.Set[construct.PropertyRef])
	}
	ctx.refs.Add(construct.PropertyRef{Resource: resId, Property: field})

	return ctx.inner.HasField(field, resId)
}

func (ctx *fauxConfigContext) FieldValue(field string, resource any) (any, error) {

	resId, err := knowledgebase.TemplateArgToRID(resource)
	if err != nil {
		return "", err
	}
	if resId.IsZero() {
		return nil, nil
	}
	if ctx.refs == nil {
		ctx.refs = make(set.Set[construct.PropertyRef])
	}
	ctx.refs.Add(construct.PropertyRef{Resource: resId, Property: field})

	value, err := ctx.inner.FieldValue(field, resId)
	if err != nil {
		return nil, err
	}
	if value != nil {
		return value, nil
	}

	tmpl, err := ctx.inner.KB().GetResourceTemplate(resId)
	if err != nil {
		return "", err
	}

	return emptyValue(tmpl, field)
}

func emptyValue(tmpl *knowledgebase.ResourceTemplate, property string) (any, error) {
	prop := tmpl.GetProperty(property)
	if prop == nil {
		return nil, fmt.Errorf("could not find property %s on template %s", property, tmpl.Id())
	}
	ptype, err := prop.PropertyType()
	if err != nil {
		return nil, fmt.Errorf(
			"could not get property type for property %s on template %s: %w",
			property, tmpl.Id(), err,
		)
	}
	return ptype.ZeroValue(), nil
}

func (ctx *fauxConfigContext) HasUpstream(selector any, resource construct.ResourceId) (bool, error) {
	has, innerErr := ctx.inner.HasUpstream(selector, resource)
	if innerErr == nil && has {
		return true, nil
	}

	selId, err := knowledgebase.TemplateArgToRID(selector)
	if err != nil {
		return false, err
	}

	if ctx.graphState == nil {
		ctx.graphState = make(graphStates)
	}
	ctx.graphState[fmt.Sprintf("hasUpstream(%s, %s)", selId, resource)] = func(g construct.Graph) (bool, error) {
		upstream, err := knowledgebase.Upstream(g, ctx.KB(), resource, knowledgebase.FirstFunctionalLayer)
		if err != nil {
			return false, err
		}
		for _, up := range upstream {
			if selId.Matches(up) {
				return true, nil
			}
		}
		return false, nil
	}

	return has, innerErr
}

func (ctx *fauxConfigContext) Upstream(selector any, resource construct.ResourceId) (construct.ResourceId, error) {
	up, innerErr := ctx.inner.Upstream(selector, resource)
	if innerErr == nil && !up.IsZero() {
		return up, nil
	}

	selId, err := knowledgebase.TemplateArgToRID(selector)
	if err != nil {
		return construct.ResourceId{}, err
	}

	if ctx.graphState == nil {
		ctx.graphState = make(graphStates)
	}
	ctx.graphState[fmt.Sprintf("Upstream(%s, %s)", selId, resource)] = func(g construct.Graph) (bool, error) {
		upstream, err := knowledgebase.Upstream(g, ctx.KB(), resource, knowledgebase.FirstFunctionalLayer)
		if err != nil {
			return false, err
		}
		for _, up := range upstream {
			if selId.Matches(up) {
				return true, nil
			}
		}
		return false, nil
	}

	return up, innerErr
}

func (ctx *fauxConfigContext) AllUpstream(selector any, resource construct.ResourceId) (construct.ResourceList, error) {
	if ctx.graphState == nil {
		ctx.graphState = make(graphStates)
	}
	ctx.graphState[fmt.Sprintf("AllUpstream(%s, %s)", selector, resource)] = func(g construct.Graph) (bool, error) {
		// Can never say that [AllUpstream] is ready until it must be evaluated due to being one of the final ones
		return false, nil
	}

	return ctx.inner.AllUpstream(selector, resource)
}

func (ctx *fauxConfigContext) HasDownstream(selector any, resource construct.ResourceId) (bool, error) {
	has, innerErr := ctx.inner.HasDownstream(selector, resource)
	if innerErr == nil && has {
		return true, nil
	}

	selId, err := knowledgebase.TemplateArgToRID(selector)
	if err != nil {
		return false, err
	}

	if ctx.graphState == nil {
		ctx.graphState = make(graphStates)
	}
	ctx.graphState[fmt.Sprintf("hasDownstream(%s, %s)", selId, resource)] = func(g construct.Graph) (bool, error) {
		downstream, err := knowledgebase.Downstream(g, ctx.KB(), resource, knowledgebase.FirstFunctionalLayer)
		if err != nil {
			return false, err
		}
		for _, down := range downstream {
			if selId.Matches(down) {
				return true, nil
			}
		}
		return false, nil
	}

	return has, innerErr
}

func (ctx *fauxConfigContext) Downstream(selector any, resource construct.ResourceId) (construct.ResourceId, error) {
	down, innerErr := ctx.inner.Downstream(selector, resource)
	if innerErr == nil && !down.IsZero() {
		return down, nil
	}

	selId, err := knowledgebase.TemplateArgToRID(selector)
	if err != nil {
		return construct.ResourceId{}, err
	}

	if ctx.graphState == nil {
		ctx.graphState = make(graphStates)
	}
	ctx.graphState[fmt.Sprintf("downstream(%s, %s)", selId, resource)] = func(g construct.Graph) (bool, error) {
		downstream, err := knowledgebase.Downstream(g, ctx.KB(), resource, knowledgebase.FirstFunctionalLayer)
		if err != nil {
			return false, err
		}
		for _, down := range downstream {
			if selId.Matches(down) {
				return true, nil
			}
		}
		return false, nil
	}

	return down, innerErr
}

func (ctx *fauxConfigContext) ClosestDownstream(selector any, resource construct.ResourceId) (construct.ResourceId, error) {
	if ctx.graphState == nil {
		ctx.graphState = make(graphStates)
	}
	ctx.graphState[fmt.Sprintf("closestDownstream(%s, %s)", selector, resource)] = func(g construct.Graph) (bool, error) {
		// Can never say that [ClosestDownstream] is ready because something closer could always be added
		return false, nil
	}

	return ctx.inner.ClosestDownstream(selector, resource)
}

func (ctx *fauxConfigContext) AllDownstream(selector any, resource construct.ResourceId) (construct.ResourceList, error) {
	if ctx.graphState == nil {
		ctx.graphState = make(graphStates)
	}
	ctx.graphState[fmt.Sprintf("allDownstream(%s, %s)", selector, resource)] = func(g construct.Graph) (bool, error) {
		// Can never say that [AllDownstream] is ready until it must be evaluated due to being one of the final ones
		return false, nil
	}

	return ctx.inner.AllDownstream(selector, resource)
}
