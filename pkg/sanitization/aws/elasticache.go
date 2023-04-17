package aws

import (
	"regexp"

	"github.com/klothoplatform/klotho/pkg/sanitization"
)

// EksClusterSanitizer returns a sanitized EKS Cluster when applied.
var ElasticacheClusterSanitizer = sanitization.NewSanitizer(
	[]sanitization.Rule{
		{
			Pattern:     regexp.MustCompile(`[^a-zA-Z0-9-]`),
			Replacement: "-",
		},
		{
			Pattern:     regexp.MustCompile(`--+`),
			Replacement: "-",
		},
		{
			Pattern:     regexp.MustCompile(`-+$`),
			Replacement: "",
		},
		{
			Pattern:     regexp.MustCompile(`^[^a-zA-Z]+`),
			Replacement: "",
		},
	},
	100,
)
