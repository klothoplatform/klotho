package aws

import (
	"regexp"

	"github.com/klothoplatform/klotho/pkg/sanitization"
)

// ApiResourceSanitizer returns a sanitized api resource name key when applied.
var CloudfrontDistributionSanitizer = sanitization.NewSanitizer(
	[]sanitization.Rule{
		// strip any leading non alpha characters
		{
			Pattern:     regexp.MustCompile(`[^a-zA-Z0-9_-]+`),
			Replacement: "Klo$1",
		},
		{
			Pattern:     regexp.MustCompile(`-?-$`),
			Replacement: "",
		},
	}, 64)
