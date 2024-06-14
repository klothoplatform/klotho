package engine

import (
	"errors"
	"fmt"

	"github.com/klothoplatform/klotho/pkg/multierr"

	construct "github.com/klothoplatform/klotho/pkg/construct"
	"github.com/klothoplatform/klotho/pkg/engine/constraints"
	property_eval "github.com/klothoplatform/klotho/pkg/engine/operational_eval"
	"github.com/klothoplatform/klotho/pkg/engine/solution"
	"github.com/klothoplatform/klotho/pkg/graph_addons"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledgebase"
	"go.uber.org/zap"
)

type (
	// solutionContext implements [solution_context.SolutionContext]
	solutionContext struct {
		solution.DecisionRecords

		KB           knowledgebase.TemplateKB
		Dataflow     construct.Graph
		Deployment   construct.Graph
		constraints  *constraints.Constraints
		propertyEval *property_eval.Evaluator
		globalTag    string
		outputs      map[string]construct.Output
	}
)

func NewSolutionContext(kb knowledgebase.TemplateKB, globalTag string, constraints *constraints.Constraints) *solutionContext {
	ctx := &solutionContext{
		KB: kb,
		Dataflow: graph_addons.LoggingGraph[construct.ResourceId, *construct.Resource]{
			Log:   zap.L().With(zap.String("graph", "dataflow")).Sugar(),
			Graph: construct.NewGraph(),
			Hash:  func(r *construct.Resource) construct.ResourceId { return r.ID },
		},
		Deployment:  construct.NewAcyclicGraph(),
		constraints: constraints,
		globalTag:   globalTag,
		outputs:     make(map[string]construct.Output),
	}
	ctx.propertyEval = property_eval.NewEvaluator(ctx)
	return ctx
}

func (s *solutionContext) Solve() error {
	err := s.propertyEval.Evaluate()
	if err != nil {
		return err
	}
	return s.captureOutputs()
}

func (s *solutionContext) RawView() construct.Graph {
	return solution.NewRawView(s)
}

func (s *solutionContext) OperationalView() solution.OperationalView {
	return (*MakeOperationalView)(s)
}

func (s *solutionContext) DeploymentGraph() construct.Graph {
	return s.Deployment
}

func (s *solutionContext) DataflowGraph() construct.Graph {
	return s.Dataflow
}

func (s *solutionContext) KnowledgeBase() knowledgebase.TemplateKB {
	return s.KB
}

func (s *solutionContext) Constraints() *constraints.Constraints {
	return s.constraints
}

func (s *solutionContext) LoadGraph(graph construct.Graph) error {
	// Since often the input `graph` is loaded from a yaml file, we need to transform all the property values
	// to make sure they are of the correct type (eg, a string to ResourceId).
	err := knowledgebase.TransformAllPropertyValues(knowledgebase.DynamicValueContext{
		Graph:         graph,
		KnowledgeBase: s.KB,
	})
	if err != nil {
		return err
	}
	op := s.OperationalView()
	raw := s.RawView()
	if err := op.AddVerticesFrom(graph); err != nil {
		return err
	}

	edges, err := graph.Edges()
	if err != nil {
		return err
	}
	for _, edge := range edges {
		edgeTemplate := s.KB.GetEdgeTemplate(edge.Source, edge.Target)
		src, err := graph.Vertex(edge.Source)
		if err != nil {
			return err
		}
		dst, err := graph.Vertex(edge.Target)
		if err != nil {
			return err
		}
		if src.Imported && dst.Imported {
			if err := raw.AddEdge(edge.Source, edge.Target); err != nil {
				return err
			}
			continue
		}
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
	return construct.WalkGraph(s.RawView(), func(id construct.ResourceId, resource *construct.Resource, nerr error) error {
		return errors.Join(nerr, resource.WalkProperties(func(path construct.PropertyPath, werr error) error {
			prop := path.Get()
			err := solution.AddDeploymentDependenciesFromVal(s, resource, prop)
			return errors.Join(werr, err)
		}))
	})
}

func (s *solutionContext) GlobalTag() string {
	return s.globalTag
}

func (s *solutionContext) captureOutputs() error {
	outputConstraints := s.Constraints().Outputs
	var err multierr.Error
	for _, outputConstraint := range outputConstraints {
		if outputConstraint.Ref.Resource.IsZero() {
			s.outputs[outputConstraint.Name] = construct.Output{
				Value: outputConstraint.Value,
			}
			continue
		}

		if _, err2 := s.Dataflow.Vertex(outputConstraint.Ref.Resource); err2 != nil {
			err.Append(err2)
			continue
		}
		s.outputs[outputConstraint.Name] = construct.Output{
			Ref: outputConstraint.Ref,
		}
	}
	return err.ErrOrNil()
}

func (s *solutionContext) Outputs() map[string]construct.Output {
	return s.outputs
}
