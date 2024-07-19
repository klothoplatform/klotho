package property

type (
	// PropertyDetails defines the common details of a property
	PropertyDetails struct {
		Name string `json:"name" yaml:"name"`
		// DefaultValue has to be any because it may be a template and it may be a value of the correct type
		Namespace bool `yaml:"namespace"`
		// Required defines if the property is required
		Required bool `json:"required" yaml:"required"`
		// ConfigurationDisabled defines if the property is allowed to be configured by the user
		ConfigurationDisabled bool `json:"configuration_disabled" yaml:"configuration_disabled"`
		// OperationalRule defines a rule that is executed at runtime to determine the value of the property
		//OperationalRule *PropertyRule `json:"operational_rule" yaml:"operational_rule"`
		// Description is a description of the property. This is not used in the engine solving,
		// but is metadata returned by the `ListResourceTypes` CLI command.
		Description string `json:"description" yaml:"description"`
		// Path is the path to the property in the resource
		Path string `json:"-" yaml:"-"`
	}
)
