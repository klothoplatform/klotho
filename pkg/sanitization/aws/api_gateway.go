package aws

import (
	"regexp"

	"github.com/klothoplatform/klotho/pkg/sanitization"
)

// RestApiSanitizer returns a sanitized api name when applied.
var RestApiSanitizer = sanitization.NewSanitizer(
	[]sanitization.Rule{
		// strip any leading non alpha characters
		{
			Pattern:     regexp.MustCompile(`[^a-zA-Z0-9_-]+`),
			Replacement: "-",
		},
	}, 64)

// ApiResourceSanitizer returns a sanitized api resource name key when applied.
var ApiResourceSanitizer = sanitization.NewSanitizer(
	[]sanitization.Rule{
		// strip any leading non alpha characters
		{
			Pattern:     regexp.MustCompile(`[^a-zA-Z0-9_-]+`),
			Replacement: "-",
		},
	}, 64)
