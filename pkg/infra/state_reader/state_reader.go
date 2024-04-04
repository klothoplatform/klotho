package statereader

import (
	"errors"
	"fmt"
	"io"
	"reflect"
	"strings"

	"github.com/dominikbraun/graph"
	"github.com/klothoplatform/klotho/pkg/construct"
	stateconverter "github.com/klothoplatform/klotho/pkg/infra/state_reader/state_converter"
	statetemplate "github.com/klothoplatform/klotho/pkg/infra/state_reader/state_template"
	"github.com/klothoplatform/klotho/pkg/knowledgebase"
	"github.com/klothoplatform/klotho/pkg/knowledgebase/properties"
)

//go:generate 	mockgen -source=./state_reader.go --destination=./state_reader_mock_test.go --package=statereader

type (
	// StateReader is an interface for reading state from a state store
	StateReader interface {
		// ReadState reads the state from the state store
		ReadState(io.Reader) (construct.Graph, error)
	}

	propertyCorrelation interface {
		setProperty(
			resource *construct.Resource,
			property string,
			value any,
		) error
		checkValue(
			step knowledgebase.OperationalStep,
			value string,
			src construct.ResourceId,
			propertyRef string,
		) (*construct.Edge, *construct.PropertyRef, error)
	}

	stateReader struct {
		templates map[string]statetemplate.StateTemplate
		kb        knowledgebase.TemplateKB
		converter stateconverter.StateConverter
		graph     construct.Graph
	}

	propertyCorrelator struct {
		ctx       knowledgebase.DynamicValueContext
		resources []*construct.Resource
	}
)

func NewPulumiReader(g construct.Graph, templates map[string]statetemplate.StateTemplate, kb knowledgebase.TemplateKB) StateReader {
	return &stateReader{graph: g, templates: templates, kb: kb, converter: stateconverter.NewStateConverter("pulumi", templates)}
}

func (p stateReader) ReadState(reader io.Reader) (construct.Graph, error) {
	internalState, err := p.converter.ConvertState(reader)
	if err != nil {
		return nil, err
	}
	if p.graph == nil {
		p.graph = construct.NewGraph()
	}
	if err = p.loadGraph(internalState); err != nil {
		return p.graph, err
	}
	existingResources := make([]*construct.Resource, 0)
	adj, err := p.graph.AdjacencyMap()
	if err != nil {
		return p.graph, err
	}
	for id := range adj {
		r, err := p.graph.Vertex(id)
		if err != nil {
			return p.graph, err
		}
		existingResources = append(existingResources, r)
	}

	ctx := knowledgebase.DynamicValueContext{Graph: p.graph, KnowledgeBase: p.kb}
	pc := propertyCorrelator{
		ctx:       ctx,
		resources: existingResources,
	}
	if err = p.loadProperties(internalState, pc); err != nil {
		return p.graph, err
	}

	return p.graph, nil
}

func (p stateReader) loadGraph(state stateconverter.State) error {
	var errs error
	for id, properties := range state {
		resource, err := p.graph.Vertex(id)
		if err != nil && !errors.Is(err, graph.ErrVertexNotFound) {
			errs = errors.Join(errs, err)
			continue
		}
		if resource == nil {
			resource = &construct.Resource{
				ID:         id,
				Properties: make(construct.Properties),
			}
			err = p.graph.AddVertex(resource)
			if err != nil {
				errs = errors.Join(errs, err)
				continue
			}
		}
		rt, err := p.kb.GetResourceTemplate(id)
		if err != nil {
			errs = errors.Join(errs, err)
			continue
		} else if rt == nil {
			errs = errors.Join(errs, fmt.Errorf("resource template not found for resource %s", id))
			continue
		}
		for key, value := range properties {
			if !strings.Contains(key, "#") {
				prop := rt.GetProperty(key)
				if prop == nil {
					errs = errors.Join(errs, fmt.Errorf("property %s not found in resource template %s", key, id))
					continue
				}
				errs = errors.Join(errs, prop.SetProperty(resource, value))
			}
		}
	}
	return errs
}

func (p stateReader) loadProperties(state stateconverter.State, pc propertyCorrelation) error {
	var errs error
	for id, properties := range state {
		resource, err := p.graph.Vertex(id)
		if err != nil {
			errs = errors.Join(errs, err)
			continue
		}
		for key, value := range properties {
			errs = errors.Join(errs, pc.setProperty(resource, key, value))
		}
	}
	return errs
}

