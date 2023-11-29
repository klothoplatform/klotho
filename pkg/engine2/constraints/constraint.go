package constraints

import (
	"errors"
	"fmt"
	"os"

	construct "github.com/klothoplatform/klotho/pkg/construct2"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base2"
	"github.com/klothoplatform/klotho/pkg/yaml_util"
	"gopkg.in/yaml.v3"
)

type (

	// Constraint is an interface detailing different intents that can be applied to a resource graph
	Constraint interface {
		// Scope returns where on the resource graph the constraint is applied
		Scope() ConstraintScope
		// IsSatisfied returns whether or not the constraint is satisfied based on the resource graph
		// For a resource graph to be valid all constraints must be satisfied
		IsSatisfied(ctx ConstraintGraph) bool
		// Validate returns whether or not the constraint is valid
		Validate() error

		String() string
	}

	ConstraintGraph interface {
		GetConstructsResource(construct.ResourceId) *construct.Resource
		GetResource(construct.ResourceId) (*construct.Resource, error)
		AllPaths(src, dst construct.ResourceId) ([][]*construct.Resource, error)
		GetClassification(construct.ResourceId) knowledgebase.Classification
	}

	// BaseConstraint is the base struct for all constraints
	// BaseConstraint is used in our parsing to determine the Scope of the constraint and what go struct it corresponds to
	BaseConstraint struct {
		Scope ConstraintScope `yaml:"scope"`
	}

	// Edge is a struct that represents how we take in data about an edge in the resource graph
	Edge struct {
		Source construct.ResourceId `yaml:"source"`
		Target construct.ResourceId `yaml:"target"`
	}

	// ConstraintScope is an enum that represents the different scopes that a constraint can be applied to
	ConstraintScope string
	// ConstraintOperator is an enum that represents the different operators that can be applied to a constraint
	ConstraintOperator string

	ConstraintList []Constraint

	Constraints struct {
		Application []ApplicationConstraint
		Construct   []ConstructConstraint
		Resources   []ResourceConstraint
		Edges       []EdgeConstraint
	}
)

const (
	ApplicationConstraintScope ConstraintScope = "application"
	ConstructConstraintScope   ConstraintScope = "construct"
	EdgeConstraintScope        ConstraintScope = "edge"
	ResourceConstraintScope    ConstraintScope = "resource"

	MustExistConstraintOperator    ConstraintOperator = "must_exist"
	MustNotExistConstraintOperator ConstraintOperator = "must_not_exist"
	AddConstraintOperator          ConstraintOperator = "add"
	RemoveConstraintOperator       ConstraintOperator = "remove"
	ReplaceConstraintOperator      ConstraintOperator = "replace"
	EqualsConstraintOperator       ConstraintOperator = "equals"
)

func (cs ConstraintList) MarshalYAML() (interface{}, error) {
	var list []yaml.Node
	for _, c := range cs {
		var n yaml.Node
		err := n.Encode(c)
		if err != nil {
			return nil, err
		}
		scope := []*yaml.Node{
			{
				Kind:  yaml.ScalarNode,
				Value: "scope",
			},
			{
				Kind:  yaml.ScalarNode,
				Value: string(c.Scope()),
			},
		}
		n.Content = append(scope, n.Content...)
		list = append(list, n)
	}
	return list, nil
}

func (cs *ConstraintList) UnmarshalYAML(node *yaml.Node) error {
	var list []yaml_util.RawNode
	err := node.Decode(&list)
	if err != nil {
		return err
	}

	*cs = make(ConstraintList, len(list))
	var errs error
	for i, raw := range list {
		var base BaseConstraint
		err = raw.Decode(&base)
		if err != nil {
			errs = errors.Join(errs, err)
			continue
		}
		var c Constraint
		switch base.Scope {

		case ApplicationConstraintScope:
			var constraint ApplicationConstraint
			err = raw.Decode(&constraint)
			c = &constraint

		case ConstructConstraintScope:
			var constraint ConstructConstraint
			err = raw.Decode(&constraint)
			c = &constraint

		case EdgeConstraintScope:
			var constraint EdgeConstraint
			err = raw.Decode(&constraint)
			c = &constraint

		case ResourceConstraintScope:
			var constraint ResourceConstraint
			err = raw.Decode(&constraint)
			c = &constraint

		default:
			err = fmt.Errorf("invalid scope %q", base.Scope)
		}
		if err != nil {
			errs = errors.Join(errs, err)
			continue
		}
		if err := c.Validate(); err != nil {
			errs = errors.Join(errs, err)
			continue
		}
		(*cs)[i] = c
	}
	return errs
}

func (list ConstraintList) ToConstraints() (Constraints, error) {
	var constraints Constraints
	for _, constraint := range list {
		switch c := constraint.(type) {
		case *ApplicationConstraint:
			constraints.Application = append(constraints.Application, *c)
		case *ConstructConstraint:
			constraints.Construct = append(constraints.Construct, *c)
		case *ResourceConstraint:
			constraints.Resources = append(constraints.Resources, *c)
		case *EdgeConstraint:
			constraints.Edges = append(constraints.Edges, *c)
		default:
			return Constraints{}, fmt.Errorf("invalid constraint type %T", constraint)
		}
	}
	return constraints, nil
}

func LoadConstraintsFromFile(path string) (Constraints, error) {
	var input struct {
		Constraints ConstraintList `yaml:"constraints"`
	}

	f, err := os.Open(path)
	if err != nil {
		return Constraints{}, err
	}
	defer f.Close() //nolint:errcheck

	err = yaml.NewDecoder(f).Decode(&input)
	if err != nil {
		return Constraints{}, err
	}

	return input.Constraints.ToConstraints()
}

// ParseConstraintsFromFile parses a yaml file into a map of constraints
//
// Future spec may include ordering of the application of constraints, but for now we assume that the order of the constraints is based on the yaml file and they cannot be grouped outside of scope
func ParseConstraintsFromFile(bytes []byte) (Constraints, error) {
	var list ConstraintList
	err := yaml.Unmarshal(bytes, &list)
	if err != nil {
		return Constraints{}, err
	}

	return list.ToConstraints()
}

func (c Constraints) ToList() ConstraintList {
	var list ConstraintList
	for i := range c.Application {
		list = append(list, &c.Application[i])
	}
	for i := range c.Construct {
		list = append(list, &c.Construct[i])
	}
	for i := range c.Resources {
		list = append(list, &c.Resources[i])
	}
	for i := range c.Edges {
		list = append(list, &c.Edges[i])
	}
	return list
}
