package core

type (
	EnvironmentVariable struct {
		Name       string
		Kind       string
		ResourceID string
		Value      string
	}
)

const (
	EnvironmentVariableDirective = "environment_variables"
)
