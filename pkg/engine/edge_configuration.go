package engine

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/graph"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

func (e *Engine) configureEdges(graph *core.ResourceGraph) (map[core.ResourceId]map[core.ResourceId]bool, error) {
	configuredEdges := map[core.ResourceId]map[core.ResourceId]bool{}
	joinedErr := error(nil)
	zap.S().Debug("Engine configuring edges")
	for _, dep := range graph.ListDependencies() {
		if _, ok := configuredEdges[dep.Source.Id()]; !ok {
			configuredEdges[dep.Source.Id()] = make(map[core.ResourceId]bool)
		}
		templateKey := fmt.Sprintf("%s:%s:-%s:%s:", dep.Source.Id().Provider, dep.Source.Id().Type, dep.Destination.Id().Provider, dep.Destination.Id().Type)
		if e.EdgeTemplates[templateKey] != nil {
			err := e.EdgeTemplateExpand(*e.EdgeTemplates[templateKey], graph, &dep)
			if err != nil {
				joinedErr = errors.Join(joinedErr, err)
				continue
			}
			err = EdgeTemplateConfigure(*e.EdgeTemplates[templateKey], graph, &dep)
			if err != nil {
				joinedErr = errors.Join(joinedErr, err)
				continue
			}
			configuredEdges[dep.Source.Id()][dep.Destination.Id()] = true

		} else {
			err := e.KnowledgeBase.ConfigureEdge(&dep, graph)
			if err != nil {
				joinedErr = errors.Join(joinedErr, err)
				continue
			}
			configuredEdges[dep.Source.Id()][dep.Destination.Id()] = true

		}
	}
	return configuredEdges, joinedErr
}

func (e *Engine) EdgeTemplateExpand(template knowledgebase.EdgeTemplate, graph *core.ResourceGraph, edge *graph.Edge[core.Resource]) error {
	joinedErr := error(nil)
	resourceMap := map[core.ResourceId]core.Resource{}
	resourceMap[template.Source] = edge.Source
	resourceMap[template.Destination] = edge.Destination
	for _, res := range template.Expansion.Resources {
		provider := e.Providers[res.Provider]
		resWithName := res
		resWithName.Name = nameResourceFromEdge(edge, res)
		node, err := provider.CreateResourceFromId(resWithName, e.Context.InitialState)
		if err != nil {
			joinedErr = errors.Join(joinedErr, err)
			continue
		}
		graph.AddResource(node)
		resourceMap[res] = node
	}
	for _, dep := range template.Expansion.Dependencies {
		id, fields := getIdAndFields(dep.Source)
		srcRes := graph.GetResource(resourceMap[id].Id())
		src, err := getResourceFromIdString(srcRes, fields, graph)
		if err != nil {
			joinedErr = errors.Join(joinedErr, err)
			continue
		}
		id, fields = getIdAndFields(dep.Destination)
		dstRes := graph.GetResource(resourceMap[id].Id())
		dst, err := getResourceFromIdString(dstRes, fields, graph)
		if err != nil {
			joinedErr = errors.Join(joinedErr, err)
			continue
		}
		graph.AddDependency(src, dst)
	}
	return nil
}

func nameResourceFromEdge(edge *graph.Edge[core.Resource], res core.ResourceId) string {
	return fmt.Sprintf("%s-%s-%s", edge.Source.Id().Name, edge.Destination.Id().Name, res.Name)
}

func EdgeTemplateConfigure(template knowledgebase.EdgeTemplate, graph *core.ResourceGraph, edge *graph.Edge[core.Resource]) error {
	for _, config := range template.Configuration {
		resId := config.Resource
		switch resId {
		case template.Source:
			resId = edge.Source.Id()
		case template.Destination:
			resId = edge.Destination.Id()
		default:
			return fmt.Errorf("configuration may only be on resources in edge: source: %s, dest: %s, but was on %s", template.Source, template.Destination, resId)
		}
		res := graph.GetResource(resId)
		if res == nil {
			return fmt.Errorf("unable to find resource %s", resId)
		}
		newConfig := core.Configuration{}
		valBytes, err := yaml.Marshal(config.Config)
		if err != nil {
			return err
		}
		valStr := string(valBytes)
		valStr = strings.ReplaceAll(valStr, template.Source.String(), edge.Source.Id().String())
		valStr = strings.ReplaceAll(valStr, template.Destination.String(), edge.Destination.Id().String())
		err = yaml.Unmarshal([]byte(valStr), &newConfig)
		if err != nil {
			return err
		}
		err = ConfigureField(res, config.Config.Field, newConfig.Value, graph)
		if err != nil {
			return err
		}
	}
	return nil
}

func getResourceFromIdString(res core.Resource, fields string, dag *core.ResourceGraph) (core.Resource, error) {
	if fields == "" {
		return res, nil
	} else {
		subFields := strings.Split(fields, ".")
		var field reflect.Value
		for i := 0; i < len(subFields); i++ {
			if i == 0 {
				field = reflect.ValueOf(res).Elem().FieldByName(subFields[i])
			} else {
				field = reflect.ValueOf(field).Elem().FieldByName(subFields[i])
			}
			if !field.IsValid() {
				return nil, fmt.Errorf("unable to find field %s on resource %s", subFields[i], res.Id())
			}
		}
		if !field.IsValid() {
			return nil, fmt.Errorf("unable to find field %s on resource %s", subFields[len(subFields)-1], res.Id())
		} else if field.IsNil() {
			return nil, fmt.Errorf("field %s on resource %s is nil", subFields[len(subFields)-1], res.Id())
		}
		res = field.Interface().(core.Resource)
		return res, nil
	}
}

func getIdAndFields(id core.ResourceId) (core.ResourceId, string) {
	arr := strings.Split(id.String(), "#")
	resId := &core.ResourceId{}
	err := resId.UnmarshalText([]byte(arr[0]))
	if err != nil {
		return core.ResourceId{}, ""
	}
	if len(arr) == 1 {
		return *resId, ""
	}
	return *resId, arr[1]
}
