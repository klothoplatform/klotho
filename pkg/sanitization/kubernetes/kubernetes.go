package aws

import (
	"regexp"

	"github.com/klothoplatform/klotho/pkg/sanitization"
)

// VpcSanitizer returns a sanitized vpc name when applied.
var MetadataNameSanitizer = sanitization.NewSanitizer(
	[]sanitization.Rule{
		// strip any leading non alpha characters
		{
			Pattern:     regexp.MustCompile(`^[^a-z]+`),
			Replacement: "",
		},
		// strip any ending non alpha characters
		{
			Pattern:     regexp.MustCompile(`[^a-z]$`),
			Replacement: "",
		},
		// strip any other invalid characters
		{
			Pattern:     regexp.MustCompile(`[^a-z0-9-.]+`),
			Replacement: "",
		},
	}, 253)

// VpcSanitizer returns a sanitized vpc name when applied.
var HelmValueSanitizer = sanitization.NewSanitizer(
	[]sanitization.Rule{
		{
			Pattern:     regexp.MustCompile(`[^a-zA-Z0-9]+`),
			Replacement: "",
		},
	}, 100)
