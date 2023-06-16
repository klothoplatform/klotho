package runtimes

import (
	"fmt"

	"github.com/klothoplatform/klotho/pkg/config"
	"github.com/klothoplatform/klotho/pkg/lang/csharp"
	aws_runtime "github.com/klothoplatform/klotho/pkg/lang/csharp/runtimes/aws"
)

func GetRuntime(cfg *config.Application) (csharp.Runtime, error) {
	switch cfg.Provider {
	case "gcp", "azure":
		// TODO GCP and Azure is hacked to be the same as AWS so we can generate a topology diagram, but the compilation won't work.
		fallthrough
	case "aws":
		return &aws_runtime.AwsRuntime{
			Cfg: cfg,
		}, nil
	}

	return nil, fmt.Errorf("could not get C# runtime for provider: %v", cfg.Provider)
}
