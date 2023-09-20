package constructexpansion

import (
	"errors"
	"fmt"

	"github.com/klothoplatform/klotho/pkg/collectionutil"
	"github.com/klothoplatform/klotho/pkg/construct"
	"github.com/klothoplatform/klotho/pkg/engine/constraints"
	"github.com/klothoplatform/klotho/pkg/graph"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base2"
	"go.uber.org/zap"
)

type (
	ExpansionSet struct {
		Construct  construct.Construct
		Attributes []string
	}

	ExpansionSolution struct {
		Edges                  []graph.Edge[construct.Resource]
		DirectlyMappedResource construct.Resource
	}

	ConstructExpansionContext struct {
		Construct            construct.Resource
		Kb                   *knowledgebase.KnowledgeBase
		CreateResourceFromId func(id construct.ResourceId) construct.Resource
	}
)

// ExpandConstructs expands all constructs in the working state using the engines provider
//
// The resources that result from the expanded constructs are written to the engines resource graph
// All dependencies are copied over to the resource graph
// If a dependency in the working state included a construct, the engine copies the dependency to all directly linked resources
func (ctx *ConstructExpansionContext) ExpandConstruct(res construct.Resource, constraints []constraints.ConstructConstraint) ([]ExpansionSolution, error) {
	if res.Id().Provider != construct.AbstractConstructProvider {
		return nil, fmt.Errorf("unable to expand construct %s, resource is not an abstract construct", res.Id())
	}
	zap.S().Debugf("Expanding construct %s", res.Id())
	construct, ok := res.(construct.Construct)
	if !ok {
		return nil, fmt.Errorf("unable to cast base construct %s to construct while expanding construct", res.Id())
	}

	constructType := ""
	attributes := make(map[string]any)
	for _, constructConstraint := range constraints {
		if constructConstraint.Target == construct.Id() {
			if constructType != "" && constructType != constructConstraint.Type {
				return nil, fmt.Errorf("unable to expand construct %s, conflicting types in constraints", res.Id())
			}
			for k, v := range constructConstraint.Attributes {
				if val, ok := attributes[k]; ok {
					if v != val {
						return nil, fmt.Errorf("unable to expand construct %s, attribute %s has conflicting values", res.Id(), k)
					}
				}
				attributes[k] = v
			}
		}
	}

	for k, v := range construct.Attributes() {
		if val, ok := attributes[k]; ok {
			if v != val {
				return nil, fmt.Errorf("unable to expand construct %s, attribute %s has conflicting values", res.Id(), k)
			}
		}
		attributes[k] = v
	}
	return ctx.expandConstruct(constructType, attributes, construct)
}

func (ctx *ConstructExpansionContext) expandConstruct(constructType string, attributes map[string]any, c construct.Construct) ([]ExpansionSolution, error) {
	var baseResource construct.Resource
	for _, res := range ctx.Kb.ListResources() {
		if res.Id().Type == constructType {
			baseResource = ctx.CreateResourceFromId(res.Id())
		}
	}
	expansionSet := ExpansionSet{Construct: c}
	for attribute := range attributes {
		expansionSet.Attributes = append(expansionSet.Attributes, attribute)
	}
	return ctx.findPossibleExpansions(expansionSet, baseResource)
}

func (ctx *ConstructExpansionContext) findPossibleExpansions(expansionSet ExpansionSet, baseResource construct.Resource) ([]ExpansionSolution, error) {
	var possibleExpansions []ExpansionSolution
	var joinedErr error
	for _, res := range ctx.Kb.ListResources() {
		if baseResource != nil && res.Id().Type != baseResource.Id().Type {
			continue
		}
		classifications := res.Classification
		if !collectionutil.Contains(classifications.Is, string(expansionSet.Construct.Functionality())) {
			continue
		}
		unsatisfiedAttributes := []string{}
		for _, ms := range expansionSet.Attributes {
			if !collectionutil.Contains(classifications.Is, ms) {
				unsatisfiedAttributes = append(unsatisfiedAttributes, ms)
			}
		}
		baseRes := ctx.CreateResourceFromId(construct.ResourceId{Type: res.Id().Type, Name: expansionSet.Construct.Id().Name, Provider: res.Id().Provider})
		expansions, err := ctx.findExpansions(unsatisfiedAttributes, []graph.Edge[construct.Resource]{}, baseRes, expansionSet.Construct.Functionality())
		if err != nil {
			joinedErr = errors.Join(joinedErr, err)
			continue
		}
		for _, expansion := range expansions {
			possibleExpansions = append(possibleExpansions, ExpansionSolution{Edges: expansion, DirectlyMappedResource: baseRes})
		}
	}
	return possibleExpansions, joinedErr
}

func (ctx *ConstructExpansionContext) findExpansions(attributes []string, edges []graph.Edge[construct.Resource], baseResource construct.Resource, functionality construct.Functionality) ([][]graph.Edge[construct.Resource], error) {
	if len(attributes) == 0 {
		return [][]graph.Edge[construct.Resource]{}, nil
	}
	var result [][]graph.Edge[construct.Resource]
	for _, attribute := range attributes {
		for _, res := range ctx.Kb.ListResources() {
			if res.Id().QualifiedTypeName() == baseResource.Id().QualifiedTypeName() {
				continue
			}
			if ctx.Kb.HasFunctionalPath(baseResource.Id(), res.Id()) {
				if res.GivesAttributeForFunctionality(attribute, functionality) {
					resource := ctx.CreateResourceFromId(construct.ResourceId{Type: res.Id().Type, Name: baseResource.Id().Name, Provider: res.Id().Provider})
					edges = append(edges, graph.Edge[construct.Resource]{Source: baseResource, Destination: resource})
					unsatisfiedAttributes := []string{}
					for _, ms := range attributes {
						if ms != attribute {
							unsatisfiedAttributes = append(unsatisfiedAttributes, ms)
						}
					}

					expansions, err := ctx.findExpansions(unsatisfiedAttributes, edges, baseResource, functionality)
					if err != nil {
						return nil, err
					}
					result = append(result, expansions...)
				}
			}

		}
	}
	if len(result) == 0 {
		return nil, fmt.Errorf("no expansions found for attributes %v", attributes)
	}
	return result, nil
}
