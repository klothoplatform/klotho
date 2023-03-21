package core

import (
	"fmt"
	"strings"
)

type (
	Config struct {
		AnnotationKey
		Secret bool
	}
)

const ConfigKind = "config"

func (p *Config) Provenance() AnnotationKey {
	return p.AnnotationKey
}

func (p *Config) Id() string {
	return p.AnnotationKey.ToString()
}

func GenerateSecretEnvVar(cfg *Config) environmentVariable {
	return NewEnvironmentVariable(fmt.Sprintf("%s%s", strings.ToUpper(cfg.ID), SECRET_NAME_SUFFIX), cfg, string(SECRET_NAME))
}
