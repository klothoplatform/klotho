package aws

import (
	"github.com/klothoplatform/klotho/pkg/config"
	"github.com/klothoplatform/klotho/pkg/provider/aws/resources"
)

type AWS struct {
	Config                  *config.Application
	ConstructIdToResourceId map[string]string
	PolicyGenerator         *resources.PolicyGenerator
}
