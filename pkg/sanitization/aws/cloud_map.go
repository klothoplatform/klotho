package aws

import (
	"regexp"

	"github.com/klothoplatform/klotho/pkg/sanitization"
)

// PrivateDnsNamespaceSanitizer returns a sanitized private dns namespace when applied.
var PrivateDnsNamespaceSanitizer = sanitization.NewSanitizer(
	[]sanitization.Rule{
		// strip any leading non alpha characters
		{
			Pattern:     regexp.MustCompile(`^[!-~]+$`),
			Replacement: "_",
		},
	}, 64)
