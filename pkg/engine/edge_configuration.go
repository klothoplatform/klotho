package engine

import (
	"fmt"
	"strings"

	"github.com/klothoplatform/klotho/pkg/construct"
	"github.com/klothoplatform/klotho/pkg/graph"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base"
	"gopkg.in/yaml.v3"
)

func (e *Engine) configureEdge(dep graph.Edge[construct.Resource], context *SolveContext) []EngineError {
	templateKey := fmt.Sprintf("%s:%s:-%s:%s:", dep.Source.Id().Provider, dep.Source.Id().Type, dep.Destination.Id().Provider, dep.Destination.Id().Type)
	_, found := e.KnowledgeBase.GetResourceEdge(dep.Source, dep.Destination)
	if e.EdgeTemplates[templateKey] == nil && !found {
		return []EngineError{&InternalError{Child: &EdgeConfigurationError{Edge: dep}, Cause: fmt.Errorf("no edge template found for %s", templateKey)}}
	}

	if e.EdgeTemplates[templateKey] != nil {
		resourceMap := &map[construct.ResourceId]construct.Resource{}
		decisions, engineErrors := e.EdgeTemplateExpand(*e.EdgeTemplates[templateKey], context.ResourceGraph, &dep, resourceMap)
		e.handleDecisions(context, decisions)
		if engineErrors != nil {
			return engineErrors
		}

		decisions, engineErrors = e.EdgeTemplateMakeOperational(*e.EdgeTemplates[templateKey], context.ResourceGraph, &dep, *resourceMap)
		e.handleDecisions(context, decisions)
		if engineErrors != nil {
			return engineErrors
		}

		decisions, engineErrors = EdgeTemplateConfigure(*e.EdgeTemplates[templateKey], context.ResourceGraph, &dep, *resourceMap)
		e.handleDecisions(context, decisions)
		if engineErrors != nil {
			return engineErrors
		}
	}

	err := e.KnowledgeBase.ConfigureEdge(&dep, context.ResourceGraph)
	if err != nil {
		return []EngineError{&EdgeConfigurationError{
			Edge:  dep,
			Cause: err,
		}}
	}
	return nil
}

func (e *Engine) EdgeTemplateExpand(template knowledgebase.EdgeTemplate, resourceGraph *construct.ResourceGraph, edge *graph.Edge[construct.Resource], resourcemap *map[construct.ResourceId]construct.Resource) (decisions []Decision, engineErrors []EngineError) {
	resourceMap := map[construct.ResourceId]construct.Resource{}
	resourceMap[template.Source] = edge.Source
	resourceMap[template.Destination] = edge.Destination
	for _, res := range template.Expansion.Resources {
		provider := e.Providers[res.Provider]
		resWithName := res
		resWithName.Name = nameResourceFromEdge(edge, res)
		node, err := provider.CreateConstructFromId(resWithName, e.Context.InitialState)
		if err != nil {
			engineErrors = append(engineErrors, &EdgeConfigurationError{
				Edge:  *edge,
				Cause: err,
			})
			continue
		}
		if r, ok := node.(construct.Resource); ok {
			decisions = append(decisions, Decision{
				Level:  LevelInfo,
				Result: &DecisionResult{Resource: r},
				Action: ActionCreate,
				Cause: &Cause{
					EdgeExpansion: edge,
				},
			})
			resourceMap[res] = r
		} else {
			engineErrors = append(engineErrors, &InternalError{
				Child: &EdgeConfigurationError{Edge: *edge},
				Cause: fmt.Errorf("node %s is not a resource (was %T)", node.Id(), node),
			})
			continue
		}
	}
	if engineErrors != nil {
		return
	}

	for _, dep := range template.Expansion.Dependencies {
		id, fields := getIdAndFields(dep.Source)
		srcRes := resourceGraph.GetResource(resourceMap[id].Id())
		src, err := getResourceFromIdString(srcRes, fields, resourceGraph)
		if err != nil {
			engineErrors = append(engineErrors, &InternalError{
				Child: &EdgeConfigurationError{Edge: *edge},
				Cause: err,
			})
			continue
		}
		// if the src is nil its because we havent created it yet and it does not yet exist in the graph
		if src == nil {
			src = resourceMap[id]
		}
		id, fields = getIdAndFields(dep.Destination)
		dstRes := resourceGraph.GetResource(resourceMap[id].Id())
		dst, err := getResourceFromIdString(dstRes, fields, resourceGraph)
		if err != nil {
			engineErrors = append(engineErrors, &InternalError{
				Child: &EdgeConfigurationError{Edge: *edge},
				Cause: err,
			})
			continue
		}
		// if the src is nil its because we havent created it yet and it does not yet exist in the graph
		if dst == nil {
			dst = resourceMap[id]
		}
		decisions = append(decisions, Decision{
			Level:  LevelInfo,
			Result: &DecisionResult{Edge: &graph.Edge[construct.Resource]{Source: src, Destination: dst}},
			Action: ActionConnect,
			Cause: &Cause{
				EdgeExpansion: edge,
			},
		})
	}
	return
}

