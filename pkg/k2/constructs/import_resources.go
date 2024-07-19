package constructs

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/dominikbraun/graph"
	"github.com/klothoplatform/klotho/pkg/construct"
	"github.com/klothoplatform/klotho/pkg/engine/solution"
	stateconverter "github.com/klothoplatform/klotho/pkg/infra/state_reader/state_converter"
	statetemplate "github.com/klothoplatform/klotho/pkg/infra/state_reader/state_template"
	"github.com/klothoplatform/klotho/pkg/k2/constructs/template/property"
	"github.com/klothoplatform/klotho/pkg/k2/model"
	"github.com/klothoplatform/klotho/pkg/logging"
)

func (ce *ConstructEvaluator) importFrom(ctx context.Context, o InfraOwner, ic *Construct) error {
	log := logging.GetLogger(ctx).Sugar()
	initGraph := o.GetInitialGraph()
	sol := ic.Solution
	stackState, hasState := ce.stackStateManager.ConstructStackState[ic.URN]

	// NOTE(gg): using topo sort to get all resources, order doesn't matter
	resourceIds, err := construct.TopologicalSort(sol.DataflowGraph())
	if err != nil {
		return fmt.Errorf("could not get resources from %s solution: %w", ic.URN, err)
	}
	resources := make(map[construct.ResourceId]*construct.Resource)
	for _, rId := range resourceIds {
		var liveStateRes *construct.Resource
		if hasState {
			if state, ok := stackState.Resources[rId]; ok {
				liveStateRes, err = ce.stateConverter.ConvertResource(stateconverter.Resource{
					Urn:     string(state.URN),
					Type:    string(state.Type),
					Outputs: state.Outputs,
				})
				if err != nil {
					return fmt.Errorf("could not convert state for %s.%s: %w", ic.URN, rId, err)
				}
				log.Debugf("Imported %s from state", rId)
			}
		}
		originalRes, err := sol.DataflowGraph().Vertex(rId)
		if err != nil {
			return fmt.Errorf("could not get resource %s.%s from solution: %w", ic.URN, rId, err)
		}

		tmpl, err := sol.KnowledgeBase().GetResourceTemplate(rId)
		if err != nil {
			return fmt.Errorf("could not get resource template %s.%s: %w", ic.URN, rId, err)
		}

		props := make(construct.Properties)
		for k, v := range originalRes.Properties {
			props[k] = v
		}
		hasImportId := false
		// set a fake import id, otherwise index.ts will have things like
		//   Type.get("name", <no value>)
		for k, prop := range tmpl.Properties {
			if prop.Details().Required && prop.Details().DeployTime {
				if liveStateRes == nil {
					if ce.DryRun > 0 {
						props[k] = fmt.Sprintf("preview(id=%s)", rId)
						hasImportId = true
						continue
					} else {
						return fmt.Errorf("could not get live state resource %s (%s)", ic.URN, rId)
					}
				}
				liveIdProp, err := liveStateRes.GetProperty(k)
				if err != nil {
					return fmt.Errorf("could not get property %s for %s: %w", k, rId, err)
				}
				props[k] = liveIdProp
				hasImportId = true
			}
		}
		if !hasImportId {
			continue
		}

		res := &construct.Resource{
			ID:         originalRes.ID,
			Properties: props,
			Imported:   true,
		}

		log.Debugf("Imported %s from solution", rId)

		if err := initGraph.AddVertex(res); err != nil {
			return fmt.Errorf("could not create imported resource %s from %s: %w", rId, ic.URN, err)
		}
		resources[rId] = res
	}
	err = filterImportProperties(resources)
	if err != nil {
		return fmt.Errorf("could not filter import properties for %s: %w", ic.URN, err)
	}

	edges, err := sol.DataflowGraph().Edges()
	if err != nil {
		return fmt.Errorf("could not get edges from %s solution: %w", ic.URN, err)
	}
	for _, e := range edges {
		err := initGraph.AddEdge(e.Source, e.Target, func(ep *graph.EdgeProperties) {
			ep.Data = e.Properties.Data
		})
		switch {
		case err == nil:
			log.Debugf("Imported edge %s -> %s from solution", e.Source, e.Target)

		case errors.Is(err, graph.ErrVertexNotFound):
			log.Debugf("Skipping import edge %s -> %s from solution", e.Source, e.Target)

		default:
			return fmt.Errorf("could not create imported edge %s -> %s from %s: %w", e.Source, e.Target, ic.URN, err)
		}
	}

	return nil
}

// filterImportProperties filters out any references to resources that were skipped from importing.
func filterImportProperties(resources map[construct.ResourceId]*construct.Resource) error {
	var errs []error
	clearProp := func(id construct.ResourceId, path construct.PropertyPath) {
		if err := path.Remove(nil); err != nil {
			errs = append(errs,
				fmt.Errorf("error clearing %s: %w", construct.PropertyRef{Resource: id, Property: path.String()}, err),
			)
		}
	}
	for id, r := range resources {
		_ = r.WalkProperties(func(path construct.PropertyPath, _ error) error {
			v, ok := path.Get()
			if !ok {
				return nil
			}
			switch v := v.(type) {
			case construct.ResourceId:
				if _, ok := resources[v]; !ok {
					clearProp(id, path)
				}

			case construct.PropertyRef:
				if _, ok := resources[v.Resource]; !ok {
					clearProp(id, path)
				}
			}
			return nil
		})
	}
	return errors.Join(errs...)
}

// importResourcesFromInputs imports resources from the construct-type inputs of the provided [InfraOwner], o.
// It returns an error if the input value is does not represent a valid construct
// or if importing the resources fails.
func (ce *ConstructEvaluator) importResourcesFromInputs(o InfraOwner, ctx context.Context) error {
	return o.ForEachInput(func(i property.Property) error {
		// if the input is a construct, import the resources from it
		if !strings.HasPrefix(i.Type(), "construct") {
			return nil
		}

		resolvedInput, err := o.GetInputValue(i.Details().Path)
		if err != nil {
			return fmt.Errorf("could not get input %s: %w", i.Details().Path, err)
		}

		cURN, ok := resolvedInput.(model.URN)
		if !ok || !cURN.IsResource() || cURN.Type != "construct" {
			return fmt.Errorf("input %s is not a construct URN", i.Details().Path)
		}

		c, ok := ce.Constructs.Get(cURN)
		if !ok {
			return fmt.Errorf("could not find construct %s", cURN)
		}

		if err := ce.importFrom(ctx, o, c); err != nil {
			return fmt.Errorf("could not import resources from %s: %w", cURN, err)
		}
		return nil
	})
}

func (ce *ConstructEvaluator) importBindingToResources(ctx context.Context, b *Binding) error {
	return ce.importFrom(ctx, b, b.To)
}

func (ce *ConstructEvaluator) RegisterOutputValues(urn model.URN, outputs map[string]any) {
	if c, ok := ce.Constructs.Get(urn); ok {
		c.Outputs = outputs
	}
}

func (ce *ConstructEvaluator) AddSolution(urn model.URN, sol solution.Solution) {
	// panic is fine here if urn isn't in map
	// will only happen in programmer error cases
	c, _ := ce.Constructs.Get(urn)
	c.Solution = sol
}

func loadStateConverter() (stateconverter.StateConverter, error) {
	templates, err := statetemplate.LoadStateTemplates("pulumi")
	if err != nil {
		return nil, err
	}
	return stateconverter.NewStateConverter("pulumi", templates), nil
}