func (p propertyCorrelator) setProperty(
	resource *construct.Resource,
	property string,
	value any,
) error {
	edges := make([]*construct.Edge, 0)
	rt, err := p.ctx.KnowledgeBase.GetResourceTemplate(resource.ID)
	if err != nil {
		return err
	} else if rt == nil {
		return fmt.Errorf("resource template not found for resource %s", resource.ID)
	}
	parts := strings.Split(property, "#")
	property = parts[0]
	prop := rt.GetProperty(property)
	if prop == nil {
		return fmt.Errorf("property %s not found in resource template %s", property, resource.ID)
	}
	opRule := prop.Details().OperationalRule
	if opRule == nil || len(opRule.Step.Resources) == 0 {
		return resource.SetProperty(property, value)
	}
	var ref string
	if len(parts) > 1 {
		ref = parts[1]
	} else {
		ref = opRule.Step.UsePropertyRef
	}

	switch rval := reflect.ValueOf(value); rval.Kind() {
	case reflect.String:
		edge, pref, err := p.checkValue(opRule.Step, value.(string), resource.ID, ref)
		if err != nil {
			return err
		}
		if edge != nil {
			edges = append(edges, edge)
		}
		if pref != nil {
			switch prop.(type) {
			case *properties.ResourceProperty:
				err = prop.SetProperty(resource, pref.Resource)
				if err != nil {
					return err
				}
			default:
				err = prop.SetProperty(resource, pref)
				if err != nil {
					return err
				}
			}

		}
	case reflect.Slice, reflect.Array:
		var val []any
		for i := 0; i < rval.Len(); i++ {
			edge, pref, err := p.checkValue(opRule.Step, rval.Index(i).Interface().(string), resource.ID, ref)
			if err != nil {
				return err
			}
			if edge != nil {
				edges = append(edges, edge)
			}
			if pref != nil {
				val = append(val, *pref)
			}
		}
		collectionProp, ok := prop.(knowledgebase.CollectionProperty)
		if !ok {
			return fmt.Errorf("property %s is not a collection property", property)
		}
		switch collectionProp.Item().(type) {
		case *properties.ResourceProperty:
			resources := make([]construct.ResourceId, 0)
			for _, v := range val {
				resources = append(resources, v.(construct.PropertyRef).Resource)
			}
			err = prop.SetProperty(resource, resources)
			if err != nil {
				return err
			}
		default:
			err = prop.SetProperty(resource, val)
			if err != nil {
				return err
			}
		}
	}
	for _, edge := range edges {
		err := p.ctx.Graph.AddEdge(edge.Source, edge.Target)
		if err != nil {
			return err
		}
	}
	return nil
}

func (p propertyCorrelator) checkValue(
	step knowledgebase.OperationalStep,
	value string,
	src construct.ResourceId,
	propertyRef string,
) (*construct.Edge, *construct.PropertyRef, error) {
	var possibleIds []construct.ResourceId
	data := knowledgebase.DynamicValueData{Resource: src}
	for _, selector := range step.Resources {
		ids, err := selector.ExtractResourceIds(p.ctx, data)
		if err != nil {
			return nil, nil, err
		}
		possibleIds = append(possibleIds, ids...)
		for _, id := range ids {
			for _, resource := range p.resources {
				if id.Matches(resource.ID) {
					val, err := p.ctx.FieldValue(propertyRef, resource.ID)
					if err != nil {
						return nil, nil, err
					}
					if val == value {
						if step.Direction == knowledgebase.DirectionDownstream {
							return &construct.Edge{
									Source: src,
									Target: resource.ID,
								}, &construct.PropertyRef{
									Resource: resource.ID,
									Property: propertyRef,
								}, nil
						} else {
							return &construct.Edge{
									Source: resource.ID,
									Target: src,
								}, &construct.PropertyRef{
									Resource: resource.ID,
									Property: propertyRef,
								}, nil
						}
					}
				}
			}
		}
	}
	if len(step.Resources) == 1 {
		idToUse := possibleIds[0]
		newRes := &construct.Resource{
			ID: construct.ResourceId{Provider: idToUse.Provider, Type: idToUse.Type, Name: value},
		}
		err := newRes.SetProperty(propertyRef, value)
		if err != nil {
			return nil, nil, err
		}
		err = p.ctx.Graph.AddVertex(newRes)
		if err != nil {
			return nil, nil, err
		}
		if step.Direction == knowledgebase.DirectionDownstream {
			return &construct.Edge{
					Source: src,
					Target: newRes.ID,
				}, &construct.PropertyRef{
					Resource: newRes.ID,
					Property: propertyRef,
				}, nil
		} else {
			return &construct.Edge{
					Source: newRes.ID,
					Target: src,
				}, &construct.PropertyRef{
					Resource: newRes.ID,
					Property: propertyRef,
				}, nil
		}
	}
	return nil, nil, nil
}
