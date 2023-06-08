package aws

import (
	"regexp"

	"github.com/klothoplatform/klotho/pkg/sanitization"
)

// KmsKeySanitizer returns a sanitized kms key name when applied.
var KinesisStreamSanitizer = sanitization.NewSanitizer(
	[]sanitization.Rule{
		// strip any characters not matching [a-zA-Z0-9-_/]
		{
			Pattern:     regexp.MustCompile(`[^a-zA-Z0-9-._]+`),
			Replacement: "",
		},
	}, 128)
