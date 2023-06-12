package core

import (
	"fmt"
	"strings"

	"github.com/klothoplatform/klotho/pkg/annotation"
)

type (
	Config struct {
		Name   string
		Secret bool
	}
)

const CONFIG_TYPE = "config"

func (p *Config) Id() ResourceId {
	return ResourceId{
		Provider: AbstractConstructProvider,
		Type:     CONFIG_TYPE,
		Name:     p.Name,
	}
}

func (p *Config) AnnotationCapability() string {
	return annotation.ConfigCapability
}

func GenerateSecretEnvVar(cfg *Config) environmentVariable {
	return NewEnvironmentVariable(fmt.Sprintf("%s%s", strings.ToUpper(cfg.Name), SECRET_NAME_SUFFIX), cfg, string(SECRET_NAME))
}