func EdgeTemplateConfigure(template knowledgebase.EdgeTemplate, graph *construct.ResourceGraph, edge *graph.Edge[construct.Resource], resourceMap map[construct.ResourceId]construct.Resource) (decisions []Decision, engineErrors []EngineError) {
	for _, config := range template.Configuration {
		id, fields := getIdAndFields(config.Resource)
		res := resourceMap[id]
		res, err := getResourceFromIdString(res, fields, graph)
		if err != nil {
			engineErrors = append(engineErrors, &InternalError{
				Child: &EdgeConfigurationError{Edge: *edge},
				Cause: err,
			})
			continue
		}
		if res == nil {
			engineErrors = append(engineErrors, &EdgeConfigurationError{
				Edge:  *edge,
				Cause: fmt.Errorf("resource %s not found when attempting to configure", id.String()),
			})
			continue
		}
		newConfig := knowledgebase.Configuration{}
		valBytes, err := yaml.Marshal(config.Config)
		if err != nil {
			engineErrors = append(engineErrors, &InternalError{
				Child: &EdgeConfigurationError{Edge: *edge},
				Cause: err,
			})
			continue
		}
		valStr := string(valBytes)
		for id, resource := range resourceMap {
			valStr = strings.ReplaceAll(valStr, id.String(), resource.Id().String())
		}
		err = yaml.Unmarshal([]byte(valStr), &newConfig)
		if err != nil {
			engineErrors = append(engineErrors, &InternalError{
				Child: &EdgeConfigurationError{Edge: *edge},
				Cause: err,
			})
			continue
		}
		decisions = append(decisions, Decision{
			Level:  LevelInfo,
			Result: &DecisionResult{Resource: res, Config: &config},
			Action: ActionConfigure,
			Cause: &Cause{
				EdgeExpansion: edge,
			},
		})
	}
	return
}

func (e *Engine) EdgeTemplateMakeOperational(template knowledgebase.EdgeTemplate, graph *construct.ResourceGraph, edge *graph.Edge[construct.Resource], resourceMap map[construct.ResourceId]construct.Resource) (decisions []Decision, engineErrors []EngineError) {
	for _, rule := range template.OperationalRules {
		id, fields := getIdAndFields(rule.Resource)
		res := resourceMap[id]
		resource, err := getResourceFromIdString(res, fields, graph)
		if err != nil {
			engineErrors = append(engineErrors, &InternalError{
				Child: &EdgeConfigurationError{Edge: *edge},
				Cause: err,
			})
			continue
		}
		ruleDecisions, errs := e.handleOperationalRule(resource, rule.Rule, graph, nil)
		if errs != nil {
			for _, err := range errs {
				if ore, ok := err.(*OperationalResourceError); ok {
					oreDecisions, err := e.handleOperationalResourceError(ore, graph)
					ruleDecisions = append(ruleDecisions, oreDecisions...)
					if err != nil {
						engineErrors = append(engineErrors, &EdgeConfigurationError{
							Edge:  *edge,
							Cause: err,
						})
					}
				} else {
					engineErrors = append(engineErrors, err)
				}
			}
			continue
		}
		for _, d := range ruleDecisions {
			d.Cause = &Cause{EdgeConfiguration: edge}
			decisions = append(decisions, d)
		}
	}

	return
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
