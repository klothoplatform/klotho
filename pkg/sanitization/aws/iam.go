package aws

import (
	"regexp"

	"github.com/klothoplatform/klotho/pkg/sanitization"
)

// EnvVarKeySanitizer returns a sanitized environment key when applied.
var IamRoleSanitizer = sanitization.NewSanitizer(
	[]sanitization.Rule{
		// replace "-" or whitespace with "_"
		{
			Pattern:     regexp.MustCompile(`[^\w+=,.@-]`),
			Replacement: "_",
		},
	}, 64)

var IamPolicySanitizer = sanitization.NewSanitizer(
	[]sanitization.Rule{
		// replace "-" or whitespace with "_"
		{
			Pattern:     regexp.MustCompile(`[^\w+=,.@-]`),
			Replacement: "_",
		},
	}, 64)
