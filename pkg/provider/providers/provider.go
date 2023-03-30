package providers

import (
	"fmt"

	"github.com/klothoplatform/klotho/pkg/config"
	"github.com/klothoplatform/klotho/pkg/provider"
	"github.com/klothoplatform/klotho/pkg/provider/aws"
	"github.com/klothoplatform/klotho/pkg/provider/aws/resources"
)

func GetProvider(cfg *config.Application) (provider.Provider, error) {
	switch cfg.Provider {
	case "gcp", "azure":
		// TODO GCP and Azure is hacked to be the same as AWS so we can generate a topology diagram, but the compilation won't work.
		fallthrough
	case "aws":
		return &aws.AWS{
			Config:                  cfg,
			ConstructIdToResourceId: make(map[string]string),
			PolicyGenerator:         resources.NewPolicyGenerator(),
		}, nil
	}

	return nil, fmt.Errorf("could not get provider: %v", cfg.Provider)
}
