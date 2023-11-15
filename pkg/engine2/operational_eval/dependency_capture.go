package operational_eval

import (
	"errors"
	"fmt"
	"io"
	"strings"
	"text/template"

	construct "github.com/klothoplatform/klotho/pkg/construct2"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base2"
	"go.uber.org/zap"
)

// fauxConfigContext acts like a [knowledgebase.DynamicValueContext] but replaces the [FieldValue] function
// with one that just returns the zero value of the property type and records the property reference.
type fauxConfigContext struct {
	propRef construct.PropertyRef
	inner   knowledgebase.DynamicValueContext
	changes graphChanges
	src     Key
}

func newDepCapture(inner knowledgebase.DynamicValueContext, changes graphChanges, src Key) *fauxConfigContext {
	return &fauxConfigContext{
		inner:   inner,
		changes: changes,
		src:     src,
	}
}

func (ctx *fauxConfigContext) addRef(ref construct.PropertyRef) {
	ctx.changes.addEdge(ctx.src, Key{Ref: ref})
}

func (ctx *fauxConfigContext) addGraphState(v *graphStateVertex) {
	ctx.changes.nodes[v.Key()] = v
	ctx.changes.addEdge(ctx.src, v.Key())
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
	err = ctx.inner.ExecuteTemplateDecode(t, data, value)
	if err != nil {
		zap.S().Debugf("ignoring error from ExecuteTemplateDecode during deps calculation on %s: %s", ctx.propRef, err)
	}
	return nil
}

func (ctx *fauxConfigContext) ExecuteValue(v any, data knowledgebase.DynamicValueData) {
	_, err := knowledgebase.TransformToPropertyValue(ctx.propRef.Resource, ctx.propRef.Property, v, ctx, data)
	if err != nil {
		zap.S().Debugf("ignoring error from TransformToPropertyValue during deps calculation on %s: %s", ctx.propRef, err)
	}
}

