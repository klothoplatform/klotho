package engine2

import (
	construct "github.com/klothoplatform/klotho/pkg/construct2"
	"github.com/klothoplatform/klotho/pkg/engine2/constraints"
	"github.com/klothoplatform/klotho/pkg/engine2/solution_context"
	"github.com/klothoplatform/klotho/pkg/graph_store"
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
		constraints     constraints.Constraints
	}
)

func NewSolutionContext(kb knowledgebase.TemplateKB) solutionContext {
	return solutionContext{
		KB: kb,
		Dataflow: graph_store.LoggingGraph[construct.ResourceId, *construct.Resource]{
			Log:   zap.S(),
			Graph: construct.NewGraph(),
			Hash:  func(r *construct.Resource) construct.ResourceId { return r.ID },
		},
		Deployment:      construct.NewAcyclicGraph(),
		decisions:       &solution_context.MemoryRecord{},
		mappedResources: make(map[construct.ResourceId]construct.ResourceId),
	}
}
func (c solutionContext) Clone(keepStack bool) (solutionContext, error) {
	dfClone, err := c.Dataflow.Clone()
	if err != nil {
		return solutionContext{}, err
	}
	deployClone, err := c.Deployment.Clone()
	if err != nil {
		return solutionContext{}, err
	}
	newCtx := solutionContext{
		KB:              c.KB,
		Dataflow:        dfClone,
		Deployment:      deployClone,
		decisions:       c.decisions,
		mappedResources: make(map[construct.ResourceId]construct.ResourceId),
	}
	for k, v := range c.mappedResources {
		newCtx.mappedResources[k] = v
	}
	if keepStack {
		newCtx.stack = c.stack
	}
	return newCtx, nil
}

func (s solutionContext) With(key string, value any) solution_context.SolutionContext {
	return solutionContext{
		Dataflow:        s.Dataflow,
		Deployment:      s.Deployment,
		decisions:       s.decisions,
		KB:              s.KB,
		mappedResources: s.mappedResources,
		constraints:     s.constraints,

		stack: append(s.stack, solution_context.KV{Key: key, Value: value}),
	}
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
	return &ctx.constraints
}

func (ctx solutionContext) LoadGraph(graph construct.Graph) error {
	// Since often the input `graph` is loaded from a yaml file, we need to transform all the property values
	// to make sure they are of the correct type (eg, a string to ResourceId).
	err := knowledgebase.TransformAllPropertyValues(knowledgebase.DynamicValueContext{DAG: graph, KB: ctx.KB})
	if err != nil {
		return err
	}
	raw := ctx.RawView()
	if err := raw.AddVerticesFrom(graph); err != nil {
		return err
	}
	return raw.AddEdgesFrom(graph)
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
