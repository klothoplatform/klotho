package solution_context

import (
	"fmt"

	"github.com/klothoplatform/klotho/pkg/construct"
	"github.com/klothoplatform/klotho/pkg/engine/constraints"
	constructexpansion "github.com/klothoplatform/klotho/pkg/engine2/construct_expansion"
	"github.com/klothoplatform/klotho/pkg/engine2/operational_rule"
	"github.com/klothoplatform/klotho/pkg/engine2/path_selection"
	"github.com/klothoplatform/klotho/pkg/graph"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base2"
)

type (
	SolutionContext struct {
		dataflowGraph        *construct.ResourceGraph
		deploymentGraph      *construct.ResourceGraph
		decisions            DecisionRecords
		stack                []KV
		kb                   *knowledgebase.KnowledgeBase
		CreateResourcefromId func(id construct.ResourceId) construct.Resource
		EdgeConstraints      []constraints.EdgeConstraint
		ResourceConstraints  []constraints.ResourceConstraint
	}

	KV struct {
		key   string
		value any
	}

	DecisionRecords interface {
		// AddRecord stores each decision (the what) with the context (the why) in some datastore
		AddRecord(context []KV, decision SolveDecision)
		// FindDecision(decision SolveDecision) []KV
		// FindContext(context KV) []SolveDecision
	}

	SolveDecision interface {
		// having a private method here prevents other packages from implementing this interface
		// not necessary, but could prevent some accidental bad practices from emerging
		internal()
	}

	AddResourceDecision struct {
		Resource construct.ResourceId
	}

	RemoveResourceDecision struct {
		Resource construct.ResourceId
	}

	AddDependencyDecision struct {
		From construct.ResourceId
		To   construct.ResourceId
	}

	RemoveDependencyDecision struct {
		From construct.ResourceId
		To   construct.ResourceId
	}

	SetPropertyDecision struct {
		Resource construct.ResourceId
		Property string
		Value    any
	}
)

func NewSolutionContext() SolutionContext {
	return SolutionContext{
		dataflowGraph:   construct.NewResourceGraph(),
		deploymentGraph: construct.NewAcyclicResourceGraph(),
		decisions:       &memoryRecord{},
	}
}

func (c SolutionContext) Clone() SolutionContext {
	return SolutionContext{
		dataflowGraph:   c.dataflowGraph.Clone(),
		deploymentGraph: c.deploymentGraph.Clone(),
		decisions:       c.decisions,
	}
}

func (s SolutionContext) With(key string, value any) SolutionContext {
	return SolutionContext{
		dataflowGraph:   s.dataflowGraph,
		deploymentGraph: s.deploymentGraph,
		decisions:       s.decisions,

		stack: append(s.stack, KV{key: key, value: value}),
	}
}

func (c SolutionContext) GetDecisions() DecisionRecords {
	return c.decisions
}

// RecordDecision snapshots the current stack and records the decision
func (c SolutionContext) RecordDecision(d SolveDecision) {
	c.decisions.AddRecord(c.stack, d)
}

func (ctx SolutionContext) nodeMakeOperational(r construct.Resource) {

	ctx = ctx.With("resource", r) // add the resource to the context stack

	// handle resource constraints before to prevent unnecessary actions

	template, err := ctx.kb.GetResourceTemplate(r.Id())
	if err != nil {
		panic(err)
	}
	for _, property := range template.Properties {
		if property.OperationalStep == nil {
			continue
		}
		ruleCtx := operational_rule.OperationalRuleContext{
			Property:             &property,
			ConfigCtx:            knowledgebase.ConfigTemplateContext{DAG: ctx.dataflowGraph},
			Data:                 knowledgebase.ConfigTemplateData{Resource: r.Id()},
			Graph:                ctx,
			KB:                   ctx.kb,
			CreateResourcefromId: ctx.CreateResourcefromId,
		}
		ruleCtx.HandleOperationalStep(*property.OperationalStep)
	}
}

