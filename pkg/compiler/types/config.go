package types

import (
	"fmt"
	"strings"

	"github.com/klothoplatform/klotho/pkg/annotation"
	"github.com/klothoplatform/klotho/pkg/construct"
)

type (
	Config struct {
		Name   string
		Secret bool
	}
)

const CONFIG_TYPE = "config"

func (p *Config) Id() construct.ResourceId {
	return construct.ResourceId{
		Provider: construct.AbstractConstructProvider,
		Type:     CONFIG_TYPE,
		Name:     p.Name,
	}
}

func (p *Config) AnnotationCapability() string {
	return annotation.ConfigCapability
}

func (p *Config) Functionality() construct.Functionality {
	return construct.Storage
}

func (p *Config) Attributes() map[string]any {
	if p.Secret {
		return map[string]any{
			"secret": nil,
		}
	}
	return map[string]any{}
}

func GenerateSecretEnvVar(cfg *Config) environmentVariable {
	return NewEnvironmentVariable(fmt.Sprintf("%s%s", strings.ToUpper(cfg.Name), SECRET_NAME_SUFFIX), cfg, string(SECRET_NAME))
}
