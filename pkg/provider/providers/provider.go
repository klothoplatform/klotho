package providers

import (
	"fmt"

	"github.com/klothoplatform/klotho/pkg/config"
	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base"
	"github.com/klothoplatform/klotho/pkg/provider"
	"github.com/klothoplatform/klotho/pkg/provider/aws"
	awsknowledgebase "github.com/klothoplatform/klotho/pkg/provider/aws/knowledgebase"
)

func GetProvider(cfg *config.Application) (provider.Provider, error) {
	switch cfg.Provider {
	case "gcp", "azure":
		// TODO GCP and Azure is hacked to be the same as AWS so we can generate a topology diagram, but the compilation won't work.
		fallthrough
	case "aws":
		return &aws.AWS{AppName: cfg.AppName}, nil
	}

	return nil, fmt.Errorf("could not get provider: %v", cfg.Provider)
}

func GetKnowledgeBase(cfg *config.Application) (knowledgebase.EdgeKB, error) {
	switch cfg.Provider {
	case "gcp", "azure":
		// TODO GCP and Azure is hacked to be the same as AWS so we can generate a topology diagram, but the compilation won't work.
		fallthrough
	case "aws":
		return awsknowledgebase.GetAwsKnowledgeBase()
	}
	return nil, fmt.Errorf("could not get provider: %v", cfg.Provider)
}
