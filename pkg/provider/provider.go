package provider

import (
	"github.com/klothoplatform/klotho/pkg/compiler"
	"github.com/klothoplatform/klotho/pkg/config"
	"github.com/klothoplatform/klotho/pkg/core"
)

type Provider interface {
	compiler.Plugin
	GetKindTypeMappings(construct core.Construct) ([]string, bool)
	GetDefaultConfig() config.Defaults
}
