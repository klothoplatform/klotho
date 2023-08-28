package resources

import (
	"github.com/klothoplatform/klotho/pkg/construct"
)

func ListAll() []construct.Resource {
	return []construct.Resource{
		&DockerImage{},
	}
}
