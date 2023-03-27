package resources

import "github.com/klothoplatform/klotho/pkg/core"

type (
	EnvironmentVariable struct {
		Resource core.Resource
		Value    string
	}

	EnvironmentVariables map[string]EnvironmentVariable
)
