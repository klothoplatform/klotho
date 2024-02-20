package constraints

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	construct "github.com/klothoplatform/klotho/pkg/construct"
)

type (
	// ResourceConstraint is a struct that represents constraints that can be applied on a specific node in the resource graph.
	// ResourceConstraints are used to control intrinsic properties of a resource in the resource graph
	//
	// Example
	//
	// To specify a constraint detailing a property of a resource in yaml
	//
	// - scope: resource
	// operator: equals
	// target: aws:rds_instance:my_instance
	// property: InstanceClass
	// value: db.t3.micro
	//
	// The end result of this should be that the the rds instance's InstanceClass property should be set to db.t3.micro
	ResourceConstraint struct {
		Operator ConstraintOperator   `yaml:"operator" json:"operator"`
		Target   construct.ResourceId `yaml:"target" json:"target"`
		Property string               `yaml:"property" json:"property"`
		Value    any                  `yaml:"value" json:"value"`
	}
)

func (constraint *ResourceConstraint) Scope() ConstraintScope {
	return ResourceConstraintScope
}

func (constraint *ResourceConstraint) IsSatisfied(ctx ConstraintGraph) bool {
	switch constraint.Operator {
	case EqualsConstraintOperator:
		res, _ := ctx.GetResource(constraint.Target)
		if res == nil {
			return false
		}
		strct := reflect.ValueOf(res)
		for strct.Kind() == reflect.Ptr {
			strct = strct.Elem()
		}
		val := strct.FieldByName(constraint.Property)
		if !val.IsValid() {
			// Try to find the field by its json or yaml tag (especially to handle case [upper/lower] [Pascal/snake])
			// Replicated from resource_configuration.go#parseFieldName so there's no dependency
			for i := 0; i < strct.NumField(); i++ {
				field := strct.Type().Field(i)
				if constraint.Property == strings.ToLower(field.Name) {
					// When YAML marshalling fields that don't have a tag, they're just lower cased
					// so this condition should catch those.
					val = strct.Field(i)
					break
				}
				tag := strings.Split(field.Tag.Get("json"), ",")[0]
				if constraint.Property == tag {
					val = strct.Field(i)
					break
				}
				tag = strings.Split(field.Tag.Get("yaml"), ",")[0]
				if constraint.Property == tag {
					val = strct.Field(i)
					break
				}
			}
			if !val.IsValid() {
				return false
			}
		}
		return val.Interface() == constraint.Value

	case AddConstraintOperator:
		res, _ := ctx.GetResource(constraint.Target)
		if res == nil {
			return false
		}
		parent := reflect.ValueOf(res).Elem()
		val := parent.FieldByName(constraint.Property)
		if !val.IsValid() {
			return false
		}
		switch val.Kind() {
		case reflect.Slice, reflect.Array:
			for i := 0; i < val.Len(); i++ {
				if val.Index(i).Interface() == constraint.Value {
					return true
				}
			}
			return false
		}
	}
	return true
}

func (constraint *ResourceConstraint) Validate() error {
	if constraint.Target.IsAbstractResource() {
		return errors.New("node constraint cannot be applied to an abstract construct")
	}
	if constraint.Property == "" {
		return errors.New("node constraint must have a property defined")
	}
	return nil
}

func (constraint *ResourceConstraint) String() string {
	return fmt.Sprintf("ResourceConstraint: %s %s %s %v", constraint.Target, constraint.Property, constraint.Operator, constraint.Value)
}
