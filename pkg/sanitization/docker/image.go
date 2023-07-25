package docker

import (
	"github.com/klothoplatform/klotho/pkg/sanitization"
	"regexp"
)

// TagSanitizer returns a sanitized Docker image tag when applied.
var TagSanitizer = sanitization.NewSanitizer(
	[]sanitization.Rule{
		{
			// must only contain lowercase letters, numbers, hyphens, underscores, and periods
			Pattern:     regexp.MustCompile(`[^a-zA-Z0-9_.-]`),
			Lowercase:   true,
			Replacement: "",
		},
		{
			// must not start with a non-alphanumeric character
			Pattern:     regexp.MustCompile(`^[^a-z0-9]+`),
			Replacement: "",
		},
		{
			// must not end with a non-alphanumeric character
			Pattern:     regexp.MustCompile(`[^a-z0-9]+$`),
			Replacement: "",
		},
	}, 128)
