package aws

import (
	"regexp"

	"github.com/klothoplatform/klotho/pkg/sanitization"
)

// EnvVarKeySanitizer returns a sanitized environment key when applied.
var S3BucketSanitizer = sanitization.NewSanitizer(
	[]sanitization.Rule{
		// replace "-" or whitespace with "_"
		{
			Pattern:     regexp.MustCompile(`[^a-z0-9.-]`),
			Replacement: "-",
		},
	},
	52, // We know that we will prepend account id onto the names here, so we want to shorten this further until we do that in the iac level
)

// EnvVarKeySanitizer returns a sanitized environment key when applied.
var S3ObjectSanitizer = sanitization.NewSanitizer(
	[]sanitization.Rule{
		{
			Pattern:     regexp.MustCompile(`^.$`),
			Replacement: "-",
		},
	},
	0,
)
