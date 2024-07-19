package property

import (
	"bytes"

	"github.com/klothoplatform/klotho/pkg/construct"
)

type (
	// ExecutionContext defines the methods to execute a go template and decode the result into a value
	ExecutionContext interface {
		// ExecuteUnmarshal executes the template tmpl using data as input and unmarshals the value into v
		ExecuteUnmarshal(tmpl string, data any, v any) error
		// Unmarshal unmarshals the template result into a value
		Unmarshal(data *bytes.Buffer, v any) error
	}

	Property interface {
		// SetProperty sets the value of the property on the properties
		SetProperty(properties construct.Properties, value any) error
		// AppendProperty appends the value to the property on the properties
		AppendProperty(properties construct.Properties, value any) error
		// RemoveProperty removes the value from the property on the properties
		RemoveProperty(properties construct.Properties, value any) error
		// Details returns the property details for the property
		Details() *PropertyDetails
		// Clone returns a clone of the property
		Clone() Property

		// Type returns the string representation of the property's type, as it should appear in a template
		Type() string
		// GetDefaultValue returns the default value for the property,
		// pertaining to the specific data being passed in for execution
		GetDefaultValue(ctx ExecutionContext, data any) (any, error)
		// Validate ensures the value is valid for the property to `Set` (not `Append` for collection types)
		// and returns an error if it is not
		Validate(properties construct.Properties, value any) error
		// SubProperties returns the sub properties of the input, if any.
		// This is used for inputs that are complex structures, such as lists, sets, or maps
		SubProperties() PropertyMap
		// Parse parses a given value to ensure it is the correct type for the property.
		// If the given value cannot be converted to the respective property type an error is returned.
		// The returned value will always be the correct type for the property
		Parse(value any, ctx ExecutionContext, data any) (any, error)
		// ZeroValue returns the zero value for the property type
		ZeroValue() any
		// Contains returns true if the value contains the given value
		Contains(value any, contains any) bool
	}

	MapProperty interface {
		// Key returns the property representing the keys of the map
		Key() Property
		// Value returns the property representing the values of the map
		Value() Property
	}

	CollectionProperty interface {
		// Item returns the structure of the items within the collection
		Item() Property
	}

	Properties interface {
		Clone() Properties
		ForEach(c construct.Properties, f func(p Property) error) error
		Get(key string) (Property, bool)
		Set(key string, value Property)
		Remove(key string)
		AsMap() map[string]Property
	}
)
