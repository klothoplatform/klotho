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

func (p *Config) Provenance() AnnotationKey {
	return p.AnnotationKey
}

func (p *Config) Id() ResourceId {
	return ConstructId(p.AnnotationKey).ToRid()
}

func GenerateSecretEnvVar(cfg *Config) environmentVariable {
	return NewEnvironmentVariable(fmt.Sprintf("%s%s", strings.ToUpper(cfg.ID), SECRET_NAME_SUFFIX), cfg, string(SECRET_NAME))
}
