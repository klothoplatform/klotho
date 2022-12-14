package providers

import (
	"fmt"

	"github.com/klothoplatform/klotho/pkg/config"
	"github.com/klothoplatform/klotho/pkg/provider"
	"github.com/klothoplatform/klotho/pkg/provider/aws"
)

func GetProvider(cfg *config.Application) (provider.Provider, error) {
	switch cfg.Provider {
	case "gcp", "azure":
		// TODO GCP and Azure is hacked to be the same as AWS so we can generate a topology diagram, but the compilation won't work.
		fallthrough
	case "aws":
		return &aws.AWS{
			Config: cfg,
		}, nil
	}

	return nil, fmt.Errorf("could not get provider: %v", cfg.Provider)
}
