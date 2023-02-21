package core

type (
	EnvironmentVariable struct {
		Name       string
		Kind       string
		ResourceID string
		Value      string
	}
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
