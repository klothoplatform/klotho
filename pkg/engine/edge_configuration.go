package engine

import (
	"errors"
	"fmt"
	"strings"

	"github.com/klothoplatform/klotho/pkg/construct"
	"github.com/klothoplatform/klotho/pkg/graph"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

func (e *Engine) configureEdges(graph *construct.ResourceGraph) (map[construct.ResourceId]map[construct.ResourceId]bool, error) {
	configuredEdges := map[construct.ResourceId]map[construct.ResourceId]bool{}
	joinedErr := error(nil)
	zap.S().Debug("Engine configuring edges")
	for _, dep := range graph.ListDependencies() {
		if _, ok := configuredEdges[dep.Source.Id()]; !ok {
			configuredEdges[dep.Source.Id()] = make(map[construct.ResourceId]bool)
		}
		templateKey := fmt.Sprintf("%s:%s:-%s:%s:", dep.Source.Id().Provider, dep.Source.Id().Type, dep.Destination.Id().Provider, dep.Destination.Id().Type)
		_, found := e.KnowledgeBase.GetResourceEdge(dep.Source, dep.Destination)
		if e.EdgeTemplates[templateKey] == nil && !found {
			zap.Error(fmt.Errorf("no edge template found for %s", templateKey))
			joinedErr = errors.Join(joinedErr, fmt.Errorf("no edge template found for %s", templateKey))
			continue
		}

		if e.EdgeTemplates[templateKey] != nil {
			resourceMap, err := e.EdgeTemplateExpand(*e.EdgeTemplates[templateKey], graph, &dep)
			if err != nil {
				joinedErr = errors.Join(joinedErr, err)
				continue
			}
			err = e.EdgeTemplateMakeOperational(*e.EdgeTemplates[templateKey], graph, &dep, resourceMap)
			if err != nil {
				joinedErr = errors.Join(joinedErr, err)
				continue
			}
			err = EdgeTemplateConfigure(*e.EdgeTemplates[templateKey], graph, &dep, resourceMap)
			if err != nil {
				joinedErr = errors.Join(joinedErr, err)
				continue
			}
		}

		err := e.KnowledgeBase.ConfigureEdge(&dep, graph)
		if err != nil {
			joinedErr = errors.Join(joinedErr, err)
			continue
		}
		configuredEdges[dep.Source.Id()][dep.Destination.Id()] = true

	}
	return configuredEdges, joinedErr
}

func (e *Engine) EdgeTemplateExpand(template knowledgebase.EdgeTemplate, graph *construct.ResourceGraph, edge *graph.Edge[construct.Resource]) (map[construct.ResourceId]construct.Resource, error) {
	joinedErr := error(nil)
	resourceMap := map[construct.ResourceId]construct.Resource{}
	resourceMap[template.Source] = edge.Source
	resourceMap[template.Destination] = edge.Destination
	for _, res := range template.Expansion.Resources {
		provider := e.Providers[res.Provider]
		resWithName := res
		resWithName.Name = nameResourceFromEdge(edge, res)
		node, err := provider.CreateConstructFromId(resWithName, e.Context.InitialState)
		if err != nil {
			joinedErr = errors.Join(joinedErr, err)
			continue
		}
		if r, ok := node.(construct.Resource); ok {
			graph.AddResource(r)
			resourceMap[res] = r
		} else {
			joinedErr = errors.Join(joinedErr, fmt.Errorf("node %s is not a resource (was %T)", node.Id(), node))
			continue
		}
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
	return resourceMap, joinedErr
}

func EdgeTemplateConfigure(template knowledgebase.EdgeTemplate, graph *construct.ResourceGraph, edge *graph.Edge[construct.Resource], resourceMap map[construct.ResourceId]construct.Resource) error {
	joinedErr := error(nil)
	for _, config := range template.Configuration {
		id, fields := getIdAndFields(config.Resource)
		res := resourceMap[id]
		res, err := getResourceFromIdString(res, fields, graph)
		if err != nil {
			joinedErr = errors.Join(joinedErr, err)
			continue
		}
		if res == nil {
			joinedErr = errors.Join(joinedErr, fmt.Errorf("resource %s not found when attempting to configure", id.String()))
			continue
		}
		newConfig := construct.Configuration{}
		valBytes, err := yaml.Marshal(config.Config)
		if err != nil {
			joinedErr = errors.Join(joinedErr, err)
			continue
		}
		valStr := string(valBytes)
		for id, resource := range resourceMap {
			valStr = strings.ReplaceAll(valStr, id.String(), resource.Id().String())
		}
		err = yaml.Unmarshal([]byte(valStr), &newConfig)
		if err != nil {
			joinedErr = errors.Join(joinedErr, err)
			continue
		}
		err = ConfigureField(res, newConfig.Field, newConfig.Value, config.Config.ZeroValueAllowed, graph)
		if err != nil {
			joinedErr = errors.Join(joinedErr, err)
			continue
		}
	}
	return joinedErr
}

func (e *Engine) EdgeTemplateMakeOperational(template knowledgebase.EdgeTemplate, graph *construct.ResourceGraph, edge *graph.Edge[construct.Resource], resourceMap map[construct.ResourceId]construct.Resource) error {
	joinedErr := error(nil)
	for _, rule := range template.OperationalRules {
		id, fields := getIdAndFields(rule.Resource)
		res := resourceMap[id]
		resource, err := getResourceFromIdString(res, fields, graph)
		if err != nil {
			joinedErr = errors.Join(joinedErr, err)
			continue
		}
		errs := e.handleOperationalRule(resource, rule.Rule, graph, nil)
		if errs != nil {
			for _, err := range errs {
				if ore, ok := err.(*OperationalResourceError); ok {
					err = e.handleOperationalResourceError(ore, graph)
					if err != nil {
						joinedErr = errors.Join(joinedErr, err)
					}
					continue
				}
				joinedErr = errors.Join(joinedErr, err)
			}
			continue
		}
	}
	return joinedErr
}

func nameResourceFromEdge(edge *graph.Edge[construct.Resource], res construct.ResourceId) string {
	return fmt.Sprintf("%s-%s-%s", edge.Source.Id().Name, edge.Destination.Id().Name, res.Name)
}

func getResourceFromIdString(res construct.Resource, fields string, dag *construct.ResourceGraph) (construct.Resource, error) {
	if fields == "" {
		return res, nil
	}
	// we pass in false for the parseFieldName's configure param so that we dont create a resource's interface if it is currently nil, leading to us adding extra resources
	field, _, err := parseFieldName(res, fields, dag, false)
	if err != nil {
		return nil, err
	}
	if !field.IsValid() {
		return nil, fmt.Errorf("field %s on resource %s is invalid", fields, res.Id())
	} else if field.IsNil() {
		return nil, fmt.Errorf("field %s on resource %s is nil", fields, res.Id())
	}
	res = field.Interface().(construct.Resource)
	return res, nil
}
