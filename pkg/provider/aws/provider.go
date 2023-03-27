package aws

import (
	"github.com/klothoplatform/klotho/pkg/config"
)

type AWS struct {
	Config                  *config.Application
	ConstructIdToResourceId map[string]string
}
