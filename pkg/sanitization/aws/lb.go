package aws

import (
	"regexp"

	"github.com/klothoplatform/klotho/pkg/sanitization"
)

// EksClusterSanitizer returns a sanitized EKS Cluster when applied.
var LoadBalancerSanitizer = sanitization.NewSanitizer(
	[]sanitization.Rule{
		{
			Pattern:     regexp.MustCompile(`[^a-zA-Z\d-]`),
			Replacement: "_",
		},
		{
			Pattern:     regexp.MustCompile(`^-`),
			Replacement: "",
		},
		{
			Pattern:     regexp.MustCompile(`-$`),
			Replacement: "",
		},
		{
			Pattern:     regexp.MustCompile(`^internal-`),
			Replacement: "",
		},
	},
	32,
)

// EksNodeGroupSanitizer returns a sanitized EKS Node Group when applied.
var TargetGroupSanitizer = sanitization.NewSanitizer(
	[]sanitization.Rule{
		{
			Pattern:     regexp.MustCompile(`/[^a-zA-Z\d-]/g`),
			Replacement: "-",
		},
		{
			Pattern:     regexp.MustCompile(`/^-+/`),
			Replacement: "",
		},
		{
			Pattern:     regexp.MustCompile(`/-+$/`),
			Replacement: "",
		},
	},
	32,
)
