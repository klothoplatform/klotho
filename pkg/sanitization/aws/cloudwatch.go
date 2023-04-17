package aws

import (
	"regexp"

	"github.com/klothoplatform/klotho/pkg/sanitization"
)

// EnvVarKeySanitizer returns a sanitized environment key when applied.
var CloudwatchLogGroupSanitizer = sanitization.NewSanitizer(
	[]sanitization.Rule{
		{
			Pattern:     regexp.MustCompile(`[^-._/#A-Za-z\d]`),
			Replacement: "_",
		},
	}, 512)
