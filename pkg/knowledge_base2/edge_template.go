package knowledgebase2

import "github.com/klothoplatform/klotho/pkg/construct"

type (
	EdgeTemplate struct {
		Source      construct.ResourceId `yaml:"source"`
		Destination construct.ResourceId `yaml:"destination"`

		// DirectEdgeOnly signals that the edge cannot be used within constructing other paths and can only be used as a direct edge
		DirectEdgeOnly bool `yaml:"direct_edge_only"`

		// DeploymentOrderReversed is specified when the edge is in the opposite      direction of the deployment order
		DeploymentOrderReversed bool `yaml:"deployment_order_reversed"`

		// DeletetionDependent is used to specify edges which should not influence the deletion criteria of a resource
		// a true value specifies the target being deleted is dependent on the source and do not need to depend on satisfication of the deletion criteria to attempt to delete the true source of the edge.
		DeletetionDependent bool `yaml:"deletion_dependent"`

		//Reuse tells us whether we can reuse an upstream or downstream resource during path selection and node creation
		Reuse Reuse `yaml:"reuse"`

		OperationalRules []OperationalRule
	}

	// Reuse is set to represent an enum of possible reuse cases for edges. The current available options are upstream and downstream
	Reuse string
)

const (
	ReuseUpstream   Reuse = "upstream"
	ReuseDownstream Reuse = "downstream"
)
