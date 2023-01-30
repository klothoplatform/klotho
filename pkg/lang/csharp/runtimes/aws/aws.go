package aws_runtime

import (
	"github.com/klothoplatform/klotho/pkg/config"
	"github.com/klothoplatform/klotho/pkg/provider/aws"
)

type AwsRuntime struct {
	TemplateConfig aws.TemplateConfig
	Cfg            *config.Application
}
