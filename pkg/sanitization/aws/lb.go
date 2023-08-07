package aws

import (
	"regexp"

	"github.com/klothoplatform/klotho/pkg/sanitization"
)

// LoadBalancerSanitizer returns a load balancer name when applied.
var LoadBalancerSanitizer = sanitization.NewSanitizer(
	[]sanitization.Rule{
		{
			Pattern:     regexp.MustCompile(`[^a-zA-Z\d-]`),
			Replacement: "-",
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

// TargetGroupSanitizer returns a sanitized target group name when applied.
var TargetGroupSanitizer = sanitization.NewSanitizer(
	[]sanitization.Rule{
		{
			Pattern:     regexp.MustCompile(`[^a-zA-Z\d-]`),
			Replacement: "-",
		},
		{
			Pattern:     regexp.MustCompile(`^-+`),
			Replacement: "",
		},
		{
			Pattern:     regexp.MustCompile(`-+$`),
			Replacement: "",
		},
	},
	32,
)
