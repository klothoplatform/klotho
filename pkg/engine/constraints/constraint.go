package constraints

import (
	"errors"
	"fmt"
	"os"
	"reflect"

	"github.com/klothoplatform/klotho/pkg/collectionutil"
	"github.com/klothoplatform/klotho/pkg/construct"
	"github.com/klothoplatform/klotho/pkg/engine/classification"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base"
	"gopkg.in/yaml.v3"
	"k8s.io/utils/strings/slices"
)

type (

	// Constraint is an interface detailing different intents that can be applied to a resource graph
	Constraint interface {
		// Scope returns where on the resource graph the constraint is applied
		Scope() ConstraintScope
		// IsSatisfied returns whether or not the constraint is satisfied based on the resource graph
		// For a resource graph to be valid all constraints must be satisfied
		IsSatisfied(dag *construct.ResourceGraph, kb knowledgebase.EdgeKB, mappedConstructResources map[construct.ResourceId][]construct.Resource, classifier *classification.ClassificationDocument) bool
		// Validate returns whether or not the constraint is valid
		Validate() error

		String() string
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
)

const (
	ApplicationConstraintScope ConstraintScope = "application"
	ConstructConstraintScope   ConstraintScope = "construct"
	EdgeConstraintScope        ConstraintScope = "edge"
	ResourceConstraintScope    ConstraintScope = "resource"

	MustExistConstraintOperator      ConstraintOperator = "must_exist"
	MustNotExistConstraintOperator   ConstraintOperator = "must_not_exist"
	MustContainConstraintOperator    ConstraintOperator = "must_contain"
	MustNotContainConstraintOperator ConstraintOperator = "must_not_contain"
	AddConstraintOperator            ConstraintOperator = "add"
	RemoveConstraintOperator         ConstraintOperator = "remove"
	ReplaceConstraintOperator        ConstraintOperator = "replace"
	EqualsConstraintOperator         ConstraintOperator = "equals"
)

// DecodeYAMLNode is a helper function that decodes a yaml node into a struct representing different constraints
func DecodeYAMLNode[T interface {
	Constraint
	*I
}, I any](node *yaml.Node) (constraint T, err error) {
	constraint = new(I)
	err = extraFields(node, reflect.ValueOf(constraint))
	if err != nil {
		return constraint, err
	}
	err = node.Decode(constraint)
	return constraint, err
}

func LoadConstraintsFromFile(path string) (map[ConstraintScope][]Constraint, error) {

	type Input struct {
		Constraints []any `yaml:"constraints"`
	}

	input := Input{}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close() // nolint:errcheck

	err = yaml.NewDecoder(f).Decode(&input)
	if err != nil {
		return nil, err
	}

	bytesArr, err := yaml.Marshal(input.Constraints)
	if err != nil {
		return nil, err
	}
	return ParseConstraintsFromFile(bytesArr)
}

// ParseConstraintsFromFile parses a yaml file into a map of constraints
//
// Future spec may include ordering of the application of constraints, but for now we assume that the order of the constraints is based on the yaml file and they cannot be grouped outside of scope
func ParseConstraintsFromFile(bytes []byte) (map[ConstraintScope][]Constraint, error) {
	constraints := map[ConstraintScope][]Constraint{}

	node := &yaml.Node{}
	err := yaml.Unmarshal(bytes, node)
	if err != nil {
		return nil, err
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
			switch base.Scope {
			case ApplicationConstraintScope:
				constraint, err := DecodeYAMLNode[*ApplicationConstraint](a)
				if err != nil {
					joinedErr = errors.Join(joinedErr, err)
					continue
				}
				validOperators := []ConstraintOperator{AddConstraintOperator, RemoveConstraintOperator, ReplaceConstraintOperator}
				if !collectionutil.Contains(validOperators, constraint.Operator) {
					joinedErr = errors.Join(joinedErr, fmt.Errorf("invalid operator %s for application constraint", constraint.Operator))
					continue
				}
				constraints[ApplicationConstraintScope] = append(constraints[ApplicationConstraintScope], constraint)
			case ConstructConstraintScope:
				constraint, err := DecodeYAMLNode[*ConstructConstraint](a)
				if err != nil {
					joinedErr = errors.Join(joinedErr, err)
					continue
				}
				validOperators := []ConstraintOperator{EqualsConstraintOperator}
				if !collectionutil.Contains(validOperators, constraint.Operator) {
					joinedErr = errors.Join(joinedErr, fmt.Errorf("invalid operator %s for construct constraint", constraint.Operator))
					continue
				}
				constraints[ConstructConstraintScope] = append(constraints[ConstructConstraintScope], constraint)
			case EdgeConstraintScope:
				constraint, err := DecodeYAMLNode[*EdgeConstraint](a)
				if err != nil {
					joinedErr = errors.Join(joinedErr, err)
					continue
				}
				validOperators := []ConstraintOperator{MustContainConstraintOperator, MustNotContainConstraintOperator, MustExistConstraintOperator, MustNotExistConstraintOperator}
				if !collectionutil.Contains(validOperators, constraint.Operator) {
					joinedErr = errors.Join(joinedErr, fmt.Errorf("invalid operator %s for application edge", constraint.Operator))
					continue
				}
				constraints[EdgeConstraintScope] = append(constraints[EdgeConstraintScope], constraint)
			case ResourceConstraintScope:
				constraint, err := DecodeYAMLNode[*ResourceConstraint](a)
				if err != nil {
					joinedErr = errors.Join(joinedErr, err)
					continue
				}
				validOperators := []ConstraintOperator{AddConstraintOperator}
				if !collectionutil.Contains(validOperators, constraint.Operator) {
					joinedErr = errors.Join(joinedErr, fmt.Errorf("invalid operator %s for resource constraint", constraint.Operator))
					continue
				}
				constraints[ResourceConstraintScope] = append(constraints[ResourceConstraintScope], constraint)
			}
		}
	}
	return constraints, joinedErr
}

// extraFields is a helper function that checks if there are any extra fields in a yaml node that are not in the struct
// Because you cant use the KnownFields in a nodes decode funtion we handle it ourselves
func extraFields(n *yaml.Node, object reflect.Value) error {
	knownFields := []string{"scope"}
	for i := 0; i < object.Elem().NumField(); i++ {
		fieldName := object.Elem().Type().Field(i).Name
		yamlTag := object.Elem().Type().Field(i).Tag.Get("yaml")
		if yamlTag != "" {
			fieldName = yamlTag
		}
		knownFields = append(knownFields, fieldName)
	}

	if n.Kind != yaml.MappingNode {
		return fmt.Errorf("got yaml node of kind %d, expected %d", n.Kind, yaml.MappingNode)
	}
	m := map[string]any{}
	if err := n.Decode(m); err != nil {
		return err
	}

	for k := range m {
		if !slices.Contains(knownFields, k) {
			return fmt.Errorf("unexpected field %s", k)
		}
	}
	return nil
}
