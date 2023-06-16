package runtimes

import (
	"fmt"

	"github.com/klothoplatform/klotho/pkg/config"
	"github.com/klothoplatform/klotho/pkg/lang/python"
	"github.com/klothoplatform/klotho/pkg/lang/python/aws_runtime"
)

func GetRuntime(cfg *config.Application) (python.Runtime, error) {
	switch cfg.Provider {
	case "gcp", "azure":
		// TODO GCP and Azure is hacked to be the same as AWS so we can generate a topology diagram, but the compilation won't work.
		fallthrough
	case "aws":
		return &aws_runtime.AwsRuntime{
			Cfg: cfg,
		}, nil
	}

	return nil, fmt.Errorf("could not get Python runtime for provider: %v", cfg.Provider)
}
