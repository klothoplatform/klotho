package resources

import (
	"github.com/klothoplatform/klotho/pkg/core"
)

func ListAll() []core.Resource {
	return []core.Resource{
		&DockerImage{},
	}
}