func (ctx *fauxConfigContext) Execute(v any, data knowledgebase.DynamicValueData) error {
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

func (ctx *fauxConfigContext) DecodeConfigRef(
	data knowledgebase.DynamicValueData,
	rule knowledgebase.ConfigurationRule,
) (construct.PropertyRef, error) {
	var ref construct.PropertyRef
	err := ctx.ExecuteDecode(rule.Config.Field, data, &ref.Property)
	if err != nil {
		return ref, fmt.Errorf("could not execute field template: %w", err)
	}
	err = ctx.ExecuteDecode(rule.Resource, data, &ref.Resource)
	if err != nil {
		return ref, fmt.Errorf("could not execute resource template: %w", err)
	}
	return ref, nil
}

func (ctx *fauxConfigContext) ExecuteOpRule(
	data knowledgebase.DynamicValueData,
	opRule knowledgebase.OperationalRule,
) error {
	var errs error
	exec := func(v any) {
		errs = errors.Join(errs, ctx.Execute(v, data))
	}
	originalSrc := ctx.src
	for _, rule := range opRule.ConfigurationRules {
		if rule.Resource != "" {
			ref, err := ctx.DecodeConfigRef(data, rule)
			if err != nil {
				errs = errors.Join(errs, err)
				continue
			}
			if ref.Resource.IsZero() || ref.Property == "" {
				// Can't determine the ref yet, continue
				// NOTE(gg): It's possible that whatever this will eventually resolve to
				// would get evaluated before this has a chance to add the dependency.
				// If that ever occurs, we may need to add speculative dependencies
				// for all refs that could match this.
				continue
			}
			// set the source to the ref that is being configured, not necessarily the key that dependencies are being
			// calculated for
			ctx.src = Key{Ref: ref}
		}
		exec(opRule.If)
		ctx.ExecuteValue(rule.Config.Value, data)
		if ctx.src != originalSrc {
			// Make sure the configured property depends on the edge
			ctx.changes.addEdge(ctx.src, originalSrc)
			// reset inside the loop in case the next rule doesn't specify the ref
			ctx.src = originalSrc
		}
	}
	if len(opRule.Steps) > 0 {
		exec(opRule.If)
	}
	for _, step := range opRule.Steps {
		for _, stepRes := range step.Resources {
			exec(stepRes.Selector)
			for _, propValue := range stepRes.Properties {
				exec(propValue)
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
	ref := construct.PropertyRef{
		Resource: resId,
		Property: field,
	}
	if bracketIdx := strings.Index(field, "["); bracketIdx != -1 {
		// Cannot depend on properties within lists, stop at the list itself
		ref.Property = field[:bracketIdx]
	}
	ctx.addRef(ref)

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
	ref := construct.PropertyRef{
		Resource: resId,
		Property: field,
	}

	value, err := ctx.inner.FieldValue(field, resId)
	if err != nil {
		if bracketIdx := strings.Index(field, "["); bracketIdx != -1 {
			// Cannot depend on properties within lists, stop at the list itself
			ref.Property = field[:bracketIdx]
		}
		ctx.addRef(ref)
		return nil, err
	}
	ctx.addRef(ref)
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

	ctx.addGraphState(&graphStateVertex{
		repr: graphStateRepr(fmt.Sprintf("hasUpstream(%s, %s)", selId, resource)),
		Test: func(g construct.Graph) (ReadyPriority, error) {
			upstream, err := knowledgebase.Upstream(g, ctx.KB(), resource, knowledgebase.FirstFunctionalLayer)
			if err != nil {
				return NotReadyMax, err
			}
			for _, up := range upstream {
				if selId.Matches(up) {
					return ReadyNow, nil
				}
			}
			return NotReadyMid, nil
		},
	})

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

	ctx.addGraphState(&graphStateVertex{
		repr: graphStateRepr(fmt.Sprintf("Upstream(%s, %s)", selId, resource)),
		Test: func(g construct.Graph) (ReadyPriority, error) {
			upstream, err := knowledgebase.Upstream(g, ctx.KB(), resource, knowledgebase.FirstFunctionalLayer)
			if err != nil {
				return NotReadyMax, err
			}
			for _, up := range upstream {
				if selId.Matches(up) {
					return ReadyNow, nil
				}
			}
			return NotReadyMid, nil
		},
	})

	return up, innerErr
}

func (ctx *fauxConfigContext) AllUpstream(selector any, resource construct.ResourceId) (construct.ResourceList, error) {
	ctx.addGraphState(&graphStateVertex{
		repr: graphStateRepr(fmt.Sprintf("AllUpstream(%s, %s)", selector, resource)),
		Test: func(g construct.Graph) (ReadyPriority, error) {
			// Can never say that [AllUpstream] is ready until it must be evaluated due to being one of the final ones
			return NotReadyHigh, nil
		},
	})

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

	ctx.addGraphState(&graphStateVertex{
		repr: graphStateRepr(fmt.Sprintf("hasDownstream(%s, %s)", selId, resource)),
		Test: func(g construct.Graph) (ReadyPriority, error) {
			downstream, err := knowledgebase.Downstream(g, ctx.KB(), resource, knowledgebase.FirstFunctionalLayer)
			if err != nil {
				return NotReadyMax, err
			}
			for _, down := range downstream {
				if selId.Matches(down) {
					return ReadyNow, nil
				}
			}
			return NotReadyMid, nil
		},
	})

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

	ctx.addGraphState(&graphStateVertex{
		repr: graphStateRepr(fmt.Sprintf("downstream(%s, %s)", selId, resource)),
		Test: func(g construct.Graph) (ReadyPriority, error) {
			downstream, err := knowledgebase.Downstream(g, ctx.KB(), resource, knowledgebase.FirstFunctionalLayer)
			if err != nil {
				return NotReadyMax, err
			}
			for _, down := range downstream {
				if selId.Matches(down) {
					return ReadyNow, nil
				}
			}
			return NotReadyMid, nil
		},
	})

	return down, innerErr
}

func (ctx *fauxConfigContext) ClosestDownstream(selector any, resource construct.ResourceId) (construct.ResourceId, error) {
	ctx.addGraphState(&graphStateVertex{
		repr: graphStateRepr(fmt.Sprintf("closestDownstream(%s, %s)", selector, resource)),
		Test: func(g construct.Graph) (ReadyPriority, error) {
			// Can never say that [ClosestDownstream] is ready because something closer could always be added
			return NotReadyMid, nil
		},
	})

	return ctx.inner.ClosestDownstream(selector, resource)
}

func (ctx *fauxConfigContext) AllDownstream(selector any, resource construct.ResourceId) (construct.ResourceList, error) {
	ctx.addGraphState(&graphStateVertex{
		repr: graphStateRepr(fmt.Sprintf("allDownstream(%s, %s)", selector, resource)),
		Test: func(g construct.Graph) (ReadyPriority, error) {
			// Can never say that [AllDownstream] is ready until it must be evaluated due to being one of the final ones
			return NotReadyHigh, nil
		},
	})

	return ctx.inner.AllDownstream(selector, resource)
}
