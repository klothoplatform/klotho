package runtimes

import (
	"fmt"

	"github.com/klothoplatform/klotho/pkg/config"
	"github.com/klothoplatform/klotho/pkg/lang/javascript"
	"github.com/klothoplatform/klotho/pkg/lang/javascript/aws_runtime"
)

func GetRuntime(cfg *config.Application) (javascript.Runtime, error) {
	switch cfg.Provider {
	case "gcp", "azure":
		// TODO GCP and Azure is hacked to be the same as AWS so we can generate a topology diagram, but the compilation won't work.
		fallthrough
	case "aws":
		return &aws_runtime.AwsRuntime{
			Config: cfg,
		}, nil
	}

	return nil, fmt.Errorf("could not get JS runtime for provider: %v", cfg.Provider)
}
