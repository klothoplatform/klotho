package engine

import (
	"errors"
	"fmt"
	"os"
	"reflect"

	"github.com/klothoplatform/klotho/pkg/core"
	"gopkg.in/yaml.v3"
	"k8s.io/utils/strings/slices"
)

type (

	// Constraint is an interface detailing different intents that can be applied to a resource graph
	Constraint interface {
		// Scope returns where on the resource graph the constraint is applied
		Scope() ConstraintScope
	}

	// BaseConstraint is the base struct for all constraints
	// BaseConstraint is used in our parsing to determine the Scope of the constraint and what go struct it corresponds to
	BaseConstraint struct {
		Scope ConstraintScope `yaml:"scope"`
	}

	// Edge is a struct that represents how we take in data about an edge in the resource graph
	Edge struct {
		Source core.ResourceId `yaml:"source"`
		Target core.ResourceId `yaml:"target"`
	}

	// ApplicationConstraint is a struct that represents constraints that can be applied on the entire resource graph
	ApplicationConstraint struct {
		Operator ConstraintOperator `yaml:"operator"`
		Node     core.ResourceId    `yaml:"node"`
		Edge     Edge               `yaml:"edge"`
	}

	// ConstructConstraint is a struct that represents constraints that can be applied on a specific construct in the resource graph
	ConstructConstraint struct {
		Operator   ConstraintOperator `yaml:"operator"`
		Target     core.ResourceId    `yaml:"target"`
		Type       string             `yaml:"type"`
		Attributes map[string]any     `yaml:"attributes"`
	}

	// EdgeConstraint is a struct that represents constraints that can be applied on a specific edge in the resource graph
	EdgeConstraint struct {
		Operator ConstraintOperator `yaml:"operator"`
		Target   Edge               `yaml:"target"`
		Node     core.ResourceId    `yaml:"node"`
	}

	// NodeConstraint is a struct that represents constraints that can be applied on a specific node in the resource graph.
	// NodeConstraints are used to control intrinsic properties of a node in the resource graph
	NodeConstraint struct {
		BaseConstraint
		Target   core.ResourceId
		Property string
		Value    any
	}

	// ConstraintScope is an enum that represents the different scopes that a constraint can be applied to
	ConstraintScope string
	// ConstraintOperator is an enum that represents the different operators that can be applied to a constraint
	ConstraintOperator string
)

const (
	ApplicationConstraintScope ConstraintScope = "application"
	ConstructConstraintScope   ConstraintScope = "construct"
	EdgeConstraintScope        ConstraintScope = "edge"
	NodeConstraintScope        ConstraintScope = "node"

	MustExistConstraintOperator      ConstraintOperator = "must_exist"
	MustContainConstraintOperator    ConstraintOperator = "must_contain"
	MustNotExistConstraintOperator   ConstraintOperator = "must_not_exist"
	MustNotContainConstraintOperator ConstraintOperator = "must_not_contain"
	AddConstraintOperator            ConstraintOperator = "add"
	EqualsConstraintOperator         ConstraintOperator = "equals"
)

func (b *ApplicationConstraint) Scope() ConstraintScope {
	return ApplicationConstraintScope
}

func (b *ConstructConstraint) Scope() ConstraintScope {
	return ConstructConstraintScope
}

func (b *EdgeConstraint) Scope() ConstraintScope {
	return EdgeConstraintScope
}

// DecodeYAMLNode is a helper function that decodes a yaml node into a struct representing different constraints
func DecodeYAMLNode[T Constraint](node *yaml.Node) (constraint T, err error) {
	constraint = reflect.New(reflect.TypeOf(constraint).Elem()).Interface().(T)
	err = extraFields(node, reflect.ValueOf(constraint))
	if err != nil {
		return constraint, err
	}
	err = node.Decode(constraint)
	return constraint, err
}

// ParseConstraintsFromFile is a helper function that parses a yaml file into a map of constraints
func ParseConstraintsFromFile(path string) (map[ConstraintScope][]Constraint, error) {
	constraints := map[ConstraintScope][]Constraint{}
	f, err := os.Open(path)
	if err != nil {
		return constraints, err
	}
	defer f.Close() // nolint:errcheck

	node := &yaml.Node{}
	err = yaml.NewDecoder(f).Decode(node)
	if err != nil {
		return constraints, err
	}

	var joinedErr error
	for _, n := range node.Content {
		for _, a := range n.Content {
			base := &BaseConstraint{}
			err = a.Decode(base)
			if err != nil {
				joinedErr = errors.Join(joinedErr, err)
				continue
			}
			fmt.Println(base)
			switch base.Scope {
			case ApplicationConstraintScope:
				appConstraint, err := DecodeYAMLNode[*ApplicationConstraint](a)
				if err != nil {
					joinedErr = errors.Join(joinedErr, err)
					continue
				}
				constraints[ApplicationConstraintScope] = append(constraints[ApplicationConstraintScope], appConstraint)
			case ConstructConstraintScope:
				constraint, err := DecodeYAMLNode[*ConstructConstraint](a)
				if err != nil {
					joinedErr = errors.Join(joinedErr, err)
					continue
				}
				constraints[ConstructConstraintScope] = append(constraints[ConstructConstraintScope], constraint)
			case EdgeConstraintScope:
				constraint, err := DecodeYAMLNode[*EdgeConstraint](a)
				if err != nil {
					joinedErr = errors.Join(joinedErr, err)
					continue
				}
				constraints[EdgeConstraintScope] = append(constraints[EdgeConstraintScope], constraint)
			}
		}
	}
	return constraints, joinedErr
}

// extraFields is a helper function that checks if there are any extra fields in a yaml node that are not in the struct
// Because you cant use the KnownFields in a nodes decode funtion we handle it ourselves
func extraFields(n *yaml.Node, object reflect.Value) error {
	knownFields := []string{}
	for i := 0; i < object.Elem().NumField(); i++ {
		knownFields = append(knownFields, object.Elem().Type().Field(i).Tag.Get("yaml"))
	}

	if n.Kind != yaml.MappingNode {
		return fmt.Errorf("got yaml node of kind %d, expected %d", n.Kind, yaml.MappingNode)
	}
	m := map[string]any{}
	if err := n.Decode(m); err != nil {
		return err
	}

	for k := range m {
		if !slices.Contains(knownFields, k) && k != "scope" {
			return fmt.Errorf("unexpected field %s", k)
		}
	}
	return nil
}
