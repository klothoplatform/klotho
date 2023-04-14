package aws

import (
	"regexp"

	"github.com/klothoplatform/klotho/pkg/sanitization"
)

// EksClusterSanitizer returns a sanitized EKS Cluster when applied.
var EksClusterSanitizer = sanitization.NewSanitizer(
	[]sanitization.Rule{
		{
			Pattern:     regexp.MustCompile(`[^a-zA-Z\d-]`),
			Replacement: "_",
		},
		{
			Pattern:     regexp.MustCompile(`^[^a-zA-Z]+`),
			Replacement: "",
		},
	},
	100,
)

// EksNodeGroupSanitizer returns a sanitized EKS Node Group when applied.
var EksNodeGroupSanitizer = sanitization.NewSanitizer(
	[]sanitization.Rule{
		{
			Pattern:     regexp.MustCompile(`[^a-zA-Z0-9_-]`),
			Replacement: "_",
		},
		{
			Pattern:     regexp.MustCompile(`^[^a-zA-Z]+`),
			Replacement: "",
		},
	},
	64,
)

// EksFargateProfileSanitizer returns a sanitized EKS Fargate Profile when applied.
var EksFargateProfileSanitizer = sanitization.NewSanitizer(
	[]sanitization.Rule{
		{
			Pattern:     regexp.MustCompile(`[^a-zA-Z\d-]`),
			Replacement: "_",
		},
		{
			Pattern:     regexp.MustCompile(`^[^a-zA-Z]+`),
			Replacement: "",
		},
	},
	64,
)
