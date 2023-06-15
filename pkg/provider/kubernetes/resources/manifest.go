package resources

import (
	"github.com/klothoplatform/klotho/pkg/core"
)

type Manifest interface {
	core.Resource
	Kind() string
	Path() string
	OutputYAML() core.File
}
