package engine2

import (
	"errors"
	"fmt"

	construct "github.com/klothoplatform/klotho/pkg/construct2"
	"github.com/klothoplatform/klotho/pkg/engine2/constraints"
	property_eval "github.com/klothoplatform/klotho/pkg/engine2/operational_eval"
	"github.com/klothoplatform/klotho/pkg/engine2/solution_context"
	"github.com/klothoplatform/klotho/pkg/graph_addons"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base2"
	"go.uber.org/zap"
)

type (
	// solutionContext implements [solution_context.SolutionContext]
	solutionContext struct {
		KB              knowledgebase.TemplateKB
		Dataflow        construct.Graph
		Deployment      construct.Graph
		decisions       solution_context.DecisionRecords
		stack           []solution_context.KV
		mappedResources map[construct.ResourceId]construct.ResourceId
		constraints     *constraints.Constraints
		propertyEval    *property_eval.Evaluator
	}
)

func NewSolutionContext(kb knowledgebase.TemplateKB) *solutionContext {
	ctx := &solutionContext{
		KB: kb,
		Dataflow: graph_addons.LoggingGraph[construct.ResourceId, *construct.Resource]{
			Log:   zap.L().With(zap.String("graph", "dataflow")).Sugar(),
			Graph: construct.NewGraph(),
			Hash:  func(r *construct.Resource) construct.ResourceId { return r.ID },
		},
		Deployment:      construct.NewAcyclicGraph(),
		decisions:       &solution_context.MemoryRecord{},
		mappedResources: make(map[construct.ResourceId]construct.ResourceId),
	}
	ctx.propertyEval = property_eval.NewEvaluator(ctx)
	return ctx
}

func (s solutionContext) Solve() error {
	return s.propertyEval.Evaluate()
}

func (s solutionContext) With(key string, value any) solution_context.SolutionContext {
	s.stack = append(s.stack, solution_context.KV{Key: key, Value: value})
	return s
}

func (ctx solutionContext) RawView() construct.Graph {
	return solution_context.NewRawView(ctx)
}

func (ctx solutionContext) OperationalView() solution_context.OperationalView {
	return MakeOperationalView(ctx)
}

func (ctx solutionContext) DeploymentGraph() construct.Graph {
	return ctx.Deployment
}

func (ctx solutionContext) DataflowGraph() construct.Graph {
	return ctx.Dataflow
}

func (ctx solutionContext) KnowledgeBase() knowledgebase.TemplateKB {
	return ctx.KB
}

func (ctx solutionContext) Constraints() *constraints.Constraints {
	return ctx.constraints
}

func (ctx solutionContext) LoadGraph(graph construct.Graph) error {
	// Since often the input `graph` is loaded from a yaml file, we need to transform all the property values
	// to make sure they are of the correct type (eg, a string to ResourceId).
	err := knowledgebase.TransformAllPropertyValues(knowledgebase.DynamicValueContext{
		Graph:         graph,
		KnowledgeBase: ctx.KB,
	})
	if err != nil {
		return err
	}
	op := ctx.OperationalView()
	raw := ctx.RawView()
	if err := op.AddVerticesFrom(graph); err != nil {
		return err
	}

	edges, err := graph.Edges()
	if err != nil {
		return err
	}
	for _, edge := range edges {
		edgeTemplate := ctx.KB.GetEdgeTemplate(edge.Source, edge.Target)
		if edgeTemplate == nil {
			return fmt.Errorf("edge template %s -> %s not found", edge.Source, edge.Target)
		}
		if edgeTemplate.AlwaysProcess {
			if err := op.AddEdge(edge.Source, edge.Target); err != nil {
				return err
			}
		} else {
			if err := raw.AddEdge(edge.Source, edge.Target); err != nil {
				return err
			}
		}
	}

	// ensure any deployment dependencies due to properties are in place
	return construct.WalkGraph(ctx.RawView(), func(id construct.ResourceId, resource *construct.Resource, nerr error) error {
		return errors.Join(nerr, resource.WalkProperties(func(path construct.PropertyPath, werr error) error {
			prop := path.Get()
			err := solution_context.AddDeploymentDependenciesFromVal(ctx, resource, prop)
			return errors.Join(werr, err)
		}))
	})
}

func (c solutionContext) GetDecisions() solution_context.DecisionRecords {
	return c.decisions
}

// RecordDecision snapshots the current stack and records the decision
func (c solutionContext) RecordDecision(d solution_context.SolveDecision) {
	c.decisions.AddRecord(c.stack, d)
}

func (ctx solutionContext) GetMappedResource(constructId construct.ResourceId) construct.ResourceId {
	return ctx.mappedResources[constructId]
}

func (ctx solutionContext) ExpandConstruct(resource *construct.Resource) ([]solutionContext, error) {
	// TODO constructs not yet supported
	return []solutionContext{ctx}, nil
}

func (ctx solutionContext) GenerateCombinations() ([]solutionContext, error) {
	// TODO constructs not yet supported
	return []solutionContext{ctx}, nil
}
