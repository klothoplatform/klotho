package core

import "go.uber.org/zap"

type (
	EnvironmentVariable struct {
		Name       string
		Kind       string
		ResourceID string
		Value      string
	}

	EnvironmentVariables []EnvironmentVariable
)

type EnvironmentVariableValue string

const (
	EnvironmentVariableDirective   = "environment_variables"
	ORM_ENV_VAR_NAME_SUFFIX        = "_PERSIST_ORM_CONNECTION"
	REDIS_PORT_ENV_VAR_NAME_SUFFIX = "_PERSIST_REDIS_PORT"
	REDIS_HOST_ENV_VAR_NAME_SUFFIX = "_PERSIST_REDIS_HOST"
	KLOTHO_PROXY_ENV_VAR_NAME      = "KLOTHO_PROXY_RESOURCE_NAME"
)

var (
	HOST              EnvironmentVariableValue = "host"
	PORT              EnvironmentVariableValue = "port"
	CONNECTION_STRING EnvironmentVariableValue = "connection_string"
	BUCKET_NAME       EnvironmentVariableValue = "bucket_name"
)

var InternalStorageVariable = EnvironmentVariable{
	Name:       KLOTHO_PROXY_ENV_VAR_NAME,
	Kind:       InternalKind,
	ResourceID: KlothoPayloadName,
	Value:      string(BUCKET_NAME),
}

// Add the given environment variable to the list. If a variable of the same name already exists, replace it.
func (vars *EnvironmentVariables) Add(v EnvironmentVariable) {
	if *vars == nil {
		*vars = make(EnvironmentVariables, 0)
	}
	for i, e := range *vars {
		if e.Name == v.Name {
			if e.Value != v.Value || e.ResourceID != v.ResourceID {
				zap.S().Debugf("Replacing variable %+v with %+v", e, v)
			}
			(*vars)[i] = v
			return
		}
	}
	*vars = append(*vars, v)
}

// AddAll is a convenience over `Add` to add many variables.
func (vars *EnvironmentVariables) AddAll(vs EnvironmentVariables) {
	for _, v := range vs {
		vars.Add(v)
	}
}