func (ctx SolutionContext) edgeMakeOperational(e graph.Edge[construct.Resource]) error {
	ctx = ctx.With("edge", e) // add the edge info to the decision context stack

	template := ctx.kb.GetEdgeTemplate(e.Source.Id(), e.Destination.Id())
	for _, rule := range template.OperationalRules {
		ruleCtx := operational_rule.OperationalRuleContext{
			ConfigCtx:            knowledgebase.ConfigTemplateContext{DAG: ctx.dataflowGraph},
			Data:                 knowledgebase.ConfigTemplateData{Edge: graph.Edge[construct.ResourceId]{Source: e.Source.Id(), Destination: e.Destination.Id()}},
			Graph:                ctx,
			KB:                   ctx.kb,
			CreateResourcefromId: ctx.CreateResourcefromId,
		}
		err := ruleCtx.HandleOperationalRule(rule)
		if err != nil {
			return err
		}
	}
	return nil
}

func (ctx SolutionContext) addPath(from, to construct.Resource) error {
	dep := ctx.dataflowGraph.GetDependency(from.Id(), to.Id())
	ctx.With("edge", dep)
	pathCtx := path_selection.PathSelectionContext{
		Graph:                ctx.dataflowGraph,
		KB:                   ctx.kb,
		CreateResourcefromId: ctx.CreateResourcefromId,
	}

	// Find any edge constraints around path selection

	edges, err := pathCtx.SelectPath(*dep)
	if err != nil {
		return err
	}
	if len(edges) == 1 {
		err := ctx.edgeMakeOperational(edges[0])
		if err != nil {
			return err
		}
		return nil
	} else {
		err := ctx.RemoveDependency(from.Id(), to.Id())
		if err != nil {
			return err
		}
	}
	for _, edge := range edges {
		err := ctx.edgeMakeOperational(edge)
		if err != nil {
			return err
		}
	}
	return nil
}

func (ctx SolutionContext) ConfigureResource(resource construct.Resource, configuration knowledgebase.Configuration, data knowledgebase.ConfigTemplateData) error {
	if resource == nil {
		return fmt.Errorf("resource does not exist")
	}
	configCtx := knowledgebase.ConfigTemplateContext{DAG: ctx.dataflowGraph}
	newConfig, err := configCtx.ResolveConfig(configuration, data)
	if err != nil {
		return err
	}
	err = ConfigureField(resource, newConfig.Field, newConfig.Value, false, ctx.dataflowGraph)
	if err != nil {
		return err
	}
	ctx.RecordDecision(SetPropertyDecision{
		Resource: resource.Id(),
		Property: configuration.Field,
		Value:    configuration.Value,
	})
	return nil
}

func (d AddResourceDecision) internal()      {}
func (d AddDependencyDecision) internal()    {}
func (d RemoveResourceDecision) internal()   {}
func (d RemoveDependencyDecision) internal() {}
func (d SetPropertyDecision) internal()      {}

func (ctx SolutionContext) ExpandConstruct(resource construct.Resource, constraints []constraints.ConstructConstraint) ([]SolutionContext, error) {
	expCtx := constructexpansion.ConstructExpansionContext{
		Construct:            resource,
		Kb:                   ctx.kb,
		CreateResourceFromId: ctx.CreateResourcefromId,
	}
	solutions, err := expCtx.ExpandConstruct(resource, constraints)
	if err != nil {
		return nil, err
	}
	result := []SolutionContext{}
	for _, solution := range solutions {
		newCtx := ctx.Clone()
		newCtx.With("construct", resource)
		for _, edge := range solution.Edges {
			newCtx.AddDependency(edge.Source, edge.Destination)
		}
		newCtx.ReplaceResourceId(resource.Id(), solution.DirectlyMappedResource)
		result = append(result, newCtx)
	}
	return result, nil
}

func GenerateContexts(
	kb *knowledgebase.KnowledgeBase,
	graph *construct.ResourceGraph,
	CreateResourceFromId func(id construct.ResourceId) construct.Resource) ([]SolutionContext, error) {

	ctx := NewSolutionContext()
	ctx.kb = kb

	for _, res := range graph.ListResources() {
		ctx.AddResource(res)
	}
	for _, res := range graph.ListDependencies() {
		ctx.AddDependency(res.Source, res.Destination)
	}

	solutions := []SolutionContext{ctx}
	for _, res := range graph.ListResources() {
		if res.Id().Provider == construct.AbstractConstructProvider {
			newSolutions := []SolutionContext{}
			for _, sol := range solutions {
				ctxs, err := sol.ExpandConstruct(res, []constraints.ConstructConstraint{})
				if err != nil {
					return nil, err
				}
				newSolutions = append(newSolutions, ctxs...)
			}
			solutions = newSolutions
		}
	}
	return solutions, nil
}
