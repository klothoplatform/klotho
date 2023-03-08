package core

import (
	"fmt"
	"strings"
)

type (
	Config struct {
		Name   string
		Secret bool
	}
)

const ConfigKind = "config"

func (p *Config) Key() ResourceKey {
	return ResourceKey{
		Name: p.Name,
		Kind: ConfigKind,
	}
}

func GenerateSecretEnvVar(id string, kind string) environmentVariable {
	return NewEnvironmentVariable(fmt.Sprintf("%s%s", strings.ToUpper(id), SECRET_NAME_SUFFIX), ConfigKind, id, string(SECRET_NAME))
}
