package runtimes

import (
	"fmt"

	"github.com/klothoplatform/klotho/pkg/config"
	"github.com/klothoplatform/klotho/pkg/lang/golang"
	"github.com/klothoplatform/klotho/pkg/lang/golang/aws_runtime"
)

func GetRuntime(cfg *config.Application) (golang.Runtime, error) {
	switch cfg.Provider {
	case "gcp", "azure":
		// TODO GCP and Azure is hacked to be the same as AWS so we can generate a topology diagram, but the compilation won't work.
		fallthrough
	case "aws":
		return &aws_runtime.AwsRuntime{
			Cfg: cfg,
		}, nil
	}

	return nil, fmt.Errorf("could not get Go runtime for provider: %v", cfg.Provider)
}
