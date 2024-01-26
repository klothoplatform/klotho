package knowledgebase2

import (
	"fmt"

	construct "github.com/klothoplatform/klotho/pkg/construct2"
	"gopkg.in/yaml.v3"
)

type (
	EdgeTemplate struct {
		Source construct.ResourceId `yaml:"source"`
		Target construct.ResourceId `yaml:"target"`

		// AlwaysProcess signals that the edge should always be processed even if the source and target exist in the input graph
		// currently we dont check edges for operational rules if they previously existed and this flag is set to false
		AlwaysProcess bool `yaml:"always_process"`

		// DirectEdgeOnly signals that the edge cannot be used within constructing other paths
		// and can only be used as a direct edge
		DirectEdgeOnly bool `yaml:"direct_edge_only"`

		// DeploymentOrderReversed is specified when the edge is in the opposite direction of the deployment order
		DeploymentOrderReversed bool `yaml:"deployment_order_reversed"`

		// DeletetionDependent is used to specify edges which should not influence the deletion criteria of a resource
		// a true value specifies the target being deleted is dependent on the source and do not need to depend on
		// satisfication of the deletion criteria to attempt to delete the true source of the edge.
		DeletionDependent bool `yaml:"deletion_dependent"`

		// Unique see type [Unique]
		Unique Unique `yaml:"unique"`

		OperationalRules []OperationalRule `yaml:"operational_rules"`

		EdgeWeightMultiplier float32 `yaml:"edge_weight_multiplier"`

		Classification []string `yaml:"classification"`

		NoIac bool `json:"no_iac" yaml:"no_iac"`
	}

	MultiEdgeTemplate struct {
		Resource construct.ResourceId   `yaml:"resource"`
		Sources  []construct.ResourceId `yaml:"sources"`
		Targets  []construct.ResourceId `yaml:"targets"`

		// DirectEdgeOnly signals that the edge cannot be used within constructing other paths
		// and can only be used as a direct edge
		DirectEdgeOnly bool `yaml:"direct_edge_only"`

		// DeploymentOrderReversed is specified when the edge is in the opposite direction of the deployment order
		DeploymentOrderReversed bool `yaml:"deployment_order_reversed"`

		// DeletetionDependent is used to specify edges which should not influence the deletion criteria of a resource
		// a true value specifies the target being deleted is dependent on the source and do not need to depend on
		// satisfication of the deletion criteria to attempt to delete the true source of the edge.
		DeletionDependent bool `yaml:"deletion_dependent"`

		// Unique see type [Unique]
		Unique Unique `yaml:"unique"`

		OperationalRules []OperationalRule `yaml:"operational_rules"`

		EdgeWeightMultiplier float32 `yaml:"edge_weight_multiplier"`

		Classification []string `yaml:"classification"`

		NoIac bool `json:"no_iac" yaml:"no_iac"`
	}

	// Unique is used to specify whether the source or target of an edge must only have a single edge of this type
	// - Source=false & Target=false (default) indicates that S->T is a many-to-many relationship
	//   (for examples, Lambda -> DynamoDB)
	// - Source=true & Target=false indicates that S->T is a one-to-many relationship
	//   (for examples, SQS -> Event Source Mapping)
	// - Source=false & Target=true indicates that S->T is a many-to-one relationship
	//   (for examples, Event Source Mapping -> Lambda)
	// - Source=true & Target=true indicates that S->T is a one-to-one relationship
	//   (for examples, RDS Proxy -> Proxy Target Group)
	Unique struct {
		// Source indicates whether the source must only have a single edge of this type.
		Source bool `yaml:"source"`
		// Target indicates whether the target must only have a single edge of this type.
		Target bool `yaml:"target"`
	}
)

func EdgeTemplatesFromMulti(multi MultiEdgeTemplate) []EdgeTemplate {
	var templates []EdgeTemplate
	for _, source := range multi.Sources {
		templates = append(templates, EdgeTemplate{
			Source:                  source,
			Target:                  multi.Resource,
			DirectEdgeOnly:          multi.DirectEdgeOnly,
			DeploymentOrderReversed: multi.DeploymentOrderReversed,
			DeletionDependent:       multi.DeletionDependent,
			Unique:                  multi.Unique,
			OperationalRules:        multi.OperationalRules,
			EdgeWeightMultiplier:    multi.EdgeWeightMultiplier,
			Classification:          multi.Classification,
		})
	}
	for _, target := range multi.Targets {
		templates = append(templates, EdgeTemplate{
			Source:                  multi.Resource,
			Target:                  target,
			DirectEdgeOnly:          multi.DirectEdgeOnly,
			DeploymentOrderReversed: multi.DeploymentOrderReversed,
			DeletionDependent:       multi.DeletionDependent,
			Unique:                  multi.Unique,
			OperationalRules:        multi.OperationalRules,
			EdgeWeightMultiplier:    multi.EdgeWeightMultiplier,
			Classification:          multi.Classification,
		})
	}
	return templates
}

func (u *Unique) UnmarshalYAML(n *yaml.Node) error {
	type helper Unique
	var h helper
	if err := n.Decode(&h); err == nil {
		*u = Unique(h)
		return nil
	}

	var str string
	if err := n.Decode(&str); err == nil {
		switch str {
		case "one_to_one", "one-to-one":
			u.Source = true
			u.Target = true
		case "one_to_many", "one-to-many":
			u.Source = true
			u.Target = false
		case "many_to_one", "many-to-one":
			u.Source = false
			u.Target = true
		case "many_to_many", "many-to-many":
			u.Source = false
			u.Target = false
		default:
			return fmt.Errorf("invalid 'unique' string: %s", str)
		}
		return nil
	}

	var b bool
	if err := n.Decode(&b); err == nil {
		u.Source = b
		u.Target = b
		return nil
	}

	return fmt.Errorf("could not decode 'unique' field")
}

// CanAdd returns whether the edge source -> target can be added based on the uniqueness rules.
// - "many-to-many" always returns true
// - "one-to-many" returns true if the target does not have any edges that match the source type
// - "many-to-one" returns true if the source does not have any edges that match the target type
// - "one-to-one" returns true if neither the source nor the target have any edges that match the other type
func (u Unique) CanAdd(edges []construct.Edge, source, target construct.ResourceId) bool {
	if !u.Source && !u.Target {
		return true
	}

	if u.Source { // one-to-many or one-to-one
		sourceSel := construct.ResourceId{Provider: source.Provider, Type: source.Type}
		for _, e := range edges {
			if e.Target != target {
				continue
			}
			// Make sure that the target doesn't have any edges that match the source type
			if sourceSel.Matches(e.Source) && e.Source != source {
				return false
			}
		}
	}
	if u.Target { // many-to-one or one-to-one
		targetSel := construct.ResourceId{Provider: target.Provider, Type: target.Type}
		for _, e := range edges {
			if e.Source != source {
				continue
			}
			// Make sure that the source doesn't have any edges that match the target type
			if targetSel.Matches(e.Target) && e.Target != target {
				return false
			}
		}
	}
	return true
}
