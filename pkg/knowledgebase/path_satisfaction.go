package knowledgebase

import (
	"fmt"
	"strings"

	construct "github.com/klothoplatform/klotho/pkg/construct"
	"gopkg.in/yaml.v3"
)

type (
	PathSatisfaction struct {
		AsTarget            []PathSatisfactionRoute `json:"as_target" yaml:"as_target"`
		AsSource            []PathSatisfactionRoute `json:"as_source" yaml:"as_source"`
		DenyClassifications []string                `yaml:"deny_classifications"`
	}

	PathSatisfactionRoute struct {
		Classification    string                            `json:"classification" yaml:"classification"`
		PropertyReference string                            `json:"property_reference" yaml:"property_reference"`
		Validity          PathSatisfactionValidityOperation `json:"validity" yaml:"validity"`
		Script            string                            `json:"script" yaml:"script"`
	}

	PathSatisfactionValidityOperation string
)

const (
	DownstreamOperation PathSatisfactionValidityOperation = "downstream"
)

func (p *PathSatisfactionRoute) UnmarshalYAML(n *yaml.Node) error {
	type h PathSatisfactionRoute
	var p2 h
	err := n.Decode(&p2)
	if err != nil {
		routeString := n.Value
		routeParts := strings.Split(routeString, "#")
		p2.Classification = routeParts[0]
		if len(routeParts) > 1 {
			p2.PropertyReference = strings.Join(routeParts[1:], "#")
		}
		*p = PathSatisfactionRoute(p2)
		return nil
	}
	p2.Validity = PathSatisfactionValidityOperation(strings.ToLower(string(p2.Validity)))
	*p = PathSatisfactionRoute(p2)
	if p.PropertyReference != "" && p.Script != "" {
		return fmt.Errorf("path satisfaction route cannot have both property reference and script")
	}
	return nil
}

func (kb *KnowledgeBase) GetPathSatisfactionsFromEdge(source, target construct.ResourceId) ([]EdgePathSatisfaction, error) {
	srcTempalte, err := kb.GetResourceTemplate(source)
	if err != nil {
		return nil, err
	}
	targetTemplate, err := kb.GetResourceTemplate(target)
	if err != nil {
		return nil, err
	}
	pathSatisfications := []EdgePathSatisfaction{}
	trgtsAdded := map[PathSatisfactionRoute]struct{}{}

	for _, src := range srcTempalte.PathSatisfaction.AsSource {
		srcClassificationHandled := false
		for _, trgt := range targetTemplate.PathSatisfaction.AsTarget {
			if trgt.Classification == src.Classification {
				useSrc := src
				useTrgt := trgt
				pathSatisfications = append(pathSatisfications, EdgePathSatisfaction{
					Classification: src.Classification,
					Source:         useSrc,
					Target:         useTrgt,
				})
				srcClassificationHandled = true
				trgtsAdded[trgt] = struct{}{}
			}
		}
		if !srcClassificationHandled {
			useSrc := src
			pathSatisfications = append(pathSatisfications, EdgePathSatisfaction{
				Classification: src.Classification,
				Source:         useSrc,
			})
		}
	}
	for _, trgt := range targetTemplate.PathSatisfaction.AsTarget {
		if _, ok := trgtsAdded[trgt]; !ok {
			useTrgt := trgt
			pathSatisfications = append(pathSatisfications, EdgePathSatisfaction{
				Classification: trgt.Classification,
				Target:         useTrgt,
			})
		}
	}
	if len(pathSatisfications) == 0 {
		pathSatisfications = append(pathSatisfications, EdgePathSatisfaction{})
	}
	return pathSatisfications, nil
}

func (v PathSatisfactionRoute) PropertyReferenceChangesBoundary() bool {
	if v.Validity != "" {
		return false
	}
	return v.PropertyReference != ""
}
