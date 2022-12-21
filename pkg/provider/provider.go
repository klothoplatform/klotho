package provider

import (
	"github.com/klothoplatform/klotho/pkg/config"
	"github.com/klothoplatform/klotho/pkg/core"
)

type Provider interface {
	core.Plugin
	GetKindTypeMappings(kind string) ([]string, bool)
	GetDefaultConfig() config.Defaults
}
