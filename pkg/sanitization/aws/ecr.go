package aws

import (
	"github.com/klothoplatform/klotho/pkg/sanitization"
	"regexp"
)

// EcrRepositorySanitizer returns a sanitized ECR Repository name when applied.
var EcrRepositorySanitizer = sanitization.NewSanitizer(
	[]sanitization.Rule{
		{
			Pattern:     regexp.MustCompile(`[^a-zA-Z0-9-_/.]`),
			Lowercase:   true,
			Replacement: "",
		},
		{
			Pattern:     regexp.MustCompile(`[/._\-]{2,}`),
			Replacement: "-",
		},
		{
			Pattern:     regexp.MustCompile(`\s+`),
			Replacement: "-",
		},
		{
			Pattern:     regexp.MustCompile(`^[^a-z0-9]+`),
			Replacement: "",
		},
		{
			Pattern:     regexp.MustCompile(`[^a-z0-9]+$`),
			Replacement: "",
		},
	}, 256)
