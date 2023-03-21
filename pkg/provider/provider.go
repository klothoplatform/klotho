package provider

import (
	"github.com/klothoplatform/klotho/pkg/compiler"
	"github.com/klothoplatform/klotho/pkg/config"
)

type Provider interface {
	compiler.Plugin
	GetKindTypeMappings(kind string) ([]string, bool)
	GetDefaultConfig() config.Defaults
}
