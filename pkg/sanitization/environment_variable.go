package sanitization

import (
	"regexp"
)

var EnvVarKeySanitizer = NewSanitizer(
	// strip any leading non alpha characters
	Rule{
		Pattern:     regexp.MustCompile(`^[^a-zA-Z]+`),
		Replacement: "",
	},
	// replace "-" or whitespace with "_"
	Rule{
		Pattern:     regexp.MustCompile(`[-\s]+`),
		Replacement: "_",
	},
	// strip any other invalid characters
	Rule{
		Pattern:     regexp.MustCompile(`[^a-zA-Z0-9_]+`),
		Replacement: "",
	})
